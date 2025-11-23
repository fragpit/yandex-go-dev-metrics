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
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	"github.com/go-resty/resty/v2"
	"golang.org/x/sync/errgroup"
)

var _ Transport = (*RESTTransport)(nil)

type RESTTransport struct {
	serverURL string
	secretKey []byte
	rateLimit int
	cryptoKey string
}

func (t *RESTTransport) SendMetrics(
	ctx context.Context,
	m map[string]model.Metric,
) error {
	slog.Info("starting reporter")

	if len(m) == 0 {
		slog.Info("no metrics to report")
		return nil
	}

	updateURL := t.serverURL + "/updates/"

	parsedURL, err := url.Parse(t.serverURL)
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

	if len(t.secretKey) > 0 {
		client.OnBeforeRequest(checksumRequestMiddleware(t.secretKey))
	}

	if len(t.cryptoKey) > 0 {
		client.OnBeforeRequest(encryptRequestMiddleware(t.cryptoKey))
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
			slog.Info(
				"reporting batch",
				slog.Int("worker_num", id),
				slog.Int("batch_size", len(batch)),
			)

			data, err := json.Marshal(batch)
			if err != nil {
				slog.Error(
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
				slog.Error(
					"error reporting metrics",
					slog.Any("error", err),
				)
				return fmt.Errorf("error reporting metrics: %w", err)
			}

			if resp.StatusCode() != http.StatusOK {
				slog.Error(
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
	for w := 1; w <= t.rateLimit; w++ {
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

func (t *RESTTransport) Close() error { return nil }

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
