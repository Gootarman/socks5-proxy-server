package message

//go:generate minimock -g -i github.com/nskondratev/socks5-proxy-server/proxy/internal/bot/handlers/message.storeI -o store_mock_test.go -n StoreIMock -p message

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

type sendCall struct {
	msg  string
	opts []interface{}
}

type contextStub struct {
	tele.Context
	sender   *tele.User
	text     string
	values   map[string]interface{}
	sendErr  error
	sendCall []sendCall
}

func (c *contextStub) Text() string {
	return c.text
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

func TestHandler_Handle_NoStateProcessing(t *testing.T) {
	t.Parallel()

	t.Run("command text is ignored", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		h := New(s)
		err := h.Handle(&contextStub{text: "/start", sender: &tele.User{Username: "admin"}})
		require.NoError(t, err)
	})

	t.Run("sender is nil", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		h := New(s)
		err := h.Handle(&contextStub{text: "hello"})
		require.NoError(t, err)
	})

	t.Run("sender username is empty", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		h := New(s)
		err := h.Handle(&contextStub{text: "hello", sender: &tele.User{Username: ""}})
		require.NoError(t, err)
	})

	t.Run("get user state error", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(nil, errors.New("boom"))

		h := New(s)
		err := h.Handle(&contextStub{text: "hello", sender: &tele.User{Username: "admin"}})
		require.Error(t, err)
		assert.EqualError(t, err, "boom")
	})

	t.Run("state is nil", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(nil, nil)

		h := New(s)
		err := h.Handle(&contextStub{text: "hello", sender: &tele.User{Username: "admin"}})
		require.NoError(t, err)
	})
}

func TestHandler_Handle_IdleState(t *testing.T) {
	t.Parallel()

	s := NewStoreIMock(t)
	s.GetUserStateMock.Return(&store.UserState{State: store.StateIdle, Data: map[string]string{}}, nil)

	c := &contextStub{text: "hello", sender: &tele.User{Username: "admin"}}

	h := New(s)
	err := h.Handle(c)
	require.NoError(t, err)
	require.Len(t, c.sendCall, 1)
	assert.Equal(t, "Enter command", c.sendCall[0].msg)
}

func TestHandler_Handle_CreateUserEnterUsername(t *testing.T) {
	t.Parallel()

	baseState := &store.UserState{State: store.StateCreateUserEnterUsername}

	t.Run("empty username", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(baseState, nil)

		c := &contextStub{text: "  ", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Equal(t, "Username can not be empty. Enter the new one.", c.sendCall[0].msg)
	})

	t.Run("username check error", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(baseState, nil)
		s.IsUsernameFreeMock.Return(false, errors.New("lookup failed"))

		c := &contextStub{text: "proxy-user", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "lookup failed")
	})

	t.Run("username is taken", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(baseState, nil)
		s.IsUsernameFreeMock.Return(false, nil)

		c := &contextStub{text: "proxy-user", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Equal(t, "This username is already taken. Enter another one.", c.sendCall[0].msg)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(&store.UserState{State: store.StateCreateUserEnterUsername}, nil)
		s.IsUsernameFreeMock.Return(true, nil)
		s.SetUserStateMock.Set(func(_ context.Context, username string, state store.UserState) error {
			assert.Equal(t, "admin", username)
			assert.Equal(t, store.StateCreateUserEnterPassword, state.State)
			assert.Equal(t, "proxy-user", state.Data["username"])
			return nil
		})

		c := &contextStub{text: "proxy-user", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Equal(t, "Ok. Enter the password or use the suggested one.", c.sendCall[0].msg)
		require.Len(t, c.sendCall[0].opts, 1)
		_, ok := c.sendCall[0].opts[0].(*tele.SendOptions)
		assert.True(t, ok)
	})
}

