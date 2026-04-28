package handlers

import (
	"fmt"
	"strings"

	"telegram-bot/database"

	"github.com/mymmrac/telego"
)

type MediaHandler struct {
	db          *database.Database
	postMessage string
}

func NewMediaHandler(db *database.Database, postMessage string) *MediaHandler {
	return &MediaHandler{
		db:          db,
		postMessage: postMessage,
	}
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func (m *MediaHandler) GetMediaInfo(msg *telego.Message) (string, string) {
	if len(msg.Photo) > 0 {
		return "photo", msg.Photo[len(msg.Photo)-1].FileID
	}
	if msg.Document != nil {
		return "document", msg.Document.FileID
	}
	if msg.Video != nil {
		return "video", msg.Video.FileID
	}
	if msg.Audio != nil {
		return "audio", msg.Audio.FileID
	}
	if msg.Voice != nil {
		return "voice", msg.Voice.FileID
	}
	if msg.Sticker != nil {
		return "sticker", msg.Sticker.FileID
	}
	if msg.VideoNote != nil {
		return "video_note", msg.VideoNote.FileID
	}
	return "text", ""
}

func (m *MediaHandler) ExtractMessageText(msg *telego.Message) string {
	if msg.Text != "" {
		return msg.Text
	}
	if msg.Caption != "" {
		return msg.Caption
	}

	switch {
	case msg.Photo != nil:
		return "🖼️ Фото"
	case msg.Document != nil:
		return "📄 Документ: " + msg.Document.FileName
	case msg.Video != nil:
		return "🎥 Видео"
	case msg.VideoNote != nil:
		return "📹 Кружочек (видеосообщение)"
	case msg.Audio != nil:
		title := "Аудио"
		if msg.Audio.Title != "" {
			title = msg.Audio.Title
		}
		return "🎵 " + title
	case msg.Voice != nil:
		return "🎤 Голосовое сообщение"
	case msg.Sticker != nil:
		return "😊 Стикер"
	default:
		return "📦 Медиа-контент"
	}
}

func (m *MediaHandler) SendMediaForModeration(bot *telego.Bot, chatID int64, message database.Message) error {
	if message.MediaType != "text" && message.MediaFileID != "" {
		var sendErr error

		switch message.MediaType {
		case "photo":
			_, sendErr = bot.SendPhoto(&telego.SendPhotoParams{
				ChatID:  telego.ChatID{ID: chatID},
				Photo:   telego.InputFile{FileID: message.MediaFileID},
				Caption: message.MessageText,
			})
		case "document":
			_, sendErr = bot.SendDocument(&telego.SendDocumentParams{
				ChatID:   telego.ChatID{ID: chatID},
				Document: telego.InputFile{FileID: message.MediaFileID},
				Caption:  message.MessageText,
			})
		case "video":
			_, sendErr = bot.SendVideo(&telego.SendVideoParams{
				ChatID:  telego.ChatID{ID: chatID},
				Video:   telego.InputFile{FileID: message.MediaFileID},
				Caption: message.MessageText,
			})
		case "video_note":
			_, sendErr = bot.SendVideoNote(&telego.SendVideoNoteParams{
				ChatID:    telego.ChatID{ID: chatID},
				VideoNote: telego.InputFile{FileID: message.MediaFileID},
			})
		case "audio":
			_, sendErr = bot.SendAudio(&telego.SendAudioParams{
				ChatID:  telego.ChatID{ID: chatID},
				Audio:   telego.InputFile{FileID: message.MediaFileID},
				Caption: message.MessageText,
			})
		case "voice":
			_, sendErr = bot.SendVoice(&telego.SendVoiceParams{
				ChatID:  telego.ChatID{ID: chatID},
				Voice:   telego.InputFile{FileID: message.MediaFileID},
				Caption: message.MessageText,
			})
		case "sticker":
			_, sendErr = bot.SendSticker(&telego.SendStickerParams{
				ChatID:  telego.ChatID{ID: chatID},
				Sticker: telego.InputFile{FileID: message.MediaFileID},
			})
		}

		if sendErr != nil {
			_, err := bot.SendMessage(&telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   fmt.Sprintf("❌ Не удалось отобразить медиафайл (тип: %s)\n💬 Описание: %s", message.MediaType, message.MessageText),
			})
			return err
		}
	} else {
		_, err := bot.SendMessage(&telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   fmt.Sprintf("💬 Текст предложения:\n%s", message.MessageText),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MediaHandler) PublishMedia(bot *telego.Bot, channelID int64, message database.Message, botUsername string, replyToMsgID int) (*telego.Message, error) {
	var sentMsg *telego.Message
	var sendErr error

	// Оборачиваем пользовательский текст в цитату, предварительно экранируя HTML
	escapedText := escapeHTML(message.MessageText)
	quotedText := fmt.Sprintf("<blockquote>%s</blockquote>", escapedText)

	var baseCaption string
	if message.ParentMessageID != nil {
		baseCaption = "💬 Ответ:\n\n" + quotedText
	} else {
		baseCaption = quotedText
	}

	// Гиперссылка остаётся снаружи цитаты
	replyLink := fmt.Sprintf("\n\n<a href=\"https://t.me/%s?start=reply_%d\">💬 Ответить</a>", botUsername, message.ID)
	safeCaption := baseCaption + replyLink

	disablePreview := true

	switch message.MediaType {
	case "photo":
		params := &telego.SendPhotoParams{
			ChatID:           telego.ChatID{ID: channelID},
			Photo:            telego.InputFile{FileID: message.MediaFileID},
			Caption:          safeCaption,
			ParseMode:        "HTML",
			ReplyToMessageID: replyToMsgID,
		}
		sentMsg, sendErr = bot.SendPhoto(params)
	case "document":
		params := &telego.SendDocumentParams{
			ChatID:           telego.ChatID{ID: channelID},
			Document:         telego.InputFile{FileID: message.MediaFileID},
			Caption:          safeCaption,
			ParseMode:        "HTML",
			ReplyToMessageID: replyToMsgID,
		}
		sentMsg, sendErr = bot.SendDocument(params)
	case "video":
		params := &telego.SendVideoParams{
			ChatID:           telego.ChatID{ID: channelID},
			Video:            telego.InputFile{FileID: message.MediaFileID},
			Caption:          safeCaption,
			ParseMode:        "HTML",
			ReplyToMessageID: replyToMsgID,
		}
		sentMsg, sendErr = bot.SendVideo(params)
	case "video_note":
		params := &telego.SendVideoNoteParams{
			ChatID:           telego.ChatID{ID: channelID},
			VideoNote:        telego.InputFile{FileID: message.MediaFileID},
			ReplyToMessageID: replyToMsgID,
		}
		sentMsg, sendErr = bot.SendVideoNote(params)
	case "audio":
		params := &telego.SendAudioParams{
			ChatID:           telego.ChatID{ID: channelID},
			Audio:            telego.InputFile{FileID: message.MediaFileID},
			Caption:          safeCaption,
			ParseMode:        "HTML",
			ReplyToMessageID: replyToMsgID,
		}
		sentMsg, sendErr = bot.SendAudio(params)
	case "voice":
		params := &telego.SendVoiceParams{
			ChatID:           telego.ChatID{ID: channelID},
			Voice:            telego.InputFile{FileID: message.MediaFileID},
			Caption:          safeCaption,
			ParseMode:        "HTML",
			ReplyToMessageID: replyToMsgID,
		}
		sentMsg, sendErr = bot.SendVoice(params)
	case "sticker":
		params := &telego.SendStickerParams{
			ChatID:           telego.ChatID{ID: channelID},
			Sticker:          telego.InputFile{FileID: message.MediaFileID},
			ReplyToMessageID: replyToMsgID,
		}
		sentMsg, sendErr = bot.SendSticker(params)
	default:
		params := &telego.SendMessageParams{
			ChatID:                telego.ChatID{ID: channelID},
			Text:                  safeCaption,
			ParseMode:             "HTML",
			DisableWebPagePreview: disablePreview,
			ReplyToMessageID:      replyToMsgID,
		}
		sentMsg, sendErr = bot.SendMessage(params)
	}

	return sentMsg, sendErr
}
