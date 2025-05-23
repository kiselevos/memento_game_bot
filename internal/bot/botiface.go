package bot

import (
	"gopkg.in/telebot.v3"
	tb "gopkg.in/telebot.v3"
)

var _ BotInterface = (*telebot.Bot)(nil)

type BotInterface interface {
	Send(to tb.Recipient, what interface{}, options ...interface{}) (*tb.Message, error)
	Delete(msg tb.Editable) error
	Handle(endpoint interface{}, handler telebot.HandlerFunc, middlwear ...telebot.MiddlewareFunc)
	Respond(c *tb.Callback, resp ...*tb.CallbackResponse) error
}
