package bot

import (
	"log"

	"telegram-bot/database"
	"telegram-bot/handlers"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

type Bot struct {
	bot        *telego.Bot
	db         *database.Database
	botHandler *th.BotHandler
	channelID  int64
	ownerID    int64
}

func NewBot(token string, channelID, ownerID int64) (*Bot, error) {
	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, err
	}

	db, err := database.NewDatabase()
	if err != nil {
		return nil, err
	}

	botInstance := &Bot{
		bot:       bot,
		db:        db,
		channelID: channelID,
		ownerID:   ownerID,
	}

	botInstance.initializeOwner()

	return botInstance, nil
}

func (b *Bot) initializeOwner() {
	if !b.db.IsAdmin(b.ownerID) {
		err := b.db.AddAdmin(b.ownerID, "vstor08")
		if err != nil {
			log.Printf("Предупреждение: не удалось добавить владельца: %v", err)
		} else {
			log.Printf("✅ Владелец %d добавлен как администратор", b.ownerID)
		}
	}
}

func (b *Bot) Start() {
	updates, err := b.bot.UpdatesViaLongPolling(nil)
	if err != nil {
		log.Printf("Ошибка получения обновлений: %v", err)
		return
	}

	botHandler, err := th.NewBotHandler(b.bot, updates)
	if err != nil {
		log.Printf("Ошибка создания обработчика: %v", err)
		return
	}

	b.registerHandlers(botHandler)
	b.botHandler = botHandler

	go botHandler.Start()

	log.Println("🤖 Бот-предложка запущен! Принимает анонимные предложения в ЛС")
}

func (b *Bot) Stop() {
	if b.botHandler != nil {
		b.botHandler.Stop()
	}
	b.bot.StopLongPolling()
	log.Println("Бот остановлен")
}

func (b *Bot) registerHandlers(bh *th.BotHandler) {

	mediaHandler := handlers.NewMediaHandler(b.db)
	proposalsHandler := handlers.NewProposalsHandler(b.db, mediaHandler, b.channelID, b.ownerID)
	moderationHandler := handlers.NewModerationHandler(b.db, mediaHandler, b.channelID, b.ownerID)
	adminHandler := handlers.NewAdminHandler(b.db, b.ownerID)

	bh.Handle(proposalsHandler.HandleStartCommand, th.CommandEqual("start"))
	bh.Handle(moderationHandler.HandleProposalsCommand, th.CommandEqual("proposals"))
	bh.Handle(adminHandler.HandleAddAdminCommand, th.CommandEqual("addadmin"))
	bh.Handle(adminHandler.HandleListAdminsCommand, th.CommandEqual("admins"))

	bh.Handle(moderationHandler.HandleCallback, th.AnyCallbackQuery())

	bh.Handle(proposalsHandler.HandleUserProposal, th.AnyMessage())
}
