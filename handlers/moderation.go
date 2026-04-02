package handlers

import (
	"fmt"
	"log"

	"telegram-bot/database"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type ModerationHandler struct {
	db        *database.Database
	media     *MediaHandler
	channelID int64
	ownerID   int64
}

func NewModerationHandler(db *database.Database, media *MediaHandler, channelID, ownerID int64) *ModerationHandler {
	return &ModerationHandler{
		db:        db,
		media:     media,
		channelID: channelID,
		ownerID:   ownerID,
	}
}

func (m *ModerationHandler) HandleProposalsCommand(bot *telego.Bot, update telego.Update) {
	msg := update.Message
	if msg == nil {
		return
	}
	m.ShowProposals(bot, msg.Chat.ID, msg.From.ID)
}

func (m *ModerationHandler) ShowProposals(bot *telego.Bot, chatID int64, userID int64) {
	if !m.db.IsAdmin(userID) && userID != m.ownerID {
		bot.SendMessage(tu.Message(
			tu.ID(chatID),
			"❌ У вас нет доступа к этой функции.",
		))
		return
	}

	messages, err := m.db.GetPendingMessages()
	if err != nil {
		bot.SendMessage(tu.Message(
			tu.ID(chatID),
			"❌ Ошибка при получении предложений: "+err.Error(),
		))
		return
	}

	if len(messages) == 0 {
		bot.SendMessage(tu.Message(
			tu.ID(chatID),
			"✅ Нет новых предложений для модерации.",
		))
		return
	}

	bot.SendMessage(tu.Message(
		tu.ID(chatID),
		fmt.Sprintf("📨 Найдено %d предложений для модерации:", len(messages)),
	))

	m.SendMessageForModeration(bot, chatID, messages[0])
}

func (m *ModerationHandler) SendMessageForModeration(bot *telego.Bot, chatID int64, message database.Message) {

	if err := m.media.SendMediaForModeration(bot, chatID, message); err != nil {
		log.Printf("Ошибка при отправке медиа для модерации: %v", err)
	}

	text := fmt.Sprintf(
		"📨 Анонимное предложение #%d\n\n"+
			"⏰ Время: %s\n\n"+
			"Выберите действие:",
		message.MessageID,
		message.CreatedAt.Format("02.01.2006 15:04"),
	)

	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("✅ ОДОБРИТЬ").WithCallbackData(fmt.Sprintf("approve_%d", message.MessageID)),
			tu.InlineKeyboardButton("❌ ОТКЛОНИТЬ").WithCallbackData(fmt.Sprintf("reject_%d", message.MessageID)),
		),
	)

	bot.SendMessage(tu.Message(
		tu.ID(chatID),
		text,
	).WithReplyMarkup(keyboard))
}

func (m *ModerationHandler) HandlePardonCommand(bot *telego.Bot, update telego.Update) {
	msg := update.Message
	if msg == nil {
		return
	}
	senderID := update.Message.Chat.ID

	if !m.db.IsAdmin(senderID) && senderID != m.ownerID {
		return
	}
	args := msg.Text
	if len(args) < 8 {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"📝 Использование: /pardon <ID_пользователя>\n\n"+
				"Пример: /pardon 123456789",
		))
		return
	}
	var targetUserID int64
	_, err := fmt.Sscanf(args, "/pardon %d", &targetUserID)
	if err != nil || targetUserID == 0 {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Неверный формат ID. Используйте: /pardon <ID_пользователя>\n\n"+
				"Пример: /pardon 123456789",
		))
		return
	}
	err = m.db.PardonUser(targetUserID)
	if err != nil {
		log.Printf("Ошибка разбана юзера %d:%s", targetUserID, err)
		return
	}
	bot.SendMessage(tu.Message(tu.ID(targetUserID), "Вы разаблокированы"))
	bot.SendMessage(tu.Message(tu.ID(msg.Chat.ID), fmt.Sprintf("Пользователь %d успешно разбанен", targetUserID)))

}

func (m *ModerationHandler) HandleCallback(bot *telego.Bot, update telego.Update) {

	callback := update.CallbackQuery
	if callback == nil {
		return
	}
	log.Printf("Вызов callback: %s", callback.ID)
	userID := callback.From.ID
	chatID := callback.Message.Chat.ID

	if !m.db.IsAdmin(userID) && userID != m.ownerID {
		bot.AnswerCallbackQuery(tu.CallbackQuery(
			callback.ID,
		).WithText("❌ У вас нет доступа."))
		return
	}

	data := callback.Data
	var messageID int
	var senderID int

	if n, _ := fmt.Sscanf(data, "approve_%d", &messageID); n == 1 {
		m.HandleApprove(bot, chatID, messageID, callback)
	} else if n, _ := fmt.Sscanf(data, "reject_%d", &messageID); n == 1 {
		m.HandleReject(bot, chatID, messageID, callback)
	} else if n, _ := fmt.Sscanf(data, "reason_%d", &senderID); n == 1 {
		m.HandleReason(bot, chatID, senderID, callback)
	} else if n, _ := fmt.Sscanf(data, "ban_reason_%d", &senderID); n == 1 {
		m.HandleBanReason(bot, chatID, senderID, callback)
	} else if n, _ := fmt.Sscanf(data, "next"); n == 0 {
		m.HandleNext(bot, chatID, messageID, callback)
	} else {
		log.Printf("Ни один хендлер не обрабатывает данный калбек: %d", n)
	}
}

