package app

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	gobot "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/henges/later/bot"
	"github.com/henges/later/later"
	"github.com/olebedev/when"
	"github.com/rs/zerolog/log"
)

func NewListRemindersCommand(l *later.Later, w *when.Parser) bot.Command {
	v := &ListReminders{l, w}
	return bot.Command{
		BotCommand: gotgbot.BotCommand{
			Command:     "list",
			Description: "List reminders",
		},
		LongDescription: `
List all reminders you have registered. The ID associated with each returned
reminder can be used to delete a reminder if desired.
		`,
		Func: v.Response,
	}
}

type ListReminders struct {
	l *later.Later
	w *when.Parser
}

func (h *ListReminders) Response(b *gotgbot.Bot, ctx *gobot.Context) error {
	message := ctx.EffectiveMessage.Text
	user := ctx.EffectiveSender.User.Username
	replyTo := ctx.EffectiveChat.Id

	logger := log.With().
		Str("messageBody", message).
		Str("username", user).
		Logger()

	logger.Trace().Msg("Handle update")

	rmds, err := h.l.GetRemindersByOwner(user)
	if err != nil {
		logger.Err(err).Send()
		return err
	}
	if len(rmds) == 0 {
		err = sendMessage(b, replyTo, fmt.Sprintf("@%s, you don't currently have any reminders (time to make some).", user))
		return err
	}
	resp := fmt.Sprintf("@%s, here are your saved reminders:\n%s", user, formatReminderList(rmds))
	err = sendMessage(b, replyTo, resp)
	if err != nil {
		return err
	}
	return nil
}
