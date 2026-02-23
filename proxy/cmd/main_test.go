package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/things-go/go-socks5"

	proxycore "github.com/nskondratev/socks5-proxy-server/proxy/internal/proxy"
)

func TestGetUsernameFromRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *socks5.Request
		want    string
		wantOK  bool
	}{
		{
			name:    "nil request",
			request: nil,
			want:    "",
			wantOK:  false,
		},
		{
			name:    "nil auth context",
			request: &socks5.Request{},
			want:    "",
			wantOK:  false,
		},
		{
			name: "username key",
			request: &socks5.Request{
				AuthContext: &socks5.AuthContext{
					Payload: map[string]string{"Username": "alice"},
				},
			},
			want:   "alice",
			wantOK: true,
		},
		{
			name: "lowercase username key",
			request: &socks5.Request{
				AuthContext: &socks5.AuthContext{
					Payload: map[string]string{"username": "bob"},
				},
			},
			want:   "bob",
			wantOK: true,
		},
		{
			name: "missing username",
			request: &socks5.Request{
				AuthContext: &socks5.AuthContext{
					Payload: map[string]string{"password": "secret"},
				},
			},
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := proxycore.UsernameFromRequest(tt.request)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
