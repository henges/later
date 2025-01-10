package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	gobot "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/henges/later/later"
	"github.com/rs/zerolog/log"
	"strings"
	"time"
)

var ErrNoCmd = errors.New("no command found")

func stripCmd(s string) (string, error) {

	split := strings.SplitN(s, " ", 2)
	if len(split) < 2 {
		return "", fmt.Errorf("for message '%s', split len was %d: %w", s, len(split), ErrNoCmd)
	}
	if split[0][0] != '/' {
		return "", fmt.Errorf("for message '%s', split first character wasn't '/': %w", s, ErrNoCmd)
	}
	return split[1], nil
}

var ErrInvalidCmd = errors.New("command wasn't valid")

func parseTimeString(s string) (time.Time, error) {

	return time.Now().Add(10 * time.Second), nil // todo
}

type TelegramCallbackData struct {
	Name    string `json:"name"`
	ReplyTo int64  `json:"replyTo"`
}

// /setreminder tomorrow 4:00pm = do the dishes
func setReminderCommandFromMsgContext(ctx *gobot.Context) (later.Reminder, error) {

	s, err := stripCmd(ctx.EffectiveMessage.Text)
	if err != nil {
		return later.Reminder{}, err
	}
	split := strings.SplitN(s, "=", 2)
	if len(split) != 2 {
		return later.Reminder{}, fmt.Errorf("for message %s, no equals sign: %w", s, ErrInvalidCmd)
	}
	timeString, name := strings.TrimSpace(split[0]), strings.TrimSpace(split[1])
	t, err := parseTimeString(timeString)
	if err != nil {
		return later.Reminder{}, fmt.Errorf("for message %s, couldn't parse time string: %w", s, ErrInvalidCmd)
	}
	cbd := TelegramCallbackData{
		Name:    name,
		ReplyTo: ctx.EffectiveChat.Id,
	}
	cbds, err := json.Marshal(cbd)
	if err != nil {
		return later.Reminder{}, err
	}

	return later.Reminder{
		Owner:        ctx.EffectiveSender.User.Username,
		FireTime:     t,
		CallbackData: string(cbds),
	}, nil
}

func NewSetReminderCommand(l *later.Later) *SetReminder {
	return &SetReminder{l}
}

type SetReminder struct {
	l *later.Later
}

func (h *SetReminder) Response(b *gotgbot.Bot, ctx *gobot.Context) error {
	log.Info().Msg("HELLO!!!")

	message := ctx.EffectiveMessage.Text
	user := ctx.EffectiveSender.User.Username
	replyTo := ctx.EffectiveChat.Id

	logger := log.With().
		Str("messageBody", message).
		Str("username", user).
		Logger()

	logger.Trace().Msg("Handle update")

	var err error
	reminder, err := setReminderCommandFromMsgContext(ctx)
	if err != nil {
		_, err2 := b.SendMessage(replyTo, err.Error(), nil)
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
	_, err = b.SendMessage(replyTo, reminder.Owner+" "+reminder.FireTime.String()+" "+reminder.CallbackData, nil)
	if err != nil {
		return err
	}
	return nil
}
