package database

import (
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Banned struct {
	ID int64 `gorm:"primaryKey"`
}

type Message struct {
	ID          uint `gorm:"primaryKey"`
	SenderID    uint `gorm:"not null"`
	MessageID   int  `gorm:"not null"`
	MessageText string
	MediaType   string `gorm:"size:50"`
	MediaFileID string
	CreatedAt   time.Time
	Status      string `gorm:"default:'pending'"`
	ChannelID   int64
}

type Admin struct {
	ID       uint  `gorm:"primaryKey"`
	UserID   int64 `gorm:"uniqueIndex;not null"`
	UserName string
	State    string `gorm:"default:'standart'"`
	Reason   int64
}

type Database struct {
	db *gorm.DB
}

func NewDatabase() (*Database, error) {
	db, err := gorm.Open(sqlite.Open("bot.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Message{}, &Admin{}, &Banned{})
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) SaveMessage(msg *Message) error {
	return d.db.Create(msg).Error
}

func (d *Database) GetPendingMessages() ([]Message, error) {
	var messages []Message
	err := d.db.Where("status = ?", "pending").Order("created_at asc").Find(&messages).Error
	return messages, err
}

func (d *Database) UpdateMessageStatus(messageID int, status string) error {
	return d.db.Model(&Message{}).Where("message_id = ?", messageID).Update("status", status).Error
}

func (d *Database) DeleteMessage(messageID int) error {
	return d.db.Delete(&Message{}, &Message{MessageID: messageID}).Error
}

func (d *Database) GetMessageSender(messageID int) (uint, error) {
	msg, err := d.GetMessageByID(messageID)
	if err != nil {
		return 0, err
	}
	return msg.SenderID, nil
}

func (d *Database) GetMessageByID(messageID int) (Message, error) {
	var message Message
	err := d.db.First(&message, &Message{MessageID: messageID}).Error
	return message, err
}

func (d *Database) IsAdmin(userID int64) bool {
	var admin Admin
	err := d.db.First(&admin, "user_id = ?", userID).Error
	return err == nil
}

func (d *Database) AddAdmin(userID int64, userName string) error {
	admin := Admin{
		UserID:   userID,
		UserName: userName,
		State:    "standart",
		Reason:   0,
	}
	return d.db.Create(&admin).Error
}

func (d *Database) GetAdmin(userID int64) (Admin, error) {
	var admin Admin
	err := d.db.First(&admin, "user_id = ?", userID).Error
	return admin, err
}

func (d *Database) UpdateAdminState(userID int64, state string) error {
	return d.db.Model(&Admin{}).Where("user_id = ?", userID).Update("state", state).Error
}
func (d *Database) GetAdminState(userID int64) (string, error) {
	admin, err := d.GetAdmin(userID)
	if err != nil {
		return "", err
	}
	return admin.State, nil
}

func (d *Database) UpdateAdminReason(userID int64, reason int64) error {
	return d.db.Model(&Admin{}).Where("user_id = ?", userID).Update("reason", reason).Error
}

func (d *Database) GetAdminReason(userID int64) (int64, error) {
	admin, err := d.GetAdmin(userID)
	return admin.Reason, err
}

func (d *Database) RemoveAdmin(userID int64) error {
	return d.db.Where("user_id = ?", userID).Delete(&Admin{}).Error
}

func (d *Database) GetAdmins() ([]Admin, error) {
	var admins []Admin
	err := d.db.Find(&admins).Error
	return admins, err
}

func (d *Database) BanUser(userID int64) error {
	banned := Banned{
		ID: userID,
	}
	return d.db.Create(&banned).Error
}

func (d *Database) IsBanned(userID int64) bool {
	var banned Banned
	err := d.db.First(&banned, "ID = ?", userID).Error
	return err == nil
}

func (d *Database) PardonUser(userID int64) error {
	return d.db.Where("ID = ?", userID).Delete(&Banned{}).Error
}
