package usersstats

//go:generate minimock -g -i github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/commands/usersstats.stateStatsStore -o state_stats_store_mock_test.go -n StateStatsStoreMock -p usersstats

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
	usersservice "github.com/nskondratev/socks5-proxy-server/proxy/internal/services/users"
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

type usersStatsServiceMock struct {
	getStats func(ctx context.Context) ([]usersservice.Stat, error)
}

func (m *usersStatsServiceMock) GetStats(ctx context.Context) ([]usersservice.Stat, error) {
	if m.getStats == nil {
		return nil, nil
	}

	return m.getStats(ctx)
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("sender is nil", func(t *testing.T) {
		t.Parallel()

		s := NewStateStatsStoreMock(t)
		h := New(s, &usersStatsServiceMock{})
		err := h.Handle(&contextStub{})
		require.NoError(t, err)
	})

	t.Run("sender username is empty", func(t *testing.T) {
		t.Parallel()

		s := NewStateStatsStoreMock(t)
		h := New(s, &usersStatsServiceMock{})
		err := h.Handle(&contextStub{sender: &tele.User{Username: ""}})
		require.NoError(t, err)
	})

	t.Run("stats read error", func(t *testing.T) {
		t.Parallel()

		s := NewStateStatsStoreMock(t)
		users := &usersStatsServiceMock{
			getStats: func(_ context.Context) ([]usersservice.Stat, error) {
				return nil, errors.New("read failed")
			},
		}

		c := &contextStub{sender: &tele.User{Username: "admin"}}
		bot.SetContext(c, context.Background())

		h := New(s, users)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "read failed")
	})

	t.Run("set state error", func(t *testing.T) {
		t.Parallel()

		s := NewStateStatsStoreMock(t)
		users := &usersStatsServiceMock{
			getStats: func(_ context.Context) ([]usersservice.Stat, error) {
				return nil, nil
			},
		}
		s.SetUserStateMock.Return(errors.New("save failed"))

		c := &contextStub{sender: &tele.User{Username: "admin"}}

		h := New(s, users)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "save failed")
	})

	t.Run("no stats", func(t *testing.T) {
		t.Parallel()

		s := NewStateStatsStoreMock(t)
		users := &usersStatsServiceMock{
			getStats: func(_ context.Context) ([]usersservice.Stat, error) {
				return []usersservice.Stat{}, nil
			},
		}
		s.SetUserStateMock.Set(func(_ context.Context, username string, state store.UserState) error {
			assert.Equal(t, "admin", username)
			assert.Equal(t, store.StateIdle, state.State)
			assert.Equal(t, map[string]string{}, state.Data)
			return nil
		})

		c := &contextStub{sender: &tele.User{Username: "admin"}}

		h := New(s, users)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Contains(t, c.sendCall[0].msg, "No usage stats.")
	})

	t.Run("with stats", func(t *testing.T) {
		t.Parallel()

		s := NewStateStatsStoreMock(t)
		users := &usersStatsServiceMock{
			getStats: func(_ context.Context) ([]usersservice.Stat, error) {
				return []usersservice.Stat{
					{Username: "alice", LastAuth: "2026-02-23T10:00:00.000Z", Usage: "10 MB"},
					{Username: "bob", LastAuth: "", Usage: "1 MB"},
				}, nil
			},
		}
		s.SetUserStateMock.Return(nil)

		c := &contextStub{sender: &tele.User{Username: "admin"}}

		h := New(s, users)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Contains(t, c.sendCall[0].msg, "<b>1.</b> alice")
		assert.Contains(t, c.sendCall[0].msg, "<b>2.</b> bob (-): 1 MB")
	})
}
