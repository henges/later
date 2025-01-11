package app

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	gobot "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/henges/later/bot"
	"strings"
)

func formatHelpMessage(cmds []bot.Command) string {

	var sb strings.Builder
	for _, cmd := range cmds {
		text := "*/" + cmd.Command + "*" + " " + cmd.Description + "\n" + strings.ReplaceAll(strings.TrimSpace(cmd.LongDescription), "\n", " ") + "\n\n"
		sb.WriteString(text)
	}

	cmdDescriptions := strings.TrimSpace(sb.String())
	return cmdDescriptions
}

func NewHelpCommand(cmds []bot.Command) bot.Command {

	v := &Help{formatHelpMessage(cmds)}
	return bot.Command{
		BotCommand: gotgbot.BotCommand{
			Command:     "help",
			Description: "Show help",
		},
		Func: v.Response,
	}
}

type Help struct {
	helpMsg string
}

func (h *Help) Response(b *gotgbot.Bot, ctx *gobot.Context) error {
	replyTo := ctx.EffectiveChat.Id
	err := sendMessage(b, replyTo, h.helpMsg)
	if err != nil {
		return err
	}
	return nil
}
