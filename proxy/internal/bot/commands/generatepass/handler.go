package generatepass

import (
	"strconv"
	"strings"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/password"
	tele "gopkg.in/telebot.v3"
)

const Command = "/generate_pass"

const (
	defaultLen = 10
)

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Handle(c tele.Context) error {
	ln := defaultLen

	if payload := strings.TrimSpace(c.Message().Payload); payload != "" {
		if n, err := strconv.Atoi(payload); err == nil && n > 0 {
			ln = n
		}
	}

	pass, err := password.Generate(ln)
	if err != nil {
		return err
	}

	return c.Send(pass)
}
