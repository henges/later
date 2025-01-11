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

var botDescription = makeSingleLine(`
This bot allows you to set reminders. Use /set to give it a time and a message, and it'll
message this chat at that time with your message.
`)

func makeSingleLine(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
}

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

var replacer = strings.NewReplacer(
	"-", "\\-",
	"(", "\\(",
	")", "\\)",
	".", "\\.",
	"+", "\\+",
	"<", "\\<",
	">", "\\>",
	"=", "\\=",
	"!", "\\!")

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
const kitchenHoursOnly = "3PM"

func kitchenFormat(future time.Time) string {

	if future.Second() == 0 {
		if future.Minute() == 0 {
			return future.Format(kitchenHoursOnly)
		}
		return future.Format(time.Kitchen)
	}
	return future.Format(kitchenSeconds)
}

func getTimeDisplayString(now, future time.Time) string {

	dayDiff := dayDifference(now, future)
	clockFmt := kitchenFormat(future)
	if dayDiff == 0 {
		return "today at " + clockFmt
	} else if dayDiff == 1 {
		return "tomorrow at " + clockFmt
	} else if dayDiff <= 7 {
		return fmt.Sprintf("in %d days at %s", dayDiff, clockFmt)
	} else {
		return fmt.Sprintf("on %s at %s", future.Format(time.DateOnly), clockFmt)
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
		sb.WriteString(fmt.Sprintf("%d: __%s__, %s", rmd.ID, tgcd.Name, getTimeDisplayString(referenceTime, timeWZone)))
	}

	return sb.String()
}

func getReminderMessage(owner, name string) string {

	return fmt.Sprintf("@%s, you asked me to remind you about this at this time:\n%s", owner, name)
}

func StartPolling(l *later.Later, b *gotgbot.Bot) error {

	return l.StartPoll(func(reminder later.Reminder) {

		var cbd TelegramCallbackData
		err := json.Unmarshal([]byte(reminder.CallbackData), &cbd)
		if err != nil {
			log.Err(err).Str("data", reminder.CallbackData).Msg("invalid callback data")
			return
		}
		err = sendMessage(b, cbd.ReplyTo, getReminderMessage(reminder.Owner, cbd.Name))
		if err != nil {
			log.Err(err).Msg("failed sending message")
			return
		}
	}, time.Second)
}
