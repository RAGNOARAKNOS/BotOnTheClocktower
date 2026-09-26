package main

import (
	"log"

	"github.com/RAGNOARAKNOS/BotOnTheClocktower/internal/bot"
	"github.com/RAGNOARAKNOS/BotOnTheClocktower/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := bot.Run(cfg.Token); err != nil {
		log.Fatal(err)
	}
}
