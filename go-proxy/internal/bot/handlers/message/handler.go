package message

import (
	tele "gopkg.in/telebot.v3"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Handle(c tele.Context) error {
	// TODO: implement logic here
	panic("not implemented")
}
