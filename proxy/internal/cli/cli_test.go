package cli

//go:generate minimock -g -i github.com/nskondratev/socks5-proxy-server/proxy/internal/cli.redis -o redis_mock_test.go -n RedisMock -p cli

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleCLICommand(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() {
		os.Args = oldArgs
	})

	t.Run("no args", func(t *testing.T) {
		os.Args = []string{"app"}
		handled := HandleCLICommand(context.Background(), &CommandsDeps{Redis: NewRedisMock(t)})
		assert.False(t, handled)
	})

	t.Run("empty command", func(t *testing.T) {
		os.Args = []string{"app", ""}
		handled := HandleCLICommand(context.Background(), &CommandsDeps{Redis: NewRedisMock(t)})
		assert.False(t, handled)
	})

	t.Run("unknown command", func(t *testing.T) {
		t.Setenv("LOG_LEVEL", "warning")
		os.Args = []string{"app", "unknown"}
		handled := HandleCLICommand(context.Background(), &CommandsDeps{Redis: NewRedisMock(t)})
		assert.True(t, handled)
	})
}
