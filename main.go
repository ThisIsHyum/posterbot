package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"telegram-bot/bot"
)

const OWNER_ID = 6569505824

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не установлен")
	}

	channelID := int64(-1002431451231)

	bot, err := bot.NewBot(token, channelID, OWNER_ID)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	bot.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	bot.Stop()
}
