package database

import (
	"math/rand"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Banned struct {
	ID int64 `gorm:"primaryKey"`
}

type BanRecord struct {
	ID        uint   `gorm:"primaryKey"`
	BanID     string `gorm:"uniqueIndex;size:10;not null"`
	UserID    int64  `gorm:"not null"`
	Reason    string
	CreatedAt time.Time
	Active    bool `gorm:"default:true"`
}

type Message struct {
	ID               uint `gorm:"primaryKey"`
	SenderID         uint `gorm:"not null"`
	MessageID        int  `gorm:"not null"`
	MessageText      string
	MediaType        string `gorm:"size:50"`
	MediaFileID      string
	CreatedAt        time.Time
	Status           string `gorm:"default:'pending'"`
	ChannelID        int64
	ParentMessageID  *int `gorm:"default:null"`
	ChannelMessageID *int `gorm:"default:null"`
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
	rand.Seed(time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open("bot.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Message{}, &Admin{}, &Banned{}, &BanRecord{})
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

func (d *Database) GenerateBanID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return "BAN-" + string(b)
}

func (d *Database) CreateBanRecord(userID int64, reason string) (string, error) {
	banID := d.GenerateBanID()
	record := BanRecord{
		BanID:  banID,
		UserID: userID,
		Reason: reason,
	}
	if err := d.db.Create(&record).Error; err != nil {
		return "", err
	}
	return banID, nil
}

func (d *Database) GetBanRecordByBanID(banID string) (*BanRecord, error) {
	var record BanRecord
	err := d.db.Where("ban_id = ? AND active = ?", banID, true).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (d *Database) GetActiveBanRecords() ([]BanRecord, error) {
	var records []BanRecord
	err := d.db.Where("active = ?", true).Order("created_at desc").Find(&records).Error
	return records, err
}

func (d *Database) PardonByBanID(banID string) error {
	return d.db.Model(&BanRecord{}).Where("ban_id = ?", banID).Update("active", false).Error
}

func (d *Database) UpdateMessageChannelID(dbID uint, channelMessageID int) error {
	return d.db.Model(&Message{}).Where("id = ?", dbID).Update("channel_message_id", channelMessageID).Error
}

func (d *Database) GetMessageByDBID(id uint) (Message, error) {
	var message Message
	err := d.db.First(&message, id).Error
	return message, err
}
