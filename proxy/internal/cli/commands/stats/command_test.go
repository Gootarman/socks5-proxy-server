package stats

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/services/users"
)

type usersMock struct {
	stats    []users.Stat
	statsErr error
}

func (m *usersMock) GetStats(_ context.Context) ([]users.Stat, error) {
	return m.stats, m.statsErr
}

func TestCommandHandler_Handle(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	h := New(&usersMock{stats: []users.Stat{
		{Username: "alice", UsageBytes: 1025, Usage: "1.00 KB", LastAuth: "2026-01-01T10:00:00Z"},
		{Username: "bob", UsageBytes: 256, Usage: "256 B"},
	}}, buf)

	err := h.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "alice") || !strings.Contains(out, "bob") {
		t.Fatalf("expected users to be printed, got %q", out)
	}

	if strings.Index(out, "alice") > strings.Index(out, "bob") {
		t.Fatalf("expected alice to be listed before bob, got %q", out)
	}

	if !strings.Contains(out, "1.00 KB") {
		t.Fatalf("expected human readable usage for alice, got %q", out)
	}

	if !strings.Contains(out, "-") {
		t.Fatalf("expected missing last login fallback, got %q", out)
	}
}

func TestCommandHandler_HandleUsersServiceError(t *testing.T) {
	t.Parallel()

	h := New(&usersMock{statsErr: errors.New("boom")}, bytes.NewBuffer(nil))

	err := h.Handle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
