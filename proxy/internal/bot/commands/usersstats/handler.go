package usersstats

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/store"
	formatter "github.com/nskondratev/socks5-proxy-server/proxy/internal/format"
	usersservice "github.com/nskondratev/socks5-proxy-server/proxy/internal/services/users"
)

const Command = "/users_stats"

type stateStatsStore interface {
	SetUserState(ctx context.Context, username string, state store.UserState) error
}

type usersStatsService interface {
	GetStats(ctx context.Context) ([]usersservice.Stat, error)
}

type Handler struct {
	store stateStatsStore
	users usersStatsService
}

func New(store stateStatsStore, users usersStatsService) *Handler {
	return &Handler{store: store, users: users}
}

func (h *Handler) Handle(c tele.Context) error {
	sender := c.Sender()
	if sender == nil || sender.Username == "" {
		return nil
	}

	ctx := bot.GetContext(c)

	stats, err := h.users.GetStats(ctx)
	if err != nil {
		return err
	}

	msg := "<b>Data usage by users:</b>\n\n"
	if len(stats) == 0 {
		msg += "No usage stats."
	} else {
		parts := make([]string, 0, len(stats))

		for i, s := range stats {
			parts = append(
				parts,
				fmt.Sprintf("<b>%d.</b> %s (%s): %s", i+1, s.Username, formatter.FromNow(s.LastAuth), s.Usage),
			)
		}

		msg += strings.Join(parts, "\n")
	}

	state := store.UserState{State: store.StateIdle, Data: map[string]string{}}
	if err = h.store.SetUserState(ctx, sender.Username, state); err != nil {
		return err
	}

	opts := &tele.SendOptions{
		ParseMode:   tele.ModeHTML,
		ReplyMarkup: &tele.ReplyMarkup{RemoveKeyboard: true},
	}

	return c.Send(msg, opts)
}
