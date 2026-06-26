package config

import (
	"os"

	"github.com/RAGNOARAKNOS/BotOnTheClocktower/internal/bot"
	"github.com/joho/godotenv"
)

func Load() (bot.Settings, error) {
	var settings bot.Settings

	if err := godotenv.Load(); err != nil {
		return settings, err
	}

	settings.ApiToken = os.Getenv("BOTAPIKEY")
	settings.GuildId = "UNSET"
	settings.ChannelId = "UNSET"
	settings.StoryTellerId = "UNSET"
	settings.GameRegistered = false

	return settings, nil
}
