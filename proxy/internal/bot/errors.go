package bot

import (
	tele "gopkg.in/telebot.v3"

	"github.com/nskondratev/socks5-proxy-server/proxy/internal/log"
)

func OnErrorCb(err error, c tele.Context) {
	log.Error(
		GetContext(c),
		"error while processing update",
		log.String(log.FieldError, err.Error()),
	)

	if c == nil {
		return
	}

	replyErr := c.Reply("Some error occurred, check server logs for details.")
	if replyErr != nil {
		log.Error(
			GetContext(c),
			"failed to send error reply message",
			log.String(log.FieldError, err.Error()),
		)
	}
}
