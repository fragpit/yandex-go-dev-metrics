package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
	"github.com/go-resty/resty/v2"
	"golang.org/x/sync/errgroup"
)

const (
	clientPostTimeout = 5 * time.Second
)

// Reporter is responsible for reporting metrics to the server.
type Reporter struct {
	l    *slog.Logger
	repo repository.Repository

	serverURL string
	secretKey []byte
	rateLimit int
	cryptoKey string
}

// NewReporter creates a new Reporter instance.
func NewReporter(
	l *slog.Logger,
	st repository.Repository,
	serverURL string,
	secretKey []byte,
	rateLimit int,
	cryptoKey string,
) *Reporter {
	return &Reporter{
		l:         l,
		repo:      st,
		serverURL: serverURL,
		secretKey: secretKey,
		rateLimit: rateLimit,
		cryptoKey: cryptoKey,
	}
}

// RunReporter starts the reporting process at the specified interval.
func (r *Reporter) RunReporter(
	ctx context.Context,
	interval time.Duration,
) error {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			m, err := r.repo.GetMetrics(ctx)
			if err != nil {
				r.l.Error("failed to get metrics",
					slog.Any("error", err))
				return fmt.Errorf("failed to get metrics: %w", err)
			}

			if err := r.repo.Reset(); err != nil {
				r.l.Error("failed to reset map",
					slog.Any("error", err))
				return fmt.Errorf("failed to reset map: %w", err)
			}

			if err := r.reportMetrics(ctx, m); err != nil {
				r.l.Error("failed to report metrics",
					slog.Any("error", err))
				return fmt.Errorf("failed to report metrics: %w", err)
			}
		}
	}
}

func (r *Reporter) reportMetrics(
	ctx context.Context,
	m map[string]model.Metric,
) error {
	r.l.Info("starting reporter")

	if len(m) == 0 {
		r.l.Info("no metrics to report")
		return nil
	}

	updateURL := r.serverURL + "/updates/"

	parsedURL, err := url.Parse(r.serverURL)
	if err != nil {
		return fmt.Errorf("failed to parse server url: %w", err)
	}

	ip, err := localIPFor(parsedURL.Hostname())
	if err != nil {
		return fmt.Errorf(
			"failed to get source ip for provided server hostname: %w",
			err,
		)
	}

	client := resty.New()
	client.
		SetTimeout(clientPostTimeout).
		SetRetryCount(3).
		SetRetryWaitTime(1*time.Second).
		SetRetryMaxWaitTime(5*time.Second).
		SetHeader("Content-Type", "application/json").
		OnBeforeRequest(addRealIPHeader(ip.String()))

	if len(r.secretKey) > 0 {
		client.OnBeforeRequest(checksumRequestMiddleware(r.secretKey))
	}

	if len(r.cryptoKey) > 0 {
		client.OnBeforeRequest(encryptRequestMiddleware(r.cryptoKey))
	} else {
		client.OnBeforeRequest(gzipRequestMiddleware())
	}

	var metrics []*model.Metrics

	for _, metric := range m {
		m := metric.ToJSON()
		metrics = append(metrics, m)
	}

	const batchSize = 10

	numBatches := (len(metrics) + batchSize - 1) / batchSize
	batches := make([][]*model.Metrics, 0, numBatches)
	for start := 0; start < len(metrics); start += batchSize {
		end := start + batchSize
		if end > len(metrics) {
			end = len(metrics)
		}
		batches = append(batches, metrics[start:end])
	}

	jobs := make(chan []*model.Metrics, len(batches))
	for _, b := range batches {
		jobs <- b
	}
	close(jobs)

	worker := func(id int, jobs <-chan []*model.Metrics) error {
		for batch := range jobs {
			r.l.Info(
				"reporting batch",
				slog.Int("worker_num", id),
				slog.Int("batch_size", len(batch)),
			)

			data, err := json.Marshal(batch)
			if err != nil {
				r.l.Error(
					"error marshaling metrics",
					slog.Any("error", err),
				)
				return fmt.Errorf("error marshaling metrics: %w", err)
			}

			resp, err := client.R().
				SetContext(ctx).
				SetBody(data).
				Post(updateURL)
			if err != nil {
				r.l.Error(
					"error reporting metrics",
					slog.Any("error", err),
				)
				return fmt.Errorf("error reporting metrics: %w", err)
			}

			if resp.StatusCode() != http.StatusOK {
				r.l.Error(
					"non-ok status code",
					slog.Int("status_code", resp.StatusCode()),
				)
				return fmt.Errorf("non-ok status code: %d",
					resp.StatusCode())
			}
		}

		return nil
	}

	g, ctx := errgroup.WithContext(ctx)
	for w := 1; w <= r.rateLimit; w++ {
		workerID := w
		g.Go(func() error {
			return worker(workerID, jobs)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("worker failed: %w", err)
	}
	return nil
}

func gzipRequestMiddleware() resty.RequestMiddleware {
	return func(c *resty.Client, req *resty.Request) error {
		if req.Body == nil {
			return nil
		}

		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)

		bodyBytes, ok := req.Body.([]byte)
		if !ok {
			data, err := json.Marshal(req.Body)
			if err != nil {
				return fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyBytes = data
		}

		if _, err := zw.Write(bodyBytes); err != nil {
			return err
		}

		if err := zw.Close(); err != nil {
			return err
		}

		req.SetBody(buf.Bytes())
		req.SetHeader("Content-Encoding", "gzip")

		return nil
	}
}

func checksumRequestMiddleware(key []byte) resty.RequestMiddleware {
	return func(c *resty.Client, req *resty.Request) error {
		mac := hmac.New(sha256.New, key)

		bodyBytes, ok := req.Body.([]byte)
		if !ok {
			data, err := json.Marshal(req.Body)
			if err != nil {
				return fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyBytes = data
		}

		mac.Write(bodyBytes)
		sum := mac.Sum(nil)
		sumEncoded := base64.RawStdEncoding.EncodeToString(sum)

		req.SetHeader("HashSHA256", sumEncoded)

		return nil
	}
}

func encryptRequestMiddleware(keyPath string) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		publicKey, err := readKey(keyPath)
		if err != nil {
			return fmt.Errorf("failed to read key: %w", err)
		}

		encBody, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, r.Body.([]byte))
		if err != nil {
			return fmt.Errorf("failed to encrypt body: %w", err)
		}

		r.Body = encBody
		r.SetHeader("X-Encrypted", "rsa")

		return nil
	}
}

func readKey(keyPath string) (*rsa.PublicKey, error) {
	publicKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	publicKeyPEM, _ := pem.Decode(publicKeyBytes)
	if publicKeyPEM == nil {
		return nil, errors.New("invalid key format")
	}

	publicKey, err := x509.ParsePKIXPublicKey(publicKeyPEM.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}

	return publicKey.(*rsa.PublicKey), nil
}

func addRealIPHeader(ip string) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		r.SetHeader("X-Real-IP", ip)

		return nil
	}
}

func localIPFor(serverHost string) (net.IP, error) {
	conn, err := net.Dial("udp4", net.JoinHostPort(serverHost, "80"))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("unexpected local addr type %T", conn.LocalAddr())
	}

	return udpAddr.IP, nil
}
