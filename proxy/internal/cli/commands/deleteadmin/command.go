package deleteadmin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/cli/commands/common"
)

const (
	command = "delete-admin"
)

type adminService interface {
	Remove(ctx context.Context, username string) error
}

type CommandHandler struct {
	adminService adminService
	in           *bufio.Reader
	out          io.Writer
}

func New(adminService adminService, in io.Reader, out io.Writer) *CommandHandler {
	if in == nil {
		in = os.Stdin
	}

	if out == nil {
		out = os.Stdout
	}

	return &CommandHandler{adminService: adminService, in: bufio.NewReader(in), out: out}
}

func (h *CommandHandler) CanHandle(_ context.Context, commandName string) bool {
	return commandName == command
}

func (h *CommandHandler) Handle(ctx context.Context) error {
	if h.adminService == nil {
		return fmt.Errorf("[delete-admin] admin service dependency is not configured")
	}

	if _, err := fmt.Fprint(h.out, "Input admin username and press Enter: "); err != nil {
		return fmt.Errorf("[delete-admin] failed to write prompt: %w", err)
	}

	username, err := h.readInputLine()
	if err != nil {
		return fmt.Errorf("[delete-admin] failed to read username: %w", err)
	}

	if err = h.adminService.Remove(ctx, username); err != nil {
		return fmt.Errorf("[delete-admin] failed to delete admin: %w", err)
	}

	if _, err = fmt.Fprintln(h.out, "Admin successfully deleted."); err != nil {
		return fmt.Errorf("[delete-admin] failed to write success message: %w", err)
	}

	return nil
}

func (h *CommandHandler) readInputLine() (string, error) {
	return common.ReadInputLine(h.in)
}
