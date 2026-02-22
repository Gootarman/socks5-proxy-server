package deleteuser

//go:generate minimock -g -i github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/commands/deleteuser.stateStore -o state_store_mock_test.go -n StateStoreMock -p deleteuser

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/store"
)

type contextStub struct {
	tele.Context
	sender  *tele.User
	values  map[string]interface{}
	sendErr error
	sent    []string
}

func (c *contextStub) Sender() *tele.User {
	return c.sender
}

func (c *contextStub) Send(what interface{}, _ ...interface{}) error {
	c.sent = append(c.sent, fmt.Sprint(what))
	return c.sendErr
}

func (c *contextStub) Set(key string, val interface{}) {
	if c.values == nil {
		c.values = map[string]interface{}{}
	}

	c.values[key] = val
}

func (c *contextStub) Get(key string) interface{} {
	if c.values == nil {
		return nil
	}

	return c.values[key]
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("sender is nil", func(t *testing.T) {
		t.Parallel()

		s := NewStateStoreMock(t)
		h := New(s)
		err := h.Handle(&contextStub{})
		require.NoError(t, err)
	})

	t.Run("sender username is empty", func(t *testing.T) {
		t.Parallel()

		s := NewStateStoreMock(t)
		h := New(s)
		err := h.Handle(&contextStub{sender: &tele.User{Username: ""}})
		require.NoError(t, err)
	})

	t.Run("set state error", func(t *testing.T) {
		t.Parallel()

		s := NewStateStoreMock(t)
		s.SetUserStateMock.Return(errors.New("boom"))

		c := &contextStub{sender: &tele.User{Username: "admin"}}
		bot.SetContext(c, context.Background())

		h := New(s)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "boom")
	})

	t.Run("send error", func(t *testing.T) {
		t.Parallel()

		s := NewStateStoreMock(t)
		s.SetUserStateMock.Return(nil)

		c := &contextStub{
			sender:  &tele.User{Username: "admin"},
			sendErr: errors.New("send failed"),
		}

		h := New(s)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "send failed")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		s := NewStateStoreMock(t)
		s.SetUserStateMock.Set(func(_ context.Context, username string, state store.UserState) error {
			assert.Equal(t, "admin", username)
			assert.Equal(t, store.StateDeleteUserEnterUsername, state.State)
			assert.Equal(t, map[string]string{}, state.Data)
			return nil
		})

		c := &contextStub{sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sent, 1)
		assert.Equal(t, "Enter username to delete.", c.sent[0])
	})
}
