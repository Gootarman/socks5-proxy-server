package getusers

//go:generate minimock -g -i github.com/nskondratev/socks5-proxy-server/internal/bot/commands/getusers.usersStore -o users_store_mock_test.go -n UsersStoreMock -p getusers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/internal/bot"
	"github.com/nskondratev/socks5-proxy-server/internal/bot/store"
)

type sendCall struct {
	msg  string
	opts []interface{}
}

type contextStub struct {
	tele.Context
	sender   *tele.User
	values   map[string]interface{}
	sendErr  error
	sendCall []sendCall
}

func (c *contextStub) Sender() *tele.User {
	return c.sender
}

func (c *contextStub) Send(what interface{}, opts ...interface{}) error {
	c.sendCall = append(c.sendCall, sendCall{msg: fmt.Sprint(what), opts: opts})
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

		s := NewUsersStoreMock(t)
		h := New(s)
		err := h.Handle(&contextStub{})
		require.NoError(t, err)
	})

	t.Run("sender username is empty", func(t *testing.T) {
		t.Parallel()

		s := NewUsersStoreMock(t)
		h := New(s)
		err := h.Handle(&contextStub{sender: &tele.User{Username: ""}})
		require.NoError(t, err)
	})

	t.Run("set state error", func(t *testing.T) {
		t.Parallel()

		s := NewUsersStoreMock(t)
		s.SetUserStateMock.Return(errors.New("save failed"))

		c := &contextStub{sender: &tele.User{Username: "admin"}}
		bot.SetContext(c, context.Background())

		h := New(s)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "save failed")
	})

	t.Run("get users error", func(t *testing.T) {
		t.Parallel()

		s := NewUsersStoreMock(t)
		s.SetUserStateMock.Return(nil)
		s.GetUsersMock.Return(nil, errors.New("read failed"))

		c := &contextStub{sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "read failed")
	})

	t.Run("no users", func(t *testing.T) {
		t.Parallel()

		s := NewUsersStoreMock(t)
		s.SetUserStateMock.Set(func(_ context.Context, username string, state store.UserState) error {
			assert.Equal(t, "admin", username)
			assert.Equal(t, store.StateIdle, state.State)
			assert.Equal(t, map[string]string{}, state.Data)
			return nil
		})
		s.GetUsersMock.Return([]string{}, nil)

		c := &contextStub{sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Equal(t, "No users.", c.sendCall[0].msg)
		require.Len(t, c.sendCall[0].opts, 1)
		_, ok := c.sendCall[0].opts[0].(*tele.SendOptions)
		assert.True(t, ok)
	})

	t.Run("user list", func(t *testing.T) {
		t.Parallel()

		s := NewUsersStoreMock(t)
		s.SetUserStateMock.Return(nil)
		s.GetUsersMock.Return([]string{"alice", "bob"}, nil)

		c := &contextStub{sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Contains(t, c.sendCall[0].msg, "<b>Users</b>")
		assert.Contains(t, c.sendCall[0].msg, "1. alice")
		assert.Contains(t, c.sendCall[0].msg, "2. bob")
		assert.Contains(t, c.sendCall[0].msg, "<b>Total: 2</b>")
	})
}
