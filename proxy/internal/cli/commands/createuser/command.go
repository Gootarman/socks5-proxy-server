package createuser

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/cli/commands/common"
	"github.com/nskondratev/socks5-proxy-server/proxy/internal/services/users"
)

const (
	command = "create-user"
)

type userService interface {
	Create(ctx context.Context, username, password string) error
}

type CommandHandler struct {
	users userService
	in    *bufio.Reader
	out   io.Writer
}

func New(users userService, in io.Reader, out io.Writer) *CommandHandler {
	if in == nil {
		in = os.Stdin
	}

	if out == nil {
		out = os.Stdout
	}

	return &CommandHandler{users: users, in: bufio.NewReader(in), out: out}
}

func (h *CommandHandler) CanHandle(_ context.Context, commandName string) bool {
	return commandName == command
}

func (h *CommandHandler) Handle(ctx context.Context) error {
	if h.users == nil {
		return fmt.Errorf("[create-user] user service dependency is not configured")
	}

	username, err := common.PromptAndReadRequiredInput(h.out, h.in, "Input username and press Enter: ", "username")
	if err != nil {
		return fmt.Errorf("[create-user] failed to read username: %w", err)
	}

	password, err := common.PromptAndReadRequiredInput(
		h.out,
		h.in,
		"Input password and press Enter to create new user: ",
		"password",
	)
	if err != nil {
		return fmt.Errorf("[create-user] failed to read password: %w", err)
	}

	if err = h.users.Create(ctx, username, password); err != nil {
		if errors.Is(err, users.ErrUserExists) {
			return fmt.Errorf("[create-user] %w", users.ErrUserExists)
		}

		return fmt.Errorf("[create-user] failed to create user: %w", err)
	}

	if err = common.WriteSuccess(h.out, "User successfully created."); err != nil {
		return fmt.Errorf("[create-user] %w", err)
	}

	return nil
}
