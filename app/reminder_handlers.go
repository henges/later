package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	gobot "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/henges/later/later"
	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
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

func (h *SetReminder) parseTimeString(s string) (time.Time, error) {
	tz, err := time.LoadLocation("Australia/Perth")
	if err != nil {
		return time.Time{}, err
	}

	parse, err := h.w.Parse(s, time.Now().Truncate(time.Second).In(tz))
	if err != nil {
		return time.Time{}, err
	}
	if parse == nil {
		return time.Time{}, errors.New("no match found for text")
	}

	return parse.Time, nil
}

type TelegramCallbackData struct {
	Name    string `json:"name"`
	ReplyTo int64  `json:"replyTo"`
}

// /setreminder tomorrow 4:00pm = do the dishes
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

func NewSetReminderCommand(l *later.Later) *SetReminder {
	w := when.New(nil)
	w.Add(en.All...)
	w.Add(common.All...)

	return &SetReminder{l, w}
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
	_, err = b.SendMessage(replyTo, fmt.Sprintf("@%s, I'll remind you about _%s_ at %s.", user, cbd.Name, reminder.FireTime.Format(time.RFC3339)), nil)
	if err != nil {
		return err
	}
	return nil
}

func StartPolling(l *later.Later, b *gotgbot.Bot) error {

	return l.StartPoll(func(reminder later.Reminder) {

		var cbd TelegramCallbackData
		err := json.Unmarshal([]byte(reminder.CallbackData), &cbd)
		if err != nil {
			log.Err(err).Str("data", reminder.CallbackData).Msg("invalid callback data")
			return
		}
		_, err = b.SendMessage(cbd.ReplyTo, cbd.Name, nil)
		if err != nil {
			log.Err(err).Msg("failed sending message")
			return
		}
	}, time.Second)
}
