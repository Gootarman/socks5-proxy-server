package middleware

//go:generate minimock -g -i github.com/nskondratev/socks5-proxy-server/internal/bot/middleware.adminService -o admin_service_mock_test.go -n AdminServiceMock -p middleware

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/internal/bot"
)

type middlewareContextStub struct {
	tele.Context
	sender   *tele.User
	values   map[string]interface{}
	sentText []string
}

func (c *middlewareContextStub) Sender() *tele.User {
	return c.sender
}

func (c *middlewareContextStub) Send(what interface{}, _ ...interface{}) error {
	c.sentText = append(c.sentText, fmt.Sprint(what))
	return nil
}

func (c *middlewareContextStub) Set(key string, val interface{}) {
	if c.values == nil {
		c.values = map[string]interface{}{}
	}

	c.values[key] = val
}

func (c *middlewareContextStub) Get(key string) interface{} {
	if c.values == nil {
		return nil
	}

	return c.values[key]
}

func TestRestrictByAdminUserID(t *testing.T) {
	t.Parallel()

	next := func(c tele.Context) error {
		c.Set("next_called", true)
		return nil
	}

	t.Run("sender is nil", func(t *testing.T) {
		t.Parallel()

		adminSvc := NewAdminServiceMock(t)
		c := &middlewareContextStub{}

		err := RestrictByAdminUserID(adminSvc)(next)(c)
		require.NoError(t, err)
		require.Len(t, c.sentText, 1)
		assert.Contains(t, c.sentText[0], "only for admin users")
		assert.Nil(t, c.Get("next_called"))
	})

	t.Run("username is empty", func(t *testing.T) {
		t.Parallel()

		adminSvc := NewAdminServiceMock(t)
		c := &middlewareContextStub{sender: &tele.User{Username: ""}}

		err := RestrictByAdminUserID(adminSvc)(next)(c)
		require.NoError(t, err)
		require.Len(t, c.sentText, 1)
		assert.Contains(t, c.sentText[0], "only for admin users")
		assert.Nil(t, c.Get("next_called"))
	})

	t.Run("admin service error", func(t *testing.T) {
		t.Parallel()

		adminSvc := NewAdminServiceMock(t)
		adminSvc.IsAdminMock.Return(false, assert.AnError)
		c := &middlewareContextStub{sender: &tele.User{Username: "alice"}}

		err := RestrictByAdminUserID(adminSvc)(next)(c)
		require.Error(t, err)
		assert.Equal(t, assert.AnError, err)
		assert.Nil(t, c.Get("next_called"))
	})

	t.Run("user is not admin", func(t *testing.T) {
		t.Parallel()

		adminSvc := NewAdminServiceMock(t)
		adminSvc.IsAdminMock.Return(false, nil)
		c := &middlewareContextStub{sender: &tele.User{Username: "alice"}}

		err := RestrictByAdminUserID(adminSvc)(next)(c)
		require.NoError(t, err)
		require.Len(t, c.sentText, 1)
		assert.Contains(t, c.sentText[0], "only for admin users")
		assert.Nil(t, c.Get("next_called"))
	})

	t.Run("admin passes through", func(t *testing.T) {
		t.Parallel()

		adminSvc := NewAdminServiceMock(t)
		adminSvc.IsAdminMock.Return(true, nil)
		c := &middlewareContextStub{sender: &tele.User{Username: "alice"}}

		err := RestrictByAdminUserID(adminSvc)(next)(c)
		require.NoError(t, err)
		assert.Equal(t, true, c.Get("next_called"))
		assert.Empty(t, c.sentText)
	})
}

func TestSetTimeoutCtx(t *testing.T) {
	t.Parallel()

	t.Run("creates child context with deadline", func(t *testing.T) {
		t.Parallel()

		c := &middlewareContextStub{}
		timeout := 100 * time.Millisecond
		var gotCtx context.Context

		err := SetTimeoutCtx(timeout)(func(tc tele.Context) error {
			gotCtx = bot.GetContext(tc)
			return nil
		})(c)
		require.NoError(t, err)
		require.NotNil(t, gotCtx)
		_, hasDeadline := gotCtx.Deadline()
		assert.True(t, hasDeadline)
	})

	t.Run("inherits parent context values", func(t *testing.T) {
		t.Parallel()

		c := &middlewareContextStub{}
		parent := context.WithValue(context.Background(), "k", "v")
		bot.SetContext(c, parent)

		err := SetTimeoutCtx(time.Second)(func(tc tele.Context) error {
			got := bot.GetContext(tc)
			assert.Equal(t, "v", got.Value("k"))
			return nil
		})(c)
		require.NoError(t, err)
	})
}