func TestHandler_Handle_CreateUserEnterPassword(t *testing.T) {
	t.Run("empty password", func(t *testing.T) {
		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(&store.UserState{
			State: store.StateCreateUserEnterPassword,
			Data:  map[string]string{"username": "proxy-user"},
		}, nil)

		c := &contextStub{text: "", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Equal(t, "Password can not be empty. Enter the new one.", c.sendCall[0].msg)
	})

	t.Run("create user error", func(t *testing.T) {
		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(&store.UserState{
			State: store.StateCreateUserEnterPassword,
			Data:  map[string]string{"username": "proxy-user"},
		}, nil)
		s.CreateUserMock.Return(errors.New("create failed"))

		c := &contextStub{text: "pass", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "create failed")
	})

	t.Run("set state error", func(t *testing.T) {
		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(&store.UserState{
			State: store.StateCreateUserEnterPassword,
			Data:  map[string]string{"username": "proxy-user"},
		}, nil)
		s.CreateUserMock.Return(nil)
		s.SetUserStateMock.Return(errors.New("state failed"))

		c := &contextStub{text: "pass", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "state failed")
	})

	t.Run("success", func(t *testing.T) {
		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(&store.UserState{
			State: store.StateCreateUserEnterPassword,
			Data:  map[string]string{"username": "proxy-user"},
		}, nil)
		s.CreateUserMock.Expect(context.Background(), "proxy-user", "pass").Return(nil)
		s.SetUserStateMock.Return(nil)

		c := &contextStub{text: "pass", sender: &tele.User{Username: "admin"}}
		bot.SetContext(c, context.Background())

		t.Setenv("PUBLIC_URL", "https://proxy.example.com/")
		t.Setenv("APP_PORT", "1080")

		h := New(s)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Contains(t, c.sendCall[0].msg, "<b>host:</b> proxy.example.com")
		assert.Contains(t, c.sendCall[0].msg, "<b>port:</b> 1080")
		assert.Contains(t, c.sendCall[0].msg, "<b>username:</b> proxy-user")
		assert.Contains(t, c.sendCall[0].msg, "<b>password:</b> pass")
		assert.Contains(
			t,
			c.sendCall[0].msg,
			"<b>telegram deeplink:</b> tg://socks?server=proxy.example.com"+
				"&amp;port=1080&amp;user=proxy-user&amp;pass=pass",
		)
	})
}

func TestBuildTelegramSocks5Deeplink(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"tg://socks?server=proxy.example.com&port=1080&user=alice&pass=qwerty",
		buildTelegramSocks5Deeplink("https://proxy.example.com/", 1080, "alice", "qwerty"),
	)
	assert.Equal(
		t,
		"tg://socks?server=proxy.example.com&port=1080&user=alice&pass=qwerty",
		buildTelegramSocks5Deeplink("https://proxy.example.com:8443/path/", 1080, "alice", "qwerty"),
	)
	assert.Equal(
		t,
		"tg://socks?server=proxy.example.com&port=1080&user=user+name&pass=p%40ss%26word",
		buildTelegramSocks5Deeplink("proxy.example.com", 1080, "user name", "p@ss&word"),
	)
}

func TestHandler_Handle_DeleteUserEnterUsername(t *testing.T) {
	t.Parallel()

	baseState := &store.UserState{State: store.StateDeleteUserEnterUsername}

	t.Run("username check error", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(baseState, nil)
		s.IsUsernameFreeMock.Return(false, errors.New("lookup failed"))

		c := &contextStub{text: "proxy-user", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "lookup failed")
	})

	t.Run("user not found", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(baseState, nil)
		s.IsUsernameFreeMock.Return(true, nil)

		c := &contextStub{text: "proxy-user", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Equal(t, "User with provided username does not exists. Enter another one.", c.sendCall[0].msg)
	})

	t.Run("delete error", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(baseState, nil)
		s.IsUsernameFreeMock.Return(false, nil)
		s.DeleteUserMock.Return(errors.New("delete failed"))

		c := &contextStub{text: "proxy-user", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "delete failed")
	})

	t.Run("set state error", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(baseState, nil)
		s.IsUsernameFreeMock.Return(false, nil)
		s.DeleteUserMock.Return(nil)
		s.SetUserStateMock.Return(errors.New("state failed"))

		c := &contextStub{text: "proxy-user", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.Error(t, err)
		assert.EqualError(t, err, "state failed")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		s := NewStoreIMock(t)
		s.GetUserStateMock.Return(baseState, nil)
		s.IsUsernameFreeMock.Return(false, nil)
		s.DeleteUserMock.Return(nil)
		s.SetUserStateMock.Return(nil)

		c := &contextStub{text: "proxy-user", sender: &tele.User{Username: "admin"}}

		h := New(s)
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sendCall, 1)
		assert.Equal(t, "User deleted.", c.sendCall[0].msg)
	})
}

func TestGenerateSuggestedPassword(t *testing.T) {
	t.Parallel()

	pass, err := generateSuggestedPassword()
	require.NoError(t, err)
	assert.Len(t, pass, 10)
}
