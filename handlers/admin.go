package handlers

import (
	"fmt"

	"telegram-bot/database"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type AdminHandler struct {
	db      *database.Database
	ownerID int64
}

func NewAdminHandler(db *database.Database, ownerID int64) *AdminHandler {
	return &AdminHandler{
		db:      db,
		ownerID: ownerID,
	}
}

func (a *AdminHandler) IsOwner(userID int64) bool {
	return userID == a.ownerID
}

func (a *AdminHandler) HandleAddAdminCommand(bot *telego.Bot, update telego.Update) {
	msg := update.Message
	if msg == nil {
		return
	}

	if !a.IsOwner(msg.From.ID) {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Только владелец бота может добавлять администраторов.",
		))
		return
	}

	args := msg.Text
	if len(args) < 10 {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"📝 Использование: /addadmin <ID_пользователя>\n\n"+
				"Пример: /addadmin 123456789",
		))
		return
	}

	var targetUserID int64
	_, err := fmt.Sscanf(args, "/addadmin %d", &targetUserID)
	if err != nil || targetUserID == 0 {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Неверный формат ID. Используйте: /addadmin <ID_пользователя>\n\n"+
				"Пример: /addadmin 123456789",
		))
		return
	}

	if targetUserID == a.ownerID {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Вы уже являетесь владельцем бота.",
		))
		return
	}

	if a.db.IsAdmin(targetUserID) {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			fmt.Sprintf("❌ Пользователь уже является администратором.", targetUserID),
		))
		return
	}

	var userName string
	user, err := bot.GetChat(&telego.GetChatParams{
		ChatID: tu.ID(targetUserID),
	})
	if err != nil {
		userName = fmt.Sprintf("user_%d", targetUserID)
	} else {
		if user.Type == "private" {
			userName = user.FirstName
			if user.Username != "" {
				userName = user.Username
			}
		} else {
			userName = user.Title
		}
	}

	err = a.db.AddAdmin(targetUserID, userName)
	if err != nil {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Ошибка при добавлении администратора: "+err.Error(),
		))
		return
	}

	successMsg := fmt.Sprintf("✅ Пользователь %s добавлен как администратор!", userName)
	bot.SendMessage(tu.Message(
		tu.ID(msg.Chat.ID),
		successMsg,
	))

	notificationMsg := "🎉 Вы были добавлены как модератор бота-предложки!\n\n" +
		"Используйте команду /start для доступа к панели модерации."

	_, _ = bot.SendMessage(tu.Message(
		tu.ID(targetUserID),
		notificationMsg,
	))
}

func (a *AdminHandler) HandleListAdminsCommand(bot *telego.Bot, update telego.Update) {
	msg := update.Message
	if msg == nil {
		return
	}

	if !a.IsOwner(msg.From.ID) {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Только владелец бота может просматривать список администраторов.",
		))
		return
	}

	admins, err := a.db.GetAdmins()
	if err != nil {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Ошибка при получении списка администраторов: "+err.Error(),
		))
		return
	}

	if len(admins) == 0 {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"📋 Список модераторов пуст.",
		))
		return
	}

	adminList := "📋 Список модераторов:\n\n"
	adminList += fmt.Sprintf("👑 Владелец: ID %d\n", a.ownerID)

	for i, admin := range admins {
		adminList += fmt.Sprintf("%d. @%s\n", i+1, admin.UserName)
	}

	bot.SendMessage(tu.Message(
		tu.ID(msg.Chat.ID),
		adminList,
	))
}

func (a *AdminHandler) HandleBannedCommand(bot *telego.Bot, update telego.Update) {
	msg := update.Message
	if msg == nil {
		return
	}

	if !a.db.IsAdmin(msg.From.ID) && msg.From.ID != a.ownerID {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ У вас нет доступа к этой функции.",
		))
		return
	}

	records, err := a.db.GetActiveBanRecords()
	if err != nil {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"❌ Ошибка при получении списка банов: "+err.Error(),
		))
		return
	}

	if len(records) == 0 {
		bot.SendMessage(tu.Message(
			tu.ID(msg.Chat.ID),
			"✅ Список банов пуст.",
		))
		return
	}

	banList := "🚫 Активные блокировки:\n\n"
	for i, record := range records {
		banList += fmt.Sprintf("%d. %s\n   Причина: %s\n   Дата: %s\n",
			i+1,
			record.BanID,
			record.Reason,
			record.CreatedAt.Format("02.01.2006 15:04"),
		)
	}

	bot.SendMessage(tu.Message(
		tu.ID(msg.Chat.ID),
		banList,
	))
}
