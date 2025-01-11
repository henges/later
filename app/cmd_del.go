package app

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	gobot "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/henges/later/bot"
	"github.com/henges/later/later"
	"github.com/olebedev/when"
	"github.com/rs/zerolog/log"
	"strconv"
)

func NewDeleteReminderCommand(l *later.Later, w *when.Parser) bot.Command {
	v := &DeleteReminder{l, w}
	return bot.Command{
		BotCommand: gotgbot.BotCommand{
			Command:     "del",
			Description: "<id> - Delete a reminder",
		},
		LongDescription: `
Delete a reminder. The <id> value provided should correspond with a value
returned by /list.
		`,
		Func: v.Response,
	}
}

type DeleteReminder struct {
	l *later.Later
	w *when.Parser
}

func (h *DeleteReminder) deleteReminderCommandFromContext(ctx *gobot.Context) (int64, error) {

	s, err := stripCmd(ctx.EffectiveMessage.Text)
	if err != nil {
		return 0, err
	}
	asint, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return asint, nil
}

func (h *DeleteReminder) Response(b *gotgbot.Bot, ctx *gobot.Context) error {
	message := ctx.EffectiveMessage.Text
	user := ctx.EffectiveSender.User.Username
	replyTo := ctx.EffectiveChat.Id

	logger := log.With().
		Str("messageBody", message).
		Str("username", user).
		Logger()

	logger.Trace().Msg("Handle update")

	id, err := h.deleteReminderCommandFromContext(ctx)
	if err != nil {
		logger.Err(err).Send()
		return err
	}
	didDelete, err := h.l.DeleteReminderWithOwner(user, id)
	if err != nil {
		return err
	}
	if !didDelete {
		err = sendMessage(b, replyTo, fmt.Sprintf("@%s, I couldn't find a reminder with ID %d to delete...", user, id))
		return err
	}

	resp := fmt.Sprintf("@%s, I successfully deleted reminder with ID %d", user, id)
	err = sendMessage(b, replyTo, resp)
	if err != nil {
		return err
	}
	return nil
}