func (m *ModerationHandler) HandleNext(bot *telego.Bot, chatID int64, messageID int, callback *telego.CallbackQuery) {
	log.Printf("Вызов Next хендлера")
	m.ShowProposals(bot, chatID, chatID)
	bot.AnswerCallbackQuery(tu.CallbackQuery(callback.ID).WithText("хуй"))
}

func (m *ModerationHandler) HandleReason(bot *telego.Bot, chatID int64, senderID int, callback *telego.CallbackQuery) {
	bot.SendMessage(tu.Message(tu.ID(chatID), "Введите причину отказа"))
	m.db.UpdateAdminState(chatID, "reason")
	log.Printf("senderID: %d", senderID)
	m.db.UpdateAdminReason(chatID, int64(senderID))
	bot.AnswerCallbackQuery(tu.CallbackQuery(callback.ID).WithText("хуй"))

}

func (m *ModerationHandler) HandleBanReason(bot *telego.Bot, chatID int64, senderID int, callback *telego.CallbackQuery) {
	bot.SendMessage(tu.Message(tu.ID(chatID), "Введите причину блокировки"))
	m.db.UpdateAdminState(chatID, "ban_reason")
	log.Printf("senderID: %d", senderID)
	m.db.UpdateAdminReason(chatID, int64(senderID))
	bot.AnswerCallbackQuery(tu.CallbackQuery(callback.ID).WithText("Good"))
}
func (m *ModerationHandler) HandleApprove(bot *telego.Bot, chatID int64, messageID int, callback *telego.CallbackQuery) {
	message, err := m.db.GetMessageByID(messageID)
	if err != nil {
		bot.AnswerCallbackQuery(tu.CallbackQuery(
			callback.ID,
		).WithText("❌ Ошибка: предложение не найдено"))
		return
	}

	if err := m.media.PublishMedia(bot, m.channelID, message); err != nil {
		log.Printf("Ошибка отправки в канал: %v", err)
		bot.AnswerCallbackQuery(tu.CallbackQuery(
			callback.ID,
		).WithText("❌ Ошибка при публикации"))
		return
	}

	m.db.UpdateMessageStatus(messageID, "approved")
	m.db.DeleteMessage(messageID)

	bot.AnswerCallbackQuery(tu.CallbackQuery(
		callback.ID,
	).WithText("✅ Предложение опубликовано!"))

	bot.DeleteMessage(&telego.DeleteMessageParams{
		ChatID:    tu.ID(chatID),
		MessageID: callback.Message.MessageID,
	})

	m.ShowProposals(bot, chatID, callback.From.ID)
}

func (m *ModerationHandler) HandleReject(bot *telego.Bot, chatID int64, messageID int, callback *telego.CallbackQuery) {
	msg, err := m.db.GetMessageByID(messageID)
	if err != nil {
		bot.AnswerCallbackQuery(tu.CallbackQuery(
			callback.ID,
		).WithText("❌ Ошибка: предложение не найдено"))
		return
	}
	senderID := msg.SenderID
	m.db.UpdateMessageStatus(messageID, "rejected")
	m.db.DeleteMessage(messageID)
	bot.SendMessage(tu.Message(tu.ID(chatID), ""))
	bot.AnswerCallbackQuery(tu.CallbackQuery(
		callback.ID,
	).WithText("✅ Предложение отклонено!"))

	bot.DeleteMessage(&telego.DeleteMessageParams{
		ChatID:    tu.ID(chatID),
		MessageID: callback.Message.MessageID,
	})
	log.Printf("senderID: %d", senderID)
	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Причина").WithCallbackData(fmt.Sprintf("reason_%d", senderID)),
			tu.InlineKeyboardButton("Далее").WithCallbackData("next"),
			tu.InlineKeyboardButton("Бан").WithCallbackData(fmt.Sprintf("ban_reason_%d", senderID)),
		),
	)
	bot.SendMessage(tu.Message(tu.ID(chatID), "Выберете действие").WithReplyMarkup(kb))
}
