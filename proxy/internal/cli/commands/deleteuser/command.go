package deleteuser

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/cli/commands/common"
)

const (
	command = "delete-user"
)

type userService interface {
	Delete(ctx context.Context, username string) error
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
		return fmt.Errorf("[delete-user] user service dependency is not configured")
	}

	if _, err := fmt.Fprint(h.out, "Input username and press Enter: "); err != nil {
		return fmt.Errorf("[delete-user] failed to write prompt: %w", err)
	}

	username, err := h.readInputLine()
	if err != nil {
		return fmt.Errorf("[delete-user] failed to read username: %w", err)
	}

	if err = h.users.Delete(ctx, username); err != nil {
		return fmt.Errorf("[delete-user] failed to delete user: %w", err)
	}

	if _, err = fmt.Fprintln(h.out, "User successfully deleted."); err != nil {
		return fmt.Errorf("[delete-user] failed to write success message: %w", err)
	}

	return nil
}

func (h *CommandHandler) readInputLine() (string, error) {
	return common.ReadInputLine(h.in)
}
