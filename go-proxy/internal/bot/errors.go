package bot

import (
	"log/slog"

	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/internal/log"
)

func OnErrorCb(err error, c tele.Context) {
	slog.LogAttrs(
		GetContext(c),
		slog.LevelError,
		"error while processing update",
		slog.String(log.FieldError, err.Error()),
	)

	replyErr := c.Reply("Some error occurred, check server logs for details.")
	if replyErr != nil {
		slog.LogAttrs(
			GetContext(c),
			slog.LevelError,
			"failed to send error reply message",
			slog.String(log.FieldError, err.Error()),
		)
	}
}
