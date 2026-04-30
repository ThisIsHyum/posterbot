package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"telegram-bot/bot"
	"telegram-bot/internal/config"
)

func main() {
	cfg, err := config.LoadConfig()

	bot, err := bot.NewBot(
		cfg.Token, cfg.ChannelID, cfg.OwnerID, cfg.OwnerName, cfg.BotUsername)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	bot.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	bot.Stop()
}
