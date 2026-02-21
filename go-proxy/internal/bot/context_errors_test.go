package bot

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tele "gopkg.in/telebot.v3"
)

type contextStub struct {
	tele.Context
	values   map[string]interface{}
	replyErr error
	replies  []string
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

func (c *contextStub) Reply(what interface{}, _ ...interface{}) error {
	c.replies = append(c.replies, fmt.Sprint(what))
	return c.replyErr
}

func TestSetGetContext(t *testing.T) {
	t.Parallel()

	c := &contextStub{}
	expected := context.WithValue(context.Background(), "k", "v")

	SetContext(c, expected)
	got := GetContext(c)

	assert.Equal(t, expected, got)
}

func TestGetContextFallbacks(t *testing.T) {
	t.Parallel()

	assert.Equal(t, context.Background(), GetContext(nil))

	c := &contextStub{values: map[string]interface{}{contextFieldContext: "bad"}}
	assert.Equal(t, context.Background(), GetContext(c))
}

func TestOnErrorCb(t *testing.T) {
	t.Parallel()

	t.Run("nil context does not panic", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			OnErrorCb(errors.New("boom"), nil)
		})
	})

	t.Run("reply message is sent", func(t *testing.T) {
		t.Parallel()

		c := &contextStub{}
		OnErrorCb(errors.New("boom"), c)

		require.Len(t, c.replies, 1)
		assert.Equal(t, "Some error occurred, check server logs for details.", c.replies[0])
	})

	t.Run("reply error is handled", func(t *testing.T) {
		t.Parallel()

		c := &contextStub{replyErr: errors.New("send failed")}
		assert.NotPanics(t, func() {
			OnErrorCb(errors.New("boom"), c)
		})
	})
}
