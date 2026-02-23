package createuser

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/services/users"
)

type usersMock struct {
	createErr      error
	createUsername string
	createPassword string
}

func (m *usersMock) Create(_ context.Context, username, password string) error {
	m.createUsername = username
	m.createPassword = password

	return m.createErr
}

func TestCommandHandler_Handle(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	users := &usersMock{}
	h := New(users, strings.NewReader("alice\nsecret\n"), buf)

	err := h.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if users.createUsername != "alice" {
		t.Fatalf("expected username alice, got %q", users.createUsername)
	}

	if users.createPassword != "secret" {
		t.Fatalf("expected password secret, got %q", users.createPassword)
	}

	if !strings.Contains(buf.String(), "User successfully created.") {
		t.Fatalf("expected success output, got %q", buf.String())
	}
}

func TestCommandHandler_HandleCreateError(t *testing.T) {
	t.Parallel()

	h := New(&usersMock{createErr: errors.New("boom")}, strings.NewReader("alice\nsecret\n"), bytes.NewBuffer(nil))

	err := h.Handle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommandHandler_HandleEmptyUsername(t *testing.T) {
	t.Parallel()

	h := New(&usersMock{}, strings.NewReader("\nsecret\n"), bytes.NewBuffer(nil))

	err := h.Handle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommandHandler_HandleEmptyPassword(t *testing.T) {
	t.Parallel()

	h := New(&usersMock{}, strings.NewReader("alice\n\n"), bytes.NewBuffer(nil))

	err := h.Handle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommandHandler_HandleUserAlreadyExists(t *testing.T) {
	t.Parallel()

	h := New(&usersMock{createErr: users.ErrUserExists}, strings.NewReader("alice\nsecret\n"), bytes.NewBuffer(nil))

	err := h.Handle(context.Background())
	if !errors.Is(err, users.ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}
