package message

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"strings"

	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/commands/generatepass"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/store"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/config"
)

type storeI interface {
	GetUserState(ctx context.Context, username string) (*store.UserState, error)
	SetUserState(ctx context.Context, username string, state store.UserState) error
	IsUsernameFree(ctx context.Context, username string) (bool, error)
	CreateUser(ctx context.Context, username, password string) error
	DeleteUser(ctx context.Context, username string) error
}

type Handler struct{ store storeI }

func New(store storeI) *Handler { return &Handler{store: store} }

//nolint:gocognit,gocyclo,cyclop,funlen,wsl // State machine handler is intentionally centralized.
func (h *Handler) Handle(c tele.Context) error {
	if strings.HasPrefix(strings.TrimSpace(c.Text()), "/") {
		return nil
	}

	sender := c.Sender()
	if sender == nil || sender.Username == "" {
		return nil
	}

	ctx := bot.GetContext(c)
	userState, err := h.store.GetUserState(ctx, sender.Username)
	if err != nil {
		return err
	}
	if userState == nil {
		return nil
	}

	text := strings.TrimSpace(c.Text())

	switch userState.State {
	case store.StateIdle:
		return c.Send("Enter command")
	case store.StateCreateUserEnterUsername:
		if text == "" {
			return c.Send("Username can not be empty. Enter the new one.")
		}

		isFree, err := h.store.IsUsernameFree(ctx, text)
		if err != nil {
			return err
		}
		if !isFree {
			return c.Send("This username is already taken. Enter another one.")
		}

		userState.State = store.StateCreateUserEnterPassword
		if userState.Data == nil {
			userState.Data = map[string]string{}
		}
		userState.Data["username"] = text

		suggestedPassword, err := generateSuggestedPassword()
		if err != nil {
			return err
		}

		if err = h.store.SetUserState(ctx, sender.Username, *userState); err != nil {
			return err
		}

		rm := &tele.ReplyMarkup{ResizeKeyboard: true}
		rm.Reply(rm.Row(rm.Text(suggestedPassword)))

		return c.Send("Ok. Enter the password or use the suggested one.", &tele.SendOptions{ReplyMarkup: rm})
	case store.StateCreateUserEnterPassword:
		if text == "" {
			return c.Send("Password can not be empty. Enter the new one.")
		}

		proxyUsername := userState.Data["username"]
		if err := h.store.CreateUser(ctx, proxyUsername, text); err != nil {
			return err
		}

		state := store.UserState{State: store.StateIdle, Data: map[string]string{}}
		if err := h.store.SetUserState(ctx, sender.Username, state); err != nil {
			return err
		}

		publicHost := store.CleanPublicHost(config.PublicURL())
		telegramDeeplink := buildTelegramSocks5Deeplink(config.PublicURL(), config.AppPort(), proxyUsername, text)

		message := fmt.Sprintf(
			"User created. Send these settings to the user:\n\n"+
				"<b>host:</b> %s\n"+
				"<b>port:</b> %d\n"+
				"<b>username:</b> %s\n"+
				"<b>password:</b> %s\n"+
				"<b>telegram deeplink:</b> %s",
			html.EscapeString(publicHost),
			config.AppPort(),
			html.EscapeString(proxyUsername),
			html.EscapeString(text),
			html.EscapeString(telegramDeeplink),
		)

		opts := &tele.SendOptions{
			ParseMode:   tele.ModeHTML,
			ReplyMarkup: &tele.ReplyMarkup{RemoveKeyboard: true},
		}

		return c.Send(message, opts)
	case store.StateDeleteUserEnterUsername:
		isFree, err := h.store.IsUsernameFree(ctx, text)
		if err != nil {
			return err
		}
		if isFree {
			return c.Send("User with provided username does not exists. Enter another one.")
		}

		if err = h.store.DeleteUser(ctx, text); err != nil {
			return err
		}

		state := store.UserState{State: store.StateIdle, Data: map[string]string{}}
		if err = h.store.SetUserState(ctx, sender.Username, state); err != nil {
			return err
		}

		return c.Send("User deleted.")
	}

	return nil
}

func generateSuggestedPassword() (string, error) {
	return generatepass.Generate(10)
}

func buildTelegramSocks5Deeplink(publicURL string, port int, username, password string) string {
	server := telegramSocksServer(publicURL)

	return fmt.Sprintf(
		"tg://socks?server=%s&port=%d&user=%s&pass=%s",
		url.QueryEscape(server),
		port,
		url.QueryEscape(username),
		url.QueryEscape(password),
	)
}

func telegramSocksServer(publicURL string) string {
	normalized := strings.TrimSpace(publicURL)
	if normalized == "" {
		return ""
	}

	if !strings.Contains(normalized, "://") {
		normalized = "https://" + normalized
	}

	parsedURL, err := url.Parse(normalized)
	if err == nil && parsedURL.Hostname() != "" {
		return parsedURL.Hostname()
	}

	cleanHost := store.CleanPublicHost(publicURL)
	server, _, _ := strings.Cut(cleanHost, "/")

	return server
}
