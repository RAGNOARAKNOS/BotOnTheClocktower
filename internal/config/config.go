package config

import (
	"errors"
	"io/fs"
	"os"

	"github.com/joho/godotenv"
)

// Config is the bot's configuration, read at startup.
type Config struct {
	Token string // Discord bot token, from BOTAPIKEY
}

func Load() (Config, error) {
	// Loading a .env file is optional: a missing file is fine because the
	// configuration (e.g. BOTAPIKEY) can be supplied directly via the
	// environment, such as when running inside a container. Any other load
	// error is unexpected and is surfaced to the caller.
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, err
	}

	token := os.Getenv("BOTAPIKEY")
	if token == "" {
		return Config{}, errors.New("BOTAPIKEY is not set: put the Discord bot token in the environment or in a .env file")
	}
	return Config{Token: token}, nil
}
