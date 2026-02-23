package proxy

import (
	"context"
	"net"

	"github.com/things-go/go-socks5"
)

type dialContext interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

func NewDialAndRequest(
	dialer dialContext,
	onConnected func(request *socks5.Request, conn net.Conn) net.Conn,
) func(ctx context.Context, network, addr string, request *socks5.Request) (net.Conn, error) {
	return func(ctx context.Context, network, addr string, request *socks5.Request) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		if onConnected == nil {
			return conn, nil
		}

		updatedConn := onConnected(request, conn)
		if updatedConn == nil {
			return conn, nil
		}

		return updatedConn, nil
	}
}
