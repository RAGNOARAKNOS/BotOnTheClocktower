package config

import (
	"errors"
	"io/fs"
	"os"

	"github.com/RAGNOARAKNOS/BotOnTheClocktower/internal/bot"
	"github.com/joho/godotenv"
)

func Load() (bot.Settings, error) {
	var settings bot.Settings

	// Loading a .env file is optional: a missing file is fine because the
	// configuration (e.g. BOTAPIKEY) can be supplied directly via the
	// environment, such as when running inside a container. Any other load
	// error is unexpected and is surfaced to the caller.
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return settings, err
	}

	settings.ApiToken = os.Getenv("BOTAPIKEY")
	settings.GuildId = "UNSET"
	settings.ChannelId = "UNSET"
	settings.StoryTellerId = "UNSET"
	settings.GameRegistered = false

	return settings, nil
}
