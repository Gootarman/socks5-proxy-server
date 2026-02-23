package bootstrap

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/commands/createuser"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/commands/deleteuser"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/commands/generatepass"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/commands/getusers"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/commands/start"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/commands/usersstats"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/handlers/message"
	mw "github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/middleware"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/store"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/redis"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/services/admin"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/services/users"

	botcore "github.com/nskondratev/socks5-proxy-server/proxy/internal/bot"
)

const httpClientTimeout = 30 * time.Second

type Config struct {
	TelegramAuth            string
	UseWebHooks             bool
	PublicURL               string
	WebHookURL              string
	BotAppPort              int
	WebhookTLSCertPath      string
	WebhookTLSKeyPath       string
	UpdateProcessingTimeout time.Duration
	HTTPClientTimeout       time.Duration
}

type NewParams struct {
	Config       Config
	Redis        *redis.Redis
	UsersService *users.Users
}

func New(params NewParams) (*tele.Bot, error) {
	poller, err := makeTelegramPoller(params.Config)
	if err != nil {
		return nil, err
	}

	clientTimeout := params.Config.HTTPClientTimeout
	if clientTimeout <= 0 {
		clientTimeout = httpClientTimeout
	}

	botConf := tele.Settings{
		Token: params.Config.TelegramAuth,
		Client: &http.Client{
			Timeout: clientTimeout,
		},
		OnError: botcore.OnErrorCb,
		Poller:  poller,
	}

	b, err := tele.NewBot(botConf)
	if err != nil {
		return nil, fmt.Errorf("failed to create new telegram bot: %w", err)
	}

	if !params.Config.UseWebHooks {
		if err := b.RemoveWebhook(); err != nil {
			return nil, fmt.Errorf("failed to remove webhook: %w", err)
		}
	}

	adminService := admin.New(params.Redis)
	b.Use(
		mw.SetTimeoutCtx(params.Config.UpdateProcessingTimeout),
		mw.RestrictByAdminUserID(adminService),
	)

	botStore := store.New(params.Redis)

	b.Handle(start.Command, start.New(botStore).Handle)
	b.Handle(usersstats.Command, usersstats.New(botStore, params.UsersService).Handle)
	b.Handle(createuser.Command, createuser.New(botStore).Handle)
	b.Handle(deleteuser.Command, deleteuser.New(botStore).Handle)
	b.Handle(getusers.Command, getusers.New(botStore, params.UsersService).Handle)
	b.Handle(generatepass.Command, generatepass.New().Handle)
	b.Handle(tele.OnText, message.New(botStore, params.UsersService).Handle)

	return b, nil
}

func makeTelegramPoller(cfg Config) (tele.Poller, error) {
	if !cfg.UseWebHooks {
		return &tele.LongPoller{Timeout: 10 * time.Second}, nil
	}

	webHookURL, err := buildTelegramWebhookURL(cfg)
	if err != nil {
		return nil, err
	}

	webhookEndpoint := &tele.WebhookEndpoint{PublicURL: webHookURL}
	poller := &tele.Webhook{
		Listen:   fmt.Sprintf(":%d", cfg.BotAppPort),
		Endpoint: webhookEndpoint,
	}

	certPath := cfg.WebhookTLSCertPath
	keyPath := cfg.WebhookTLSKeyPath

	if certPath == "" || keyPath == "" {
		return poller, nil
	}

	webhookEndpoint.Cert = certPath
	poller.TLS = &tele.WebhookTLS{Cert: certPath, Key: keyPath}

	return poller, nil
}

func buildTelegramWebhookURL(cfg Config) (string, error) {
	parsedPublicURL, err := url.Parse(cfg.PublicURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse PUBLIC_URL: %w", err)
	}

	if parsedPublicURL.Scheme == "" || parsedPublicURL.Host == "" {
		return "", fmt.Errorf("PUBLIC_URL must be an absolute URL")
	}

	parsedPublicURL.Path = parsedPublicURL.JoinPath(cfg.WebHookURL).Path + cfg.TelegramAuth

	return parsedPublicURL.String(), nil
}
