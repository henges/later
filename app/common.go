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

var replacer = strings.NewReplacer("-", "\\-", "(", "\\(", ")", "\\)", ".", "\\.", "+", "\\+", "<", "\\<", ">", "\\>", "=", "\\=")

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

func dayDifference(now time.Time, future time.Time) int {

	// Truncate times to midnight
	start := now.Truncate(24 * time.Hour)
	end := future.Truncate(24 * time.Hour)

	// Compute the difference in days
	return int(end.Sub(start).Hours() / 24)
}

const kitchenSeconds = "3:04:05PM"

func getTimeDisplayString(now, future time.Time) string {

	dayDiff := dayDifference(now, future)
	if dayDiff == 0 {
		return "today at " + future.Format(kitchenSeconds)
	} else if dayDiff == 1 {
		return "tomorrow at " + future.Format(kitchenSeconds)
	} else if dayDiff <= 7 {
		return fmt.Sprintf("in %d days at %s", dayDiff, future.Format(kitchenSeconds))
	} else {
		return fmt.Sprintf("on %s at %s", future.Format(time.DateOnly), future.Format(kitchenSeconds))
	}
}

func formatReminderList(rmds []later.SavedReminder) string {

	referenceTime := time.Now().In(tz())
	var sb strings.Builder
	for i, rmd := range rmds {
		if i > 0 {
			sb.WriteString("\n")
		}
		var tgcd TelegramCallbackData
		if err := json.Unmarshal([]byte(rmd.CallbackData), &tgcd); err != nil {
			continue
		}

		timeWZone := rmd.FireTime.In(tz())
		sb.WriteString(fmt.Sprintf("%d: __%s__ %s", rmd.ID, tgcd.Name, getTimeDisplayString(referenceTime, timeWZone)))
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
