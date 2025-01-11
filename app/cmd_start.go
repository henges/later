package app

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	gobot "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/henges/later/bot"
)

func NewStartCommand() bot.Command {

	v := &Start{}
	return bot.Command{
		BotCommand: gotgbot.BotCommand{
			Command:     "start",
			Description: "Start bot interactions",
		},
		Func: v.Response,
	}
}

type Start struct{}

var startMsg = "Hi! " + botDescription + "\n" + "Use /help for more details."

func (h *Start) Response(b *gotgbot.Bot, ctx *gobot.Context) error {
	replyTo := ctx.EffectiveChat.Id

	err := sendMessage(b, replyTo, startMsg)
	if err != nil {
		return err
	}
	return nil
}
