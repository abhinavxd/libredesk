package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/telegram"
	"github.com/zerodha/logf"
)

const ChannelTelegram = "telegram"

type Store interface {
	RecordTelegramSend(messageUUID string, sourceIDs []string) error
	TelegramCSATButtons(uuid string) ([]telegram.Button, error)
}

type Config struct {
	GreetingMessage string `json:"greeting_message"`
	AwayMessage     string `json:"away_message"`
	BusinessHoursID int    `json:"business_hours_id"`
	Timezone        string `json:"timezone"`
	CSATMessage     string `json:"csat_message"`
	BotToken        string `json:"bot_token"`
	BotID           int64  `json:"bot_id"`
	BotUsername     string `json:"bot_username"`
	SecretToken     string `json:"secret_token"`
}

type SendMeta struct {
	CSATUUID             string            `json:"csat_uuid,omitempty"`
	Buttons              []telegram.Button `json:"buttons,omitempty"`
	ReplyToMessageID     int64             `json:"reply_to_message_id,omitempty"`
	BusinessConnectionID string            `json:"business_connection_id,omitempty"`
	ThreadID             int64             `json:"message_thread_id,omitempty"`
	ChatID               int64             `json:"chat_id"`
}

type Opts struct {
	ID         int
	Name       string
	Config     Config
	Client     *telegram.Client
	Store      Store
	Lo         *logf.Logger
	AuthStatus func(int, bool)
}

type Telegram struct {
	opts Opts
}

func New(opts Opts) (*Telegram, error) {
	if opts.Config.BotToken == "" || opts.Config.SecretToken == "" || opts.Client == nil || opts.Store == nil || opts.Lo == nil {
		return nil, fmt.Errorf("telegram credentials, client, logger and message store are required")
	}
	return &Telegram{opts: opts}, nil
}

func (t *Telegram) Identifier() int               { return t.opts.ID }
func (t *Telegram) Name() string                  { return t.opts.Name }
func (t *Telegram) Channel() string               { return ChannelTelegram }
func (t *Telegram) FromAddress() string           { return "" }
func (t *Telegram) ReplyToAddress() string        { return "" }
func (t *Telegram) FromNameTemplate() string      { return "" }
func (t *Telegram) Close() error                  { return nil }
func (t *Telegram) Receive(context.Context) error { return nil }

func (t *Telegram) Send(message models.OutboundMessage) error {
	var meta struct {
		Telegram SendMeta `json:"telegram"`
	}
	if err := json.Unmarshal(message.Meta, &meta); err != nil {
		return err
	}
	if meta.Telegram.ChatID <= 0 {
		return fmt.Errorf("missing telegram recipient")
	}
	text := message.Content
	options := telegram.SendOptions{Buttons: meta.Telegram.Buttons, ReplyToMessageID: meta.Telegram.ReplyToMessageID, BusinessConnectionID: meta.Telegram.BusinessConnectionID, ThreadID: meta.Telegram.ThreadID}
	if meta.Telegram.CSATUUID != "" {
		var err error
		options.Buttons, err = t.opts.Store.TelegramCSATButtons(meta.Telegram.CSATUUID)
		if err != nil {
			return err
		}
	}
	if message.ContentType == models.ContentTypeHTML {
		text, _ = telegram.FormatHTML(text)
		options.ParseMode = "HTML"
	} else if message.TextContent != "" {
		text = message.TextContent
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var sent []telegram.Message
	var err error
	if len(message.Attachments) > 1 {
		sent, err = t.opts.Client.SendAlbum(ctx, t.opts.Config.BotToken, meta.Telegram.ChatID, text, message.Attachments, options)
	} else {
		var id int64
		id, err = t.opts.Client.Send(ctx, t.opts.Config.BotToken, meta.Telegram.ChatID, text, message.Attachments, options)
		sent = []telegram.Message{{ID: id}}
	}
	if err != nil {
		var apiErr *telegram.APIError
		if t.opts.AuthStatus != nil && errors.As(err, &apiErr) && apiErr.Code == http.StatusUnauthorized {
			t.opts.AuthStatus(t.opts.ID, false)
		}
		return err
	}
	if t.opts.AuthStatus != nil {
		t.opts.AuthStatus(t.opts.ID, true)
	}
	if len(sent) != max(1, len(message.Attachments)) {
		return fmt.Errorf("Telegram returned an incomplete message response")
	}
	sources := make([]string, len(sent))
	for i, item := range sent {
		if item.ID <= 0 {
			return fmt.Errorf("Telegram returned no message ID")
		}
		sources[i] = telegram.Message{ID: item.ID, Chat: telegram.Chat{ID: meta.Telegram.ChatID}, BusinessConnectionID: meta.Telegram.BusinessConnectionID}.SourceID(t.opts.ID)
	}
	if err := t.opts.Store.RecordTelegramSend(message.UUID, sources); err != nil {
		t.opts.Lo.Error("error storing telegram message ids", "message_uuid", message.UUID, "error", err)
	}
	return nil
}
