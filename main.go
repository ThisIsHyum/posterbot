package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"telegram-bot/bot"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Не удалось загрузить .env, использую переменные окружения")
	}
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не установлен")
	}
	ownerID, err := strconv.Atoi(os.Getenv("OWNER_ID"))
	if err != nil {
		log.Fatal("OWNER_ID должен быть числом")
	}
	ownerName := os.Getenv("OWNER_NAME")
	channelID, err := strconv.Atoi(os.Getenv("CHANNEL_ID"))
	if err != nil {
		log.Fatal("CHANNEL_ID должен быть числом")
	}
	botUsername := os.Getenv("BOT_USERNAME")
	if botUsername == "" {
		log.Fatal("BOT_USERNAME не установлен")
	}

	bot, err := bot.NewBot(token, int64(channelID), int64(ownerID), ownerName, botUsername)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	bot.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	bot.Stop()
}
