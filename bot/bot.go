package bot

import (
	"telegram-bot/database"
	"telegram-bot/handlers"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

type Bot struct {
	bot         *telego.Bot
	db          *database.Database
	botHandler  *th.BotHandler
	channelID   int64
	ownerID     int64
	ownerName   string
	botUsername string
}

func NewBot(token string, channelID, ownerID int64, ownerName string, botUsername string) (*Bot, error) {
	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, err
	}

	db, err := database.NewDatabase()
	if err != nil {
		return nil, err
	}

	botInstance := &Bot{
		bot:         bot,
		db:          db,
		channelID:   channelID,
		ownerID:     ownerID,
		ownerName:   ownerName,
		botUsername: botUsername,
	}

	botInstance.initializeOwner()

	return botInstance, nil
}

func (b *Bot) initializeOwner() {
	if !b.db.IsAdmin(b.ownerID) {
		err := b.db.AddAdmin(b.ownerID, b.ownerName)
		if err != nil {
		} else {
		}
	}
}

func (b *Bot) Start() {
	updates, err := b.bot.UpdatesViaLongPolling(&telego.GetUpdatesParams{
		AllowedUpdates: []string{"*"},
	})

	if err != nil {
		return
	}

	botHandler, err := th.NewBotHandler(b.bot, updates)
	if err != nil {
		return
	}

	b.registerHandlers(botHandler)
	b.botHandler = botHandler

	go botHandler.Start()
}

func (b *Bot) Stop() {
	if b.botHandler != nil {
		b.botHandler.Stop()
	}
	b.bot.StopLongPolling()
}

func (b *Bot) registerHandlers(bh *th.BotHandler) {

	mediaHandler := handlers.NewMediaHandler(b.db, "")
	proposalsHandler := handlers.NewProposalsHandler(b.db, mediaHandler, b.channelID, b.ownerID, b.botUsername)
	moderationHandler := handlers.NewModerationHandler(b.db, mediaHandler, b.channelID, b.ownerID, b.botUsername)
	adminHandler := handlers.NewAdminHandler(b.db, b.ownerID)

	bh.Handle(proposalsHandler.HandleStartCommand, th.CommandEqual("start"))
	bh.Handle(moderationHandler.HandleProposalsCommand, th.CommandEqual("proposals"))
	bh.Handle(adminHandler.HandleAddAdminCommand, th.CommandEqual("addadmin"))
	bh.Handle(adminHandler.HandleListAdminsCommand, th.CommandEqual("admins"))
	bh.Handle(adminHandler.HandleBannedCommand, th.CommandEqual("banned"))
	bh.Handle(moderationHandler.HandlePardonCommand, th.CommandEqual("pardon"))

	bh.Handle(moderationHandler.HandleCallback, th.AnyCallbackQuery())

	bh.Handle(proposalsHandler.HandleUserProposal, th.AnyMessage())
}
