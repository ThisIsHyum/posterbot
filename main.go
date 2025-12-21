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
		log.Printf("Не удалось загрузить dotenv использую переменные окружения")
	}
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не установлен")
	}
	owner_id, err := strconv.Atoi(os.Getenv("OWNER_ID"))
	if err != nil {
		log.Fatalf("Ошибка айди владельца не должен быть пустым")
	}
	owner_name := os.Getenv("OWNER_NAME")
	channelID, err := strconv.Atoi(os.Getenv("CHANNEL_ID"))
	if err != nil {
		log.Fatal("Ошибка айди канала не должен быть пустым")
	}

	bot, err := bot.NewBot(token, int64(channelID), int64(owner_id), owner_name)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	bot.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	bot.Stop()
}
