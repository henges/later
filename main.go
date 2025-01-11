package main

import (
	"context"
	"encoding/json"
	"github.com/henges/later/app"
	"github.com/henges/later/bot"
	"github.com/henges/later/later"
	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
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
	w := setupWhen()
	cmds := bot.Commands{
		app.NewSetReminderCommand(l, w),
		app.NewListRemindersCommand(l, w),
		app.NewDeleteReminderCommand(l, w),
	}
	cmds = append(cmds, app.NewHelpCommand(cmds))
	webhookBot, err := bot.NewWebhookBot(&conf, cmds)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	webhookBot.Start()
	err = app.StartPolling(l, webhookBot.GetBot())
	if err != nil {
		log.Fatal().Err(err).Send()
		return
	}
	defer l.StopPoll()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	log.Info().Msg("App ready")

	<-ctx.Done()
	stop()
	err = webhookBot.Stop()
	log.Info().Err(err).Msg("App shutdown")
}

func setupWhen() *when.Parser {
	w := when.New(nil)
	w.Add(en.All...)
	w.Add(common.All...)
	return w
}
