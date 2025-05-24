package middleware

import (
	"time"

	"gopkg.in/telebot.v3"
)

// Мидлварь для обработки longpoolinga
func DropOldMessages(maxAge time.Duration) *telebot.MiddlewarePoller {
	return telebot.NewMiddlewarePoller(
		&telebot.LongPoller{Timeout: 10 * time.Second},
		func(u *telebot.Update) bool {
			if u.Message != nil && time.Since(u.Message.Time()) > maxAge {
				return false
			}
			return true
		})
}
