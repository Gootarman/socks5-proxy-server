package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/things-go/go-socks5"
	tele "gopkg.in/telebot.v3"

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

func TestBuildTelegramWebhookURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Setenv("PUBLIC_URL", "https://example.com/base/")
		t.Setenv("TELEGRAM_WEBHOOK_URL", "/webhook")
		t.Setenv("TELEGRAM_API_TOKEN", "token")

		url, err := buildTelegramWebhookURL()
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/base/webhooktoken", url)
	})

	t.Run("invalid public url", func(t *testing.T) {
		t.Setenv("PUBLIC_URL", "://bad")
		t.Setenv("TELEGRAM_WEBHOOK_URL", "/webhook")
		t.Setenv("TELEGRAM_API_TOKEN", "token")

		_, err := buildTelegramWebhookURL()
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse PUBLIC_URL")
	})

	t.Run("non absolute public url", func(t *testing.T) {
		t.Setenv("PUBLIC_URL", "/relative")
		t.Setenv("TELEGRAM_WEBHOOK_URL", "/webhook")
		t.Setenv("TELEGRAM_API_TOKEN", "token")

		_, err := buildTelegramWebhookURL()
		require.Error(t, err)
		assert.ErrorContains(t, err, "PUBLIC_URL must be an absolute URL")
	})
}

func TestMakeTelegramPoller(t *testing.T) {
	t.Run("long poller when webhooks disabled", func(t *testing.T) {
		t.Setenv("TELEGRAM_USE_WEBHOOKS", "false")

		p, err := makeTelegramPoller()
		require.NoError(t, err)
		_, ok := p.(*tele.LongPoller)
		assert.True(t, ok)
	})

	t.Run("webhook without tls", func(t *testing.T) {
		t.Setenv("TELEGRAM_USE_WEBHOOKS", "true")
		t.Setenv("PUBLIC_URL", "https://example.com")
		t.Setenv("TELEGRAM_WEBHOOK_URL", "/webhook")
		t.Setenv("TELEGRAM_API_TOKEN", "token")
		t.Setenv("BOT_APP_PORT", "9443")
		t.Setenv("TELEGRAM_WEBHOOK_TLS_CERT_PATH", "")
		t.Setenv("TELEGRAM_WEBHOOK_TLS_KEY_PATH", "")

		p, err := makeTelegramPoller()
		require.NoError(t, err)

		webhook, ok := p.(*tele.Webhook)
		require.True(t, ok)
		assert.Equal(t, ":9443", webhook.Listen)
		require.NotNil(t, webhook.Endpoint)
		assert.Equal(t, "https://example.com/webhooktoken", webhook.Endpoint.PublicURL)
		assert.Nil(t, webhook.TLS)
	})

	t.Run("webhook with tls", func(t *testing.T) {
		t.Setenv("TELEGRAM_USE_WEBHOOKS", "true")
		t.Setenv("PUBLIC_URL", "https://example.com")
		t.Setenv("TELEGRAM_WEBHOOK_URL", "/webhook")
		t.Setenv("TELEGRAM_API_TOKEN", "token")
		t.Setenv("BOT_APP_PORT", "9443")
		t.Setenv("TELEGRAM_WEBHOOK_TLS_CERT_PATH", "cert.pem")
		t.Setenv("TELEGRAM_WEBHOOK_TLS_KEY_PATH", "key.pem")

		p, err := makeTelegramPoller()
		require.NoError(t, err)

		webhook, ok := p.(*tele.Webhook)
		require.True(t, ok)
		require.NotNil(t, webhook.TLS)
		assert.Equal(t, "cert.pem", webhook.TLS.Cert)
		assert.Equal(t, "key.pem", webhook.TLS.Key)
		require.NotNil(t, webhook.Endpoint)
		assert.Equal(t, "cert.pem", webhook.Endpoint.Cert)
	})

	t.Run("webhook build url error", func(t *testing.T) {
		t.Setenv("TELEGRAM_USE_WEBHOOKS", "true")
		t.Setenv("PUBLIC_URL", "bad")

		_, err := makeTelegramPoller()
		require.Error(t, err)
	})
}
