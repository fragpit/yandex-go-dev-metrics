package grpcapi

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func verifySubnetInterceptor(
	trustedSubnet *net.IPNet,
) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		xRealIPKey := "x-real-ip"
		ip := metadata.ValueFromIncomingContext(ctx, xRealIPKey)
		if ip == nil {
			return nil, status.Error(codes.Unauthenticated, "x-real-ip not set")
		}

		clientIP := net.ParseIP(ip[0])
		if clientIP == nil {
			return nil, status.Error(
				codes.Unauthenticated,
				"failed to parse x-real-ip",
			)
		}

		if !trustedSubnet.Contains(clientIP) {
			return nil, status.Error(
				codes.Unauthenticated,
				fmt.Sprintf("access forbidden for ip %s", clientIP),
			)
		}

		return handler(ctx, req)
	}
}
