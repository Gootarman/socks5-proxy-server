package stats

import "context"

const command = "users-stats"

type CommandHandler struct {
	// TODO: add here needed dependencies
}

func New() *CommandHandler {
	return &CommandHandler{}
}

func (h *CommandHandler) CanHandle(_ context.Context, commandName string) bool {
	return commandName == command
}

func (h *CommandHandler) Handle(ctx context.Context) error {
	// TODO: implement logic here
	panic("not implemented")
}
