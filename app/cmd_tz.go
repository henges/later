package app

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	gobot "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/henges/later/later"
	"github.com/olebedev/when"
	"github.com/rs/zerolog/log"
	"strconv"
)

func NewSetTzCommand(l *later.Later, w *when.Parser) *DeleteReminder {
	return &DeleteReminder{l, w}
}

type SetTz struct {
	l *later.Later
	w *when.Parser
}

func (h *SetTz) setTzCommandFromContext(ctx *gobot.Context) (string, error) {

	s, err := stripCmd(ctx.EffectiveMessage.Text)
	if err != nil {
		return "", err
	}
	return s, nil
}

func (h *SetTz) Response(b *gotgbot.Bot, ctx *gobot.Context) error {
	message := ctx.EffectiveMessage.Text
	user := ctx.EffectiveSender.User.Username
	replyTo := ctx.EffectiveChat.Id

	logger := log.With().
		Str("messageBody", message).
		Str("username", user).
		Logger()

	logger.Trace().Msg("Handle update")

	tz, err := h.setTzCommandFromContext(ctx)
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
