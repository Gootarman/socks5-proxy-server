package getusers

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/store"
)

const Command = "/get_users"

type usersStore interface {
	SetUserState(ctx context.Context, username string, state store.UserState) error
}

type usersService interface {
	GetUsers(ctx context.Context) ([]string, error)
}

type Handler struct {
	store usersStore
	users usersService
}

func New(store usersStore, users usersService) *Handler {
	return &Handler{store: store, users: users}
}

func (h *Handler) Handle(c tele.Context) error {
	sender := c.Sender()
	if sender == nil || sender.Username == "" {
		return nil
	}

	ctx := bot.GetContext(c)
	state := store.UserState{State: store.StateIdle, Data: map[string]string{}}

	if err := h.store.SetUserState(ctx, sender.Username, state); err != nil {
		return err
	}

	users, err := h.users.GetUsers(ctx)
	if err != nil {
		return err
	}

	msg := "No users."

	if len(users) > 0 {
		lines := make([]string, 0, len(users)+2)
		lines = append(lines, "<b>Users</b>:\n")

		for i, u := range users {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, u))
		}

		lines = append(lines, "", fmt.Sprintf("<b>Total: %d</b>", len(users)))
		msg = strings.Join(lines, "\n")
	}

	opts := &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: &tele.ReplyMarkup{RemoveKeyboard: true},
	}

	return c.Send(msg, opts)
}
