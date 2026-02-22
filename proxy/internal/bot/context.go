package bot

import (
	"context"

	tele "gopkg.in/telebot.v3"
)

const (
	contextFieldContext = "context"
)

//nolint:revive // Other funcs get tele.Context as the first argument
func SetContext(c tele.Context, ctx context.Context) {
	c.Set(contextFieldContext, ctx)
}

func GetContext(c tele.Context) context.Context {
	if c == nil {
		return context.Background()
	}

	if ctx, ok := c.Get(contextFieldContext).(context.Context); ok {
		return ctx
	}

	return context.Background()
}
