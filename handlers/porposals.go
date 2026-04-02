package handlers

import (
	"fmt"
	"log"
	"time"

	"telegram-bot/database"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

const welcomeText = `🤖 Добро пожаловать в анонимную предложку!

Просто отправьте сюда ваше предложение, идею или сообщение, и оно будет анонимно рассмотрено модераторами.

Ваша личность будет скрыта - модераторы увидят только содержание вашего сообщения.

❓ Что можно отправлять:
• Текстовые предложения
• Фотографии
• Документы
• Видео
• Кружочки (видеосообщения)
• Аудио и голосовые сообщения
• Стикеры
• Идеи и пожелания

Ваше предложение будет рассмотрено в ближайшее время!`

type ProposalsHandler struct {
	db        *database.Database
	media     *MediaHandler
	channelID int64
	ownerID   int64
}

func NewProposalsHandler(db *database.Database, media *MediaHandler, channelID, ownerID int64) *ProposalsHandler {
	return &ProposalsHandler{
		db:        db,
		media:     media,
		channelID: channelID,
		ownerID:   ownerID,
	}
}

func (p *ProposalsHandler) HandleUserProposal(bot *telego.Bot, update telego.Update) {
	log.Printf("Обрабротка предложения")
	msg := update.Message
	if msg == nil {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID
	if p.db.IsBanned(userID) {
		bot.SendMessage(tu.Message(tu.ID(userID), "Вы заблокированы"))
		return
	}
	if msg.Chat.Type != "private" {
		return
	}

	if msg.Text != "" && msg.Text[0] == '/' {
		return
	}

	if p.db.IsAdmin(userID) || userID == p.ownerID {
		if state, _ := p.db.GetAdminState(userID); state == "reason" {
			log.Printf("Обработка причины")
			reasonUser, err := p.db.GetAdminReason(userID)
			if err != nil {
				log.Printf("Ошибка получения юзера для написания причины")
				return
			}
			log.Printf("Юзер для отправки причины: %d", reasonUser)
			bot.SendMessage(tu.Message(tu.ID(reasonUser), fmt.Sprintf("Ваше сообщение отклонено по причине: %s", msg.Text)))
			kb := tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("Далее").WithCallbackData("next"),
				),
			)
			bot.SendMessage(tu.Message(tu.ID(msg.From.ID), "Причина отправленна").WithReplyMarkup(kb))
			p.db.UpdateAdminReason(userID, 0)
			p.db.UpdateAdminState(userID, "standart")
			return
		} else if state, _ := p.db.GetAdminState(userID); state == "ban_reason" {
			log.Printf("Обработка причины блокировки")
			reasonUser, err := p.db.GetAdminReason(userID)
			if err != nil {
				log.Printf("Ошибка получения юзера для написания причины")
				return
			}
			log.Printf("Юзер для отправки причины: %d", reasonUser)
			err = p.db.BanUser(reasonUser)
			if err != nil {
				log.Printf("Ошибка блокировки юзера: %d", reasonUser)
				bot.SendMessage(tu.Message(tu.ID(reasonUser), "Ошибка при блокировке юзера, обратитесь к тех. администратору"))
				p.db.UpdateAdminReason(userID, 0)
				p.db.UpdateAdminState(userID, "standart")
				return
			}
			bot.SendMessage(tu.Message(tu.ID(reasonUser), fmt.Sprintf("Вы заблокированы по причине: %s", msg.Text)))
			p.db.UpdateAdminReason(userID, 0)
			p.db.UpdateAdminState(userID, "standart")
			bot.SendMessage(tu.Message(tu.ID(chatID), "Пользователь успешно заблокирован"))
			return
		}
	}

	if msg.Text == "" && msg.Photo == nil && msg.Document == nil &&
		msg.Video == nil && msg.VideoNote == nil && msg.Audio == nil &&
		msg.Voice == nil && msg.Sticker == nil {
		return
	}

	log.Printf("📨 Новое предложение от пользователя %d", userID)

	mediaType, mediaFileID := p.media.GetMediaInfo(msg)
	messageText := p.media.ExtractMessageText(msg)

	message := &database.Message{
		MessageID:   msg.MessageID,
		SenderID:    uint(userID),
		MessageText: messageText,
		MediaType:   mediaType,
		MediaFileID: mediaFileID,
		CreatedAt:   time.Now(),
		Status:      "pending",
		ChannelID:   p.channelID,
	}

	if err := p.db.SaveMessage(message); err != nil {
		log.Printf("Ошибка сохранения предложения: %v", err)
		bot.SendMessage(tu.Message(
			tu.ID(chatID),
			"❌ Произошла ошибка при отправке предложения. Попробуйте позже.",
		))
		return
	}

	bot.SendMessage(tu.Message(
		tu.ID(chatID),
		"✅ Ваше предложение принято! Оно будет рассмотрено модераторами анонимно.",
	))

	log.Printf("✅ Предложение сохранено: %s (тип: %s)", messageText, mediaType)

	p.notifyAdminsAboutNewProposal(bot, message)
}

func (p *ProposalsHandler) notifyAdminsAboutNewProposal(bot *telego.Bot, message *database.Message) {
	admins, err := p.db.GetAdmins()
	if err != nil {
		log.Printf("Ошибка получения списка администраторов: %v", err)
		return
	}

	notification := fmt.Sprintf(
		"📨 Поступило новое анонимное предложение!\n\n"+
			"ID Предложения: %d\n\n"+
			"💬 Текст: %s\n"+
			"📁 Тип: %s\n\n"+
			"Используйте /proposals для просмотра всех предложений.",
		message.ID,
		message.MessageText,
		message.MediaType,
	)

	for _, admin := range admins {
		_, err := bot.SendMessage(tu.Message(
			tu.ID(admin.UserID),
			notification,
		))
		if err != nil {
			log.Printf("Ошибка отправки уведомления администратору %d: %v", admin.UserID, err)
		}
	}
}

func (p *ProposalsHandler) HandleStartCommand(bot *telego.Bot, update telego.Update) {
	msg := update.Message
	if msg == nil {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID

	log.Printf("Обработка /start от пользователя %d", userID)

	if p.db.IsAdmin(userID) || userID == p.ownerID {

		var messageText string

		if userID == p.ownerID {
			messageText = "👑 Панель владельца\n\nЭто бот для анонимных предложений. Пользователи присылают предложения в ЛС, а вы их модерируете.\n\n" +
				"Доступные команды:\n" +
				"/addadmin <ID> - добавить администратора\n" +
				"/admins - список администраторов\n" +
				"/proposals - просмотр предложений"

		} else {
			messageText = "🛠️ Панель модератора\n\nЭто бот для анонимных предложений. Пользователи присылают предложения в ЛС, а вы их модерируете.\n\n" +
				"Доступные команды:\n" +
				"/proposals - просмотр предложений"
		}

		bot.SendMessage(tu.Message(
			tu.ID(chatID),
			messageText,
		))
	} else {

		bot.SendMessage(tu.Message(
			tu.ID(chatID),
			welcomeText,
		))
	}
}
