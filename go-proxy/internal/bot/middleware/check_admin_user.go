package middleware

import (
	tele "gopkg.in/telebot.v3"
)

func RestrictByAdminUserID() func(tele.HandlerFunc) tele.HandlerFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			sender := c.Sender()
			if sender == nil {
				return nil
			}

			// TODO: здесь реализовать проверку на то, что id пользователя находится в списке админов в Redis
			// if _, ok := allowedUserIDsMap[sender.ID]; !ok {
			// return c.Send("Sorry, this functionality is available only for admin users.")
			// }

			return next(c)
		}
	}
}
