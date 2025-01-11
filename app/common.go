package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/henges/later/later"
	"github.com/rs/zerolog/log"
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

type TelegramCallbackData struct {
	Name    string `json:"name"`
	ReplyTo int64  `json:"replyTo"`
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
