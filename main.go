package main

import (
	"context"
	"encoding/json"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/henges/later/app"
	"github.com/henges/later/bot"
	"github.com/henges/later/later"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	file, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	conf := bot.Config{}
	err = json.Unmarshal(file, &conf)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if conf.ListenPort == 0 {
		conf.ListenPort = 23150
	}

	l, err := later.NewLater()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	cmds := bot.Commands{{
		BotCommand: gotgbot.BotCommand{
			Command:     "set",
			Description: "Set a reminder",
		},
		Func: app.NewSetReminderCommand(l).Response,
	}}

	webhookBot, err := bot.NewWebhookBot(&conf, cmds)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	webhookBot.Start()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	log.Info().Msg("App ready")

	<-ctx.Done()
	stop()
	err = webhookBot.Stop()
	log.Info().Err(err).Msg("App shutdown")
}
