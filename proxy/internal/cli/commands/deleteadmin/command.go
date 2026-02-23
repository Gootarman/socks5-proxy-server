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

	username, err := common.PromptAndReadRequiredInput(h.out, h.in, "Input admin username and press Enter: ", "username")
	if err != nil {
		return fmt.Errorf("[delete-admin] failed to read username: %w", err)
	}

	if err = h.adminService.Remove(ctx, username); err != nil {
		return fmt.Errorf("[delete-admin] failed to delete admin: %w", err)
	}

	if err = common.WriteSuccess(h.out, "Admin successfully deleted."); err != nil {
		return fmt.Errorf("[delete-admin] %w", err)
	}

	return nil
}
