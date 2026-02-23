package proxy

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/things-go/go-socks5"
)

type dialContextMock struct {
	dialFn func(ctx context.Context, network, addr string) (net.Conn, error)
}

type wrappedConn struct {
	net.Conn
}

func (m dialContextMock) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return m.dialFn(ctx, network, addr)
}

func TestNewDialAndRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		dialErr       error
		onConnected   func(request *socks5.Request, conn net.Conn) net.Conn
		wantErr       bool
		wantWrapped   bool
		wantConnected bool
	}{
		{
			name:    "dial error",
			dialErr: errors.New("dial failed"),
			wantErr: true,
		},
		{
			name:          "without callback",
			wantConnected: true,
		},
		{
			name: "with callback",
			onConnected: func(_ *socks5.Request, conn net.Conn) net.Conn {
				return &wrappedConn{Conn: conn}
			},
			wantWrapped:   true,
			wantConnected: true,
		},
		{
			name: "callback returns nil",
			onConnected: func(_ *socks5.Request, _ net.Conn) net.Conn {
				return nil
			},
			wantConnected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			connected := false
			dialer := dialContextMock{
				dialFn: func(_ context.Context, _, _ string) (net.Conn, error) {
					if tt.dialErr != nil {
						return nil, tt.dialErr
					}

					client, server := net.Pipe()
					t.Cleanup(func() {
						_ = client.Close()
						_ = server.Close()
					})
					connected = true

					return client, nil
				},
			}

			dialAndRequest := NewDialAndRequest(dialer, tt.onConnected)
			gotConn, err := dialAndRequest(context.Background(), "tcp", "127.0.0.1:1080", nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error state = %v, want error: %v", err, tt.wantErr)
			}

			if tt.wantErr {
				if gotConn != nil {
					t.Fatal("connection must be nil on dial error")
				}

				return
			}

			if gotConn == nil {
				t.Fatal("connection must not be nil")
			}

			if connected != tt.wantConnected {
				t.Fatalf("connected state = %v, want %v", connected, tt.wantConnected)
			}

			_, wrapped := gotConn.(*wrappedConn)
			if wrapped != tt.wantWrapped {
				t.Fatalf("wrapped state = %v, want %v", wrapped, tt.wantWrapped)
			}
		})
	}
}
