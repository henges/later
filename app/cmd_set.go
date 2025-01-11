package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	gobot "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/henges/later/bot"
	"github.com/henges/later/later"
	"github.com/olebedev/when"
	"github.com/rs/zerolog/log"
	"strings"
	"time"
)

func NewSetReminderCommand(l *later.Later, w *when.Parser) bot.Command {
	v := &SetReminder{l, w}

	return bot.Command{
		BotCommand: gotgbot.BotCommand{
			Command:     "set",
			Description: "<time string> = <description> - Set a reminder",
		},
		LongDescription: `
Set a reminder that will fire at the time specified by the given time string.
You can use date-time values like '2025-01-11' and '2025-01-11T11:39:00', as
well as conversational values like 'tomorrow', 'in three days', etc.
		`,
		Func: v.Response,
	}
}

type SetReminder struct {
	l *later.Later
	w *when.Parser
}

func (h *SetReminder) Response(b *gotgbot.Bot, ctx *gobot.Context) error {
	message := ctx.EffectiveMessage.Text
	user := ctx.EffectiveSender.User.Username
	replyTo := ctx.EffectiveChat.Id

	logger := log.With().
		Str("messageBody", message).
		Str("username", user).
		Logger()

	logger.Trace().Msg("Handle update")

	var err error
	reminder, cbd, err := h.setReminderCommandFromMsgContext(ctx)
	if err != nil {
		err2 := sendMessage(b, replyTo, err.Error())
		if err2 != nil {
			return err2
		}
		logger.Err(err).Send()
		return nil
	}
	err = h.l.InsertReminder(reminder)
	if err != nil {
		logger.Err(err).Send()
		return err
	}
	now := time.Now().In(tz())
	err = sendMessage(b, replyTo, fmt.Sprintf("@%s, I'll remind you about __%s__ %s.",
		user, cbd.Name, getTimeDisplayString(now, reminder.FireTime)))
	if err != nil {
		return err
	}
	return nil
}

func (h *SetReminder) parseTimeString(s string) (time.Time, error) {
	// some cases that 'when' doesn't get
	specialCases := []string{time.DateOnly, time.RFC3339, "2006-01-02T15:04:05"}
	for _, layout := range specialCases {
		specialCase, err := time.ParseInLocation(layout, s, tz())
		if err == nil {
			return specialCase, nil
		}
	}

	parse, err := h.w.Parse(s, time.Now().Truncate(time.Second).In(tz()))
	if err != nil {
		return time.Time{}, err
	}
	if parse == nil {
		return time.Time{}, errors.New("no match found for text")
	}

	return parse.Time, nil
}

// /set tomorrow 4:00pm = do the dishes
func (h *SetReminder) setReminderCommandFromMsgContext(ctx *gobot.Context) (later.Reminder, TelegramCallbackData, error) {

	s, err := stripCmd(ctx.EffectiveMessage.Text)
	if err != nil {
		return later.Reminder{}, TelegramCallbackData{}, err
	}
	split := strings.SplitN(s, "=", 2)
	if len(split) != 2 {
		return later.Reminder{}, TelegramCallbackData{}, fmt.Errorf("for message %s, no equals sign: %w", s, ErrInvalidCmd)
	}
	timeString, name := strings.TrimSpace(split[0]), strings.TrimSpace(split[1])
	t, err := h.parseTimeString(timeString)
	if err != nil {
		return later.Reminder{}, TelegramCallbackData{}, fmt.Errorf("for message %s, couldn't parse time string: %w", s, ErrInvalidCmd)
	}
	cbd := TelegramCallbackData{
		Name:    name,
		ReplyTo: ctx.EffectiveChat.Id,
	}
	cbds, err := json.Marshal(cbd)
	if err != nil {
		return later.Reminder{}, TelegramCallbackData{}, err
	}

	return later.Reminder{
		Owner:        ctx.EffectiveSender.User.Username,
		FireTime:     t,
		CallbackData: string(cbds),
	}, cbd, nil
}
