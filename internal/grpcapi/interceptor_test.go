package grpcapi

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func mustCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("failed to parse cidr: %v", err)
	}
	return ipnet
}

func TestVerifySubnetInterceptor(t *testing.T) {
	trusted := mustCIDR(t, "192.168.1.0/24")

	interceptor := verifySubnetInterceptor(trusted)

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	tests := []struct {
		name       string
		ctx        context.Context
		wantErr    bool
		wantCode   codes.Code
		hitHandler bool
	}{
		{
			name:     "missing header",
			ctx:      context.Background(),
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
		{
			name: "bad ip value",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("x-real-ip", "not-an-ip"),
			),
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
		{
			name: "ip not in subnet",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("x-real-ip", "10.0.0.1"),
			),
			wantErr:  true,
			wantCode: codes.Unauthenticated,
		},
		{
			name: "ip in subnet",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("x-real-ip", "192.168.1.10"),
			),
			wantErr:    false,
			wantCode:   codes.OK,
			hitHandler: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit := false
			wrappedHandler := func(ctx context.Context, req any) (any, error) {
				hit = true
				return handler(ctx, req)
			}

			resp, err := interceptor(tt.ctx, nil, info, wrappedHandler)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil, resp=%v", resp)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected status error, got %T", err)
				}
				if st.Code() != tt.wantCode {
					t.Fatalf("unexpected code: got %v want %v", st.Code(), tt.wantCode)
				}
				if hit {
					t.Fatalf("handler should not be called on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp != "ok" {
				t.Fatalf("unexpected response: %v", resp)
			}
			if tt.hitHandler && !hit {
				t.Fatalf("expected handler to be called")
			}
			if !tt.hitHandler && hit {
				t.Fatalf("did not expect handler to be called")
			}
		})
	}
}
