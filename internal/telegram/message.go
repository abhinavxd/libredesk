package telegram

import (
	"fmt"
	"strconv"
	"strings"
)

type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

type Chat struct {
	Username  string `json:"username"`
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type File struct {
	ID     string `json:"file_id"`
	Size   int64  `json:"file_size"`
	Path   string `json:"file_path"`
	Name   string `json:"file_name"`
	MIME   string `json:"mime_type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Contact struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone_number"`
}

type Venue struct {
	Location Location `json:"location"`
	Title    string   `json:"title"`
	Address  string   `json:"address"`
}

type Message struct {
	Venue                *Venue    `json:"venue"`
	MediaGroupID         string    `json:"media_group_id"`
	Entities             []Entity  `json:"entities"`
	CaptionEntities      []Entity  `json:"caption_entities"`
	CallbackID           string    `json:"-"`
	ReplyTo              *Message  `json:"reply_to_message"`
	BusinessConnectionID string    `json:"business_connection_id"`
	ThreadID             int64     `json:"message_thread_id"`
	SenderBusinessBot    *User     `json:"sender_business_bot"`
	ID                   int64     `json:"message_id"`
	Date                 int64     `json:"date"`
	EditDate             int64     `json:"edit_date"`
	From                 *User     `json:"from"`
	Chat                 Chat      `json:"chat"`
	Text                 string    `json:"text"`
	Caption              string    `json:"caption"`
	Photo                []File    `json:"photo"`
	Document             *File     `json:"document"`
	Video                *File     `json:"video"`
	Audio                *File     `json:"audio"`
	Voice                *File     `json:"voice"`
	Animation            *File     `json:"animation"`
	Sticker              *File     `json:"sticker"`
	VideoNote            *File     `json:"video_note"`
	Location             *Location `json:"location"`
	Contact              *Contact  `json:"contact"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	Data    string   `json:"data"`
	From    *User    `json:"from"`
	Message *Message `json:"message"`
}

type Update struct {
	CallbackQuery         *CallbackQuery `json:"callback_query"`
	ID                    int64          `json:"update_id"`
	Message               *Message       `json:"message"`
	EditedMessage         *Message       `json:"edited_message"`
	BusinessMessage       *Message       `json:"business_message"`
	EditedBusinessMessage *Message       `json:"edited_business_message"`
}

func (m Message) Inbound() bool {
	return m.ID > 0 && m.Chat.ID > 0 && m.Chat.Type == "private" && m.From != nil && !m.From.IsBot && m.From.ID > 0 && (m.From.ID == m.Chat.ID || m.BusinessConnectionID != "")
}

func (m Message) Attachment() (File, string) {
	if len(m.Photo) > 0 {
		largest := m.Photo[0]
		for _, photo := range m.Photo[1:] {
			if photo.Width*photo.Height > largest.Width*largest.Height {
				largest = photo
			}
		}
		largest.MIME = "image/jpeg"
		return largest, "photo"
	}
	for _, media := range []struct {
		File *File
		Kind string
	}{{m.Document, "document"}, {m.Animation, "animation"}, {m.Video, "video"}, {m.Audio, "audio"}, {m.Voice, "voice"}, {m.Sticker, "sticker"}, {m.VideoNote, "video"}} {
		if media.File != nil {
			return *media.File, media.Kind
		}
	}
	return File{}, ""
}

func (m Message) Content() string {
	if m.Text != "" {
		return m.Text
	}
	if m.Caption != "" {
		return m.Caption
	}
	if m.Venue != nil {
		return strings.TrimSpace(m.Venue.Title+"\n"+m.Venue.Address) + fmt.Sprintf("\nhttps://maps.google.com/?q=%f,%f", m.Venue.Location.Latitude, m.Venue.Location.Longitude)
	}
	if m.Location != nil {
		return fmt.Sprintf("https://maps.google.com/?q=%f,%f", m.Location.Latitude, m.Location.Longitude)
	}
	if m.Contact != nil {
		return strings.TrimSpace(m.Contact.FirstName+" "+m.Contact.LastName) + "\n" + m.Contact.Phone
	}
	return ""
}

func (m Message) Outgoing() bool {
	return m.BusinessConnectionID != "" && m.From != nil && m.From.ID != m.Chat.ID
}

func (m Message) SourceID(inboxID int) string {
	if m.CallbackID != "" {
		return fmt.Sprintf("telegram:%d:%s:%d:callback:%s", inboxID, m.BusinessConnectionID, m.Chat.ID, m.CallbackID)
	}
	if m.BusinessConnectionID != "" {
		return BusinessSourceID(inboxID, m.BusinessConnectionID, m.Chat.ID, m.ID)
	}
	return SourceID(inboxID, m.Chat.ID, m.ID)
}

func (q CallbackQuery) IncomingMessage() (Message, bool) {
	if q.ID == "" || strings.TrimSpace(q.Data) == "" || q.Message == nil {
		return Message{}, false
	}
	message := Message{ID: q.Message.ID, Chat: q.Message.Chat, From: q.From, Text: q.Data, CallbackID: q.ID, BusinessConnectionID: q.Message.BusinessConnectionID, ThreadID: q.Message.ThreadID}
	return message, message.Inbound()
}

func MessageIDFromSource(sourceID string, inboxID int, chatID int64, businessID string) (int64, error) {
	suffix := sourceID[strings.LastIndex(sourceID, ":")+1:]
	id, err := strconv.ParseInt(suffix, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid telegram reply reference")
	}
	if sourceID != (Message{ID: id, Chat: Chat{ID: chatID}, BusinessConnectionID: businessID}).SourceID(inboxID) {
		return 0, fmt.Errorf("telegram reply belongs to another chat")
	}
	return id, nil
}

func BusinessSourceID(inboxID int, connectionID string, chatID, messageID int64) string {
	return fmt.Sprintf("telegram:%d:business:%s:%d:%d", inboxID, connectionID, chatID, messageID)
}

func SourceID(inboxID int, chatID, messageID int64) string {
	return fmt.Sprintf("telegram:%d:%d:%d", inboxID, chatID, messageID)
}
