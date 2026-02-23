package deleteuser

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type usersMock struct {
	deleteErr      error
	deleteUsername string
}

func (m *usersMock) Delete(_ context.Context, username string) error {
	m.deleteUsername = username

	return m.deleteErr
}

func TestCommandHandler_Handle(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	users := &usersMock{}
	h := New(users, strings.NewReader("alice\n"), buf)

	err := h.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if users.deleteUsername != "alice" {
		t.Fatalf("expected username alice, got %q", users.deleteUsername)
	}

	if !strings.Contains(buf.String(), "User successfully deleted.") {
		t.Fatalf("expected success output, got %q", buf.String())
	}
}

func TestCommandHandler_HandleDeleteError(t *testing.T) {
	t.Parallel()

	h := New(&usersMock{deleteErr: errors.New("boom")}, strings.NewReader("alice\n"), bytes.NewBuffer(nil))

	err := h.Handle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
