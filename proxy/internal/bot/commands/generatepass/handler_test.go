package generatepass

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tele "gopkg.in/telebot.v3"
)

type contextStub struct {
	tele.Context
	msg  *tele.Message
	sent []string
}

func (c *contextStub) Message() *tele.Message {
	if c.msg == nil {
		return &tele.Message{}
	}

	return c.msg
}

func (c *contextStub) Send(what interface{}, _ ...interface{}) error {
	c.sent = append(c.sent, fmt.Sprint(what))
	return nil
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	h := New()

	t.Run("default length", func(t *testing.T) {
		t.Parallel()

		c := &contextStub{msg: &tele.Message{Payload: ""}}
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sent, 1)
		assert.Len(t, c.sent[0], defaultLen)
	})

	t.Run("payload custom length", func(t *testing.T) {
		t.Parallel()

		c := &contextStub{msg: &tele.Message{Payload: "14"}}
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sent, 1)
		assert.Len(t, c.sent[0], 14)
	})

	t.Run("invalid payload keeps default", func(t *testing.T) {
		t.Parallel()

		c := &contextStub{msg: &tele.Message{Payload: "bad"}}
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sent, 1)
		assert.Len(t, c.sent[0], defaultLen)
	})

	t.Run("non positive payload keeps default", func(t *testing.T) {
		t.Parallel()

		c := &contextStub{msg: &tele.Message{Payload: "0"}}
		err := h.Handle(c)
		require.NoError(t, err)
		require.Len(t, c.sent, 1)
		assert.Len(t, c.sent[0], defaultLen)
	})
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	pass, err := Generate(1)
	require.NoError(t, err)
	assert.Len(t, pass, 3)
	assert.Regexp(t, "[a-z]", pass)
	assert.Regexp(t, "[A-Z]", pass)
	assert.Regexp(t, "[0-9]", pass)

	pass, err = Generate(16)
	require.NoError(t, err)
	assert.Len(t, pass, 16)
	assert.Regexp(t, "[a-z]", pass)
	assert.Regexp(t, "[A-Z]", pass)
	assert.Regexp(t, "[0-9]", pass)
}
