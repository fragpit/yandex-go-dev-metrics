package agent

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/fragpit/yandex-go-dev-metrics/internal/model"
	pb "github.com/fragpit/yandex-go-dev-metrics/internal/proto"
)

var _ Transport = (*GRPCTransport)(nil)

type GRPCTransport struct {
	conn    *grpc.ClientConn
	client  pb.MetricsClient
	localIP net.IP
}

func NewGRPCTransport(
	addr string,
) (*GRPCTransport, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init grpc client: %w", err)
	}

	c := pb.NewMetricsClient(conn)

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get hostname from address %s: %w",
			addr,
			err,
		)
	}

	ip, err := localIPFor(host)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get source ip for provided server hostname: %w",
			err,
		)
	}

	return &GRPCTransport{
		conn:    conn,
		client:  c,
		localIP: ip,
	}, nil
}

func (t *GRPCTransport) SendMetrics(
	ctx context.Context,
	m map[string]model.Metric,
) error {
	slog.Info("starting reporter")

	if len(m) == 0 {
		slog.Info("no metrics to report")
		return nil
	}

	metrics := make([]*pb.Metric, 0)
	for id, metric := range m {
		pm := &pb.Metric{}
		pm.SetId(id)

		switch metric.GetType() {
		case model.CounterType:
			pm.SetType(pb.Metric_MTYPE_COUNTER)
			v, err := strconv.ParseInt(metric.GetValue(), 10, 64)
			if err != nil {
				return fmt.Errorf("failed to convert %s: %w", id, err)
			}
			pm.SetDelta(v)
		case model.GaugeType:
			pm.SetType(pb.Metric_MTYPE_GAUGE)
			v, err := strconv.ParseFloat(metric.GetValue(), 64)
			if err != nil {
				return fmt.Errorf("failed to convert %s: %w", id, err)
			}

			pm.SetValue(v)
		default:
			return fmt.Errorf("unknown metric type %s for %s", metric.GetType(), id)
		}

		metrics = append(metrics, pm)
	}

	md := metadata.New(map[string]string{"x-real-ip": t.localIP.String()})
	ctx = metadata.NewOutgoingContext(ctx, md)
	_, err := t.client.UpdateMetrics(
		ctx,
		pb.UpdateMetricsRequest_builder{Metrics: metrics}.Build(),
	)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	slog.Info("metrics sent", "num", len(metrics))
	return nil
}

func (t *GRPCTransport) Close() error {
	return t.conn.Close()
}
