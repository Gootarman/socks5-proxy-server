package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tele "gopkg.in/telebot.v3"
)

func TestBuildTelegramWebhookURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		url, err := buildTelegramWebhookURL(Config{
			PublicURL:    "https://example.com/base/",
			WebHookURL:   "/webhook",
			TelegramAuth: "token",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://example.com/base/webhooktoken", url)
	})

	t.Run("invalid public url", func(t *testing.T) {
		_, err := buildTelegramWebhookURL(Config{
			PublicURL:    "://bad",
			WebHookURL:   "/webhook",
			TelegramAuth: "token",
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse PUBLIC_URL")
	})

	t.Run("non absolute public url", func(t *testing.T) {
		_, err := buildTelegramWebhookURL(Config{
			PublicURL:    "/relative",
			WebHookURL:   "/webhook",
			TelegramAuth: "token",
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "PUBLIC_URL must be an absolute URL")
	})
}

func TestMakeTelegramPoller(t *testing.T) {
	t.Run("long poller when webhooks disabled", func(t *testing.T) {
		p, err := makeTelegramPoller(Config{UseWebHooks: false})
		require.NoError(t, err)
		_, ok := p.(*tele.LongPoller)
		assert.True(t, ok)
	})

	t.Run("webhook without tls", func(t *testing.T) {
		p, err := makeTelegramPoller(Config{
			UseWebHooks:        true,
			PublicURL:          "https://example.com",
			WebHookURL:         "/webhook",
			TelegramAuth:       "token",
			BotAppPort:         9443,
			WebhookTLSCertPath: "",
			WebhookTLSKeyPath:  "",
		})
		require.NoError(t, err)

		webhook, ok := p.(*tele.Webhook)
		require.True(t, ok)
		assert.Equal(t, ":9443", webhook.Listen)
		require.NotNil(t, webhook.Endpoint)
		assert.Equal(t, "https://example.com/webhooktoken", webhook.Endpoint.PublicURL)
		assert.Nil(t, webhook.TLS)
	})

	t.Run("webhook with tls", func(t *testing.T) {
		p, err := makeTelegramPoller(Config{
			UseWebHooks:        true,
			PublicURL:          "https://example.com",
			WebHookURL:         "/webhook",
			TelegramAuth:       "token",
			BotAppPort:         9443,
			WebhookTLSCertPath: "cert.pem",
			WebhookTLSKeyPath:  "key.pem",
		})
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
		_, err := makeTelegramPoller(Config{
			UseWebHooks: true,
			PublicURL:   "bad",
		})
		require.Error(t, err)
	})
}
