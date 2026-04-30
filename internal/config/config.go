package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload"
)

const prefix = "POSTER"

type Config struct {
	Token       string `env:"TOKEN,required"`
	ChannelID   int64  `env:"CHANNEL_ID,required"`
	OwnerID     int64  `env:"OWNER_ID,required"`
	OwnerName   string `env:"OWNER_NAME,required"`
	BotUsername string `env:"BOT_USERNAME,required"`
}

func LoadConfig() (*Config, error) {
	config, err := env.ParseAsWithOptions[Config](env.Options{Prefix: prefix})
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &config, nil
}
