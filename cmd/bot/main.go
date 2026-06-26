package main

import (
	"log"

	"github.com/RAGNOARAKNOS/BotOnTheClocktower/internal/bot"
	"github.com/RAGNOARAKNOS/BotOnTheClocktower/internal/config"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := bot.Run(settings); err != nil {
		log.Fatal(err)
	}
}
