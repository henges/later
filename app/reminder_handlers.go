package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	gobot "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/henges/later/later"
	"github.com/olebedev/when"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
	"sync"
	"time"
)

var replacer = strings.NewReplacer("-", "\\-", "(", "\\(", ")", "\\)", ".", "\\.", "+", "\\+")

var defLoc *time.Location

var locOnce sync.Once

func tz() *time.Location {

	if defLoc == nil {
		locOnce.Do(func() {
			var err error
			defLoc, err = time.LoadLocation("Australia/Perth")
			if err != nil {
				panic(err)
			}
		})
	}
	return defLoc
}

func escapeMarkdownV2(text string) string {

	return replacer.Replace(text)
}

func sendMessage(b *gotgbot.Bot, replyTo int64, text string) error {

	text = escapeMarkdownV2(text)
	_, err := b.SendMessage(replyTo, text, &gotgbot.SendMessageOpts{
		ParseMode: "MarkdownV2",
	})
	return err
}

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

func NewSetReminderCommand(l *later.Later, w *when.Parser) *SetReminder {
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
	err = sendMessage(b, replyTo, fmt.Sprintf("@%s, I'll remind you about __%s__ at %s.",
		user, cbd.Name, reminder.FireTime.Format(time.RFC3339)))
	if err != nil {
		return err
	}
	return nil
}

func NewListRemindersCommand(l *later.Later, w *when.Parser) *ListReminders {
	return &ListReminders{l, w}
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

func formatReminderList(rmds []later.SavedReminder) string {

	var sb strings.Builder
	for i, rmd := range rmds {
		if i > 0 {
			sb.WriteString("\n")
		}
		var tgcd TelegramCallbackData
		if err := json.Unmarshal([]byte(rmd.CallbackData), &tgcd); err != nil {
			continue
		}

		sb.WriteString(fmt.Sprintf("%d: __%s__ at %s", rmd.ID, tgcd.Name, rmd.FireTime.In(tz()).Format(time.RFC3339)))
	}

	return sb.String()
}

func NewDeleteReminderCommand(l *later.Later, w *when.Parser) *DeleteReminder {
	return &DeleteReminder{l, w}
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
