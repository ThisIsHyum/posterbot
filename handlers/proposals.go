package handlers

import (
	"fmt"
	"strconv"
	"strings"
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
	db          *database.Database
	media       *MediaHandler
	channelID   int64
	ownerID     int64
	botUsername string
}

func NewProposalsHandler(db *database.Database, media *MediaHandler, channelID, ownerID int64, botUsername string) *ProposalsHandler {
	return &ProposalsHandler{
		db:          db,
		media:       media,
		channelID:   channelID,
		ownerID:     ownerID,
		botUsername: botUsername,
	}
}

func (p *ProposalsHandler) HandleUserProposal(bot *telego.Bot, update telego.Update) {
	msg := update.Message
	if msg == nil {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID

	if p.db.IsBanned(userID) {
		bot.SendMessage(tu.Message(tu.ID(userID), "Вы заблокированы."))
		return
	}

	if msg.Chat.Type != "private" {
		return
	}

	if msg.Text != "" && msg.Text[0] == '/' {
		if strings.HasPrefix(msg.Text, "/reply") {
			p.handleReplyCommand(bot, msg)
		}
		return
	}

	if p.db.IsAdmin(userID) || userID == p.ownerID {
		if state, _ := p.db.GetAdminState(userID); state == "reason" {
			reasonUser, err := p.db.GetAdminReason(userID)
			if err != nil {
				return
			}
			bot.SendMessage(tu.Message(tu.ID(reasonUser), fmt.Sprintf("Ваше сообщение отклонено по причине: %s", msg.Text)))
			kb := tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("Далее").WithCallbackData("next"),
				),
			)
			bot.SendMessage(tu.Message(tu.ID(msg.From.ID), "Причина отправлена").WithReplyMarkup(kb))
			_ = p.db.UpdateAdminReason(userID, 0)
			_ = p.db.UpdateAdminState(userID, "standart")
			return
		} else if state, _ := p.db.GetAdminState(userID); state == "ban_reason" {
			reasonUser, err := p.db.GetAdminReason(userID)
			if err != nil {
				return
			}
			banID, err := p.db.CreateBanRecord(reasonUser, msg.Text)
			if err != nil {
				bot.SendMessage(tu.Message(tu.ID(reasonUser), "Ошибка при блокировке."))
				_ = p.db.UpdateAdminReason(userID, 0)
				_ = p.db.UpdateAdminState(userID, "standart")
				return
			}
			_ = p.db.BanUser(reasonUser)
			bot.SendMessage(tu.Message(tu.ID(reasonUser), fmt.Sprintf("Вы заблокированы. Код обращения: %s", banID)))
			_ = p.db.UpdateAdminReason(userID, 0)
			_ = p.db.UpdateAdminState(userID, "standart")
			bot.SendMessage(tu.Message(tu.ID(chatID), "Пользователь успешно заблокирован."))
			return
		} else if state, _ := p.db.GetAdminState(userID); state == "reply_mode" {
			p.handleReplyContent(bot, msg, userID)
			return
		}
	}

	if msg.Text == "" && msg.Photo == nil && msg.Document == nil &&
		msg.Video == nil && msg.VideoNote == nil && msg.Audio == nil &&
		msg.Voice == nil && msg.Sticker == nil {
		return
	}

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

	p.notifyAdminsAboutNewProposal(bot, message)
}

func (p *ProposalsHandler) HandleStartCommand(bot *telego.Bot, update telego.Update) {
	msg := update.Message
	if msg == nil {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	if strings.HasPrefix(text, "/start") {
		payload := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
		if strings.HasPrefix(payload, "reply_") {
			p.handleDeepLinkReply(bot, msg, strings.TrimPrefix(payload, "reply_"))
			return
		}
	}

	if p.db.IsAdmin(userID) || userID == p.ownerID {
		var messageText string

		if userID == p.ownerID {
			messageText = "👑 Панель владельца\n\nЭто бот для анонимных предложений. Пользователи присылают предложения в ЛС, а вы их модерируете.\n\n" +
				"Доступные команды:\n" +
				"/addadmin <ID> - добавить администратора\n" +
				"/admins - список администраторов\n" +
				"/banned - список блокировок\n" +
				"/proposals - просмотр предложений\n" +
				"/pardon <BAN-ID> - разбан по коду"
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

func (p *ProposalsHandler) handleReplyCommand(bot *telego.Bot, msg *telego.Message) {
	args := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/reply"))
	if args == "" {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"📝 Использование: /reply <ID_поста>\n\n"+
				"Пример: /reply 12345",
		))
		return
	}

	parentID, err := strconv.Atoi(args)
	if err != nil {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Неверный формат ID поста.",
		))
		return
	}

	_, err = p.db.GetMessageByDBID(uint(parentID))
	if err != nil {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Пост не найден.",
		))
		return
	}

	_ = p.db.UpdateAdminState(msg.From.ID, "reply_mode")
	_ = p.db.UpdateAdminReason(msg.From.ID, int64(parentID))

	bot.SendMessage(tu.Message(
		tu.ID(msg.Chat.ID),
		"✍️ Отправьте ваш ответ на пост. Он будет отправлен на модерацию.",
	))
}

func (p *ProposalsHandler) handleDeepLinkReply(bot *telego.Bot, msg *telego.Message, parentIDStr string) {
	parentID, err := strconv.Atoi(parentIDStr)
	if err != nil {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Неверная ссылка для ответа.",
		))
		return
	}

	_, err = p.db.GetMessageByDBID(uint(parentID))
	if err != nil {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Пост не найден.",
		))
		return
	}

	_ = p.db.UpdateAdminState(msg.From.ID, "reply_mode")
	_ = p.db.UpdateAdminReason(msg.From.ID, int64(parentID))

	bot.SendMessage(tu.Message(
		tu.ID(msg.Chat.ID),
		"✍️ Отправьте ваш ответ на пост. Он будет отправлен на модерацию.",
	))
}

func (p *ProposalsHandler) handleReplyContent(bot *telego.Bot, msg *telego.Message, userID int64) {
	parentMsgID64, err := p.db.GetAdminReason(userID)
	if err != nil {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Произошла ошибка при получении информации о посте.",
		))
		return
	}
	parentMsgID := int(parentMsgID64)

	mediaType, mediaFileID := p.media.GetMediaInfo(msg)
	messageText := p.media.ExtractMessageText(msg)

	message := &database.Message{
		MessageID:       msg.MessageID,
		SenderID:        uint(userID),
		MessageText:     messageText,
		MediaType:       mediaType,
		MediaFileID:     mediaFileID,
		CreatedAt:       time.Now(),
		Status:          "pending",
		ChannelID:       p.channelID,
		ParentMessageID: &parentMsgID,
	}

	if err := p.db.SaveMessage(message); err != nil {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Произошла ошибка при отправке ответа. Попробуйте позже.",
		))
		return
	}

	_ = p.db.UpdateAdminState(userID, "standart")
	_ = p.db.UpdateAdminReason(userID, 0)

	bot.SendMessage(tu.Message(
		tu.ID(msg.Chat.ID),
		"✅ Ваш ответ принят! Он будет рассмотрен модераторами анонимно.",
	))

	p.notifyAdminsAboutNewProposal(bot, message)
}

func (p *ProposalsHandler) notifyAdminsAboutNewProposal(bot *telego.Bot, message *database.Message) {
	admins, err := p.db.GetAdmins()
	if err != nil {
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
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(admin.UserID),
			notification,
		))
	}
}
