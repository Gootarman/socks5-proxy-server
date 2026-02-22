package middleware

import (
	"context"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot"
)

func SetTimeoutCtx(timeout time.Duration) func(tele.HandlerFunc) tele.HandlerFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			ctx, cancel := context.WithTimeout(bot.GetContext(c), timeout)
			defer cancel()

			bot.SetContext(c, ctx)

			return next(c)
		}
	}
}
