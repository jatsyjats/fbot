// Main bot structure

package main

import (
	"database/sql"
	"github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
	"os"
	"os/signal"
	"syscall"
)

type Config struct {
	DiscordToken       string
	DbConnectionString string

	VerificationSystem *VerificationConfig
}

type Module interface {
	Register(*Bot)
}

type Bot struct {
	Discord *Discord
	DB      *sql.DB
	Modules []Module
}

func NewBot(config *Config) (*Bot, error) {
	Logf("Initializing bot instance")

	discord, err := NewDiscord(config.DiscordToken)
	if err != nil {
		return nil, err
	}

	modules := []Module{
		NewVerificationModule(config.VerificationSystem),
	}

	bot := &Bot{
		Discord: discord,
		Modules: modules,
	}
	Logf("Initialization done")
	return bot, nil
}

func (bot *Bot) Run() error {
	bot.Discord.Identify.Intents = discordgo.IntentsAll

	for _, module := range bot.Modules {
		module.Register(bot)
	}

	err := bot.Discord.Open()
	if err != nil {
		return WrapError(err)
	}
	Logf("Connected!")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	Logf("Exiting ...")
	bot.Discord.Close()
	return nil
}
