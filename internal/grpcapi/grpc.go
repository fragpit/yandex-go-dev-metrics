package grpcapi

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"

	pb "github.com/fragpit/yandex-go-dev-metrics/internal/proto"
	"github.com/fragpit/yandex-go-dev-metrics/internal/repository"
)

const (
	apiShutdownTimeout = 5 * time.Second
)

type GRPCAPI struct {
	address       string
	repo          repository.Repository
	trustedSubnet *net.IPNet
}

type Option func(*GRPCAPI) error

func WithTrustedSubnet(subnet string) Option {
	return func(g *GRPCAPI) error {
		_, sNet, err := net.ParseCIDR(subnet)
		if err != nil {
			return fmt.Errorf("failed to parse trusted subnet: %w", err)
		}

		g.trustedSubnet = sNet
		return nil
	}
}

func NewGRPCAPI(
	address string,
	repo repository.Repository,
	opts ...Option,
) (*GRPCAPI, error) {
	g := &GRPCAPI{
		address: address,
		repo:    repo,
	}

	for _, opt := range opts {
		if err := opt(g); err != nil {
			return nil, err
		}
	}

	return g, nil
}

func (g *GRPCAPI) Run(ctx context.Context) error {
	slog.Info("grpc api server started", "address", g.address)

	listener, err := net.Listen("tcp4", g.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", g.address, err)
	}

	var opts []grpc.ServerOption
	if g.trustedSubnet != nil {
		opts = append(
			opts,
			grpc.UnaryInterceptor(verifySubnetInterceptor(g.trustedSubnet)),
		)
	}

	gs := grpc.NewServer(opts...)
	pb.RegisterMetricsServer(gs, &MetricsService{repo: g.repo})

	errChan := make(chan error, 1)
	go func() {
		if err := gs.Serve(listener); err != nil {
			slog.Error("grpc api server failed", slog.Any("error", err))
			errChan <- err
			return
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			apiShutdownTimeout,
		)
		defer cancel()

		done := make(chan struct{})
		go func() {
			gs.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("grpc api server shut down gracefully")
		case <-shutdownCtx.Done():
			slog.Warn("grpc api server shut down")
			gs.Stop()
		}
	}

	return nil
}
