package bot

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	gobot "github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/rs/zerolog/log"
)

type WebhookBot struct {
	c          *Config
	b          *gotgbot.Bot
	dispatcher *gobot.Dispatcher
	updater    *gobot.Updater
	cmds       Commands
}

type Config struct {
	ListenPort   int    `json:"listenPort"`
	Host         string `json:"host"`
	UrlPath      string `json:"urlPath"`
	AuthToken    string `json:"authToken"`
	SharedSecret string `json:"sharedSecret"`
}

type Command struct {
	gotgbot.BotCommand
	Func handlers.Response
}

func NewWebhookBot(c *Config, cmds Commands) (*WebhookBot, error) {

	bot, err := gotgbot.NewBot(c.AuthToken, nil)
	if err != nil {
		return nil, err
	}

	dispatcher := gobot.NewDispatcher(&gobot.DispatcherOpts{
		// If an error is returned by a handler, log it and continue going.
		Error: func(b *gotgbot.Bot, ctx *gobot.Context, err error) gobot.DispatcherAction {
			log.Err(err).Msg("error occurred while handling update")
			return gobot.DispatcherActionNoop
		},
		MaxRoutines: gobot.DefaultMaxRoutines,
	})
	for _, v := range cmds {
		dispatcher.AddHandler(handlers.NewCommand(v.Command, v.Func))
	}
	updater := gobot.NewUpdater(dispatcher, nil)
	err = updater.AddWebhook(bot, c.UrlPath, &gobot.AddWebhookOpts{SecretToken: c.SharedSecret})
	if err != nil {
		return nil, err
	}

	return &WebhookBot{b: bot, dispatcher: dispatcher, updater: updater, c: c, cmds: cmds}, nil
}

type Commands []Command

func CommandsEqual(v1 []Command, v2 []gotgbot.BotCommand) bool {

	if len(v1) != len(v2) {
		return false
	}
	for i, v := range v1 {
		if v2[i] != v.BotCommand {
			return false
		}
	}

	return true
}

func (c Commands) GetGobotCommands() []gotgbot.BotCommand {

	ret := make([]gotgbot.BotCommand, len(c))
	for i, e := range c {
		ret[i] = e.BotCommand
	}
	return ret
}

func (b *WebhookBot) Start() error {
	oldCommands, err := b.b.GetMyCommands(nil)
	if err != nil {
		return err
	}
	if !CommandsEqual(b.cmds, oldCommands) {

		ok, err := b.b.SetMyCommands(b.cmds.GetGobotCommands(), nil)
		if err != nil {
			return err
		}
		if !ok {
			log.Error().Msg("Non ok result when trying to update commands")
			return nil
		}
		log.Info().Msg("Updated commands")
	}

	err = b.updater.StartServer(gobot.WebhookOpts{ListenAddr: fmt.Sprintf("0.0.0.0:%d", b.c.ListenPort), SecretToken: b.c.SharedSecret})
	if err != nil {
		return err
	}
	return b.updater.SetAllBotWebhooks(b.c.Host, &gotgbot.SetWebhookOpts{SecretToken: b.c.SharedSecret})
}

func (b *WebhookBot) Stop() error {

	return b.updater.Stop()
}

func (b *WebhookBot) GetBot() *gotgbot.Bot {
	return b.b
}
