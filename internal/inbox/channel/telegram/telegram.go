package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/telegram"
	"github.com/zerodha/logf"
)

const ChannelTelegram = "telegram"

type SourceIDUpdater interface {
	RecordTelegramSend(messageUUID string, sourceIDs []string) error
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
	ID            int
	Name          string
	Config        Config
	Client        *telegram.Client
	CSATButtons   func(string) ([]telegram.Button, error)
	SourceUpdater SourceIDUpdater
	Lo            *logf.Logger
	AuthStatus    func(int, bool)
}

type Telegram struct {
	id            int
	name          string
	config        Config
	client        *telegram.Client
	csatButtons   func(string) ([]telegram.Button, error)
	sourceUpdater SourceIDUpdater
	lo            *logf.Logger
	authStatus    func(int, bool)
}

func New(opts Opts) (*Telegram, error) {
	if opts.Config.BotToken == "" || opts.Config.SecretToken == "" || opts.Client == nil || opts.SourceUpdater == nil || opts.Lo == nil {
		return nil, fmt.Errorf("telegram credentials, client, logger and message store are required")
	}
	return &Telegram{id: opts.ID, name: opts.Name, config: opts.Config, client: opts.Client, sourceUpdater: opts.SourceUpdater, csatButtons: opts.CSATButtons, lo: opts.Lo, authStatus: opts.AuthStatus}, nil
}

func (t *Telegram) Identifier() int               { return t.id }
func (t *Telegram) Name() string                  { return t.name }
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
	plain := text
	options := telegram.SendOptions{Buttons: meta.Telegram.Buttons, ReplyToMessageID: meta.Telegram.ReplyToMessageID, BusinessConnectionID: meta.Telegram.BusinessConnectionID, ThreadID: meta.Telegram.ThreadID}
	if meta.Telegram.CSATUUID != "" {
		if t.csatButtons == nil {
			return fmt.Errorf("CSAT survey is unavailable")
		}
		var err error
		options.Buttons, err = t.csatButtons(meta.Telegram.CSATUUID)
		if err != nil {
			return err
		}
	}
	if message.ContentType == models.ContentTypeHTML {
		text, plain = telegram.FormatHTML(text)
		options.ParseMode = "HTML"
	} else if message.TextContent != "" {
		text = message.TextContent
		plain = text
	}
	limit := telegram.MaxTextLength
	if len(message.Attachments) > 0 {
		limit = telegram.MaxCaptionLength
	}
	if utf8.RuneCountInString(plain) > limit {
		return fmt.Errorf("Telegram message exceeds %d characters", limit)
	}
	if strings.TrimSpace(plain) == "" && len(message.Attachments) == 0 {
		return fmt.Errorf("message has no content")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var sent []telegram.Message
	var err error
	if len(message.Attachments) > 1 {
		sent, err = t.client.SendAlbum(ctx, t.config.BotToken, meta.Telegram.ChatID, text, message.Attachments, options)
	} else {
		var id int64
		id, err = t.client.Send(ctx, t.config.BotToken, meta.Telegram.ChatID, text, message.Attachments, options)
		sent = []telegram.Message{{ID: id}}
	}
	if err != nil {
		var apiErr *telegram.APIError
		if t.authStatus != nil && errors.As(err, &apiErr) && apiErr.Code == http.StatusUnauthorized {
			t.authStatus(t.id, false)
		}
		return err
	}
	if t.authStatus != nil {
		t.authStatus(t.id, true)
	}
	if len(sent) != max(1, len(message.Attachments)) {
		return fmt.Errorf("Telegram returned an incomplete message response")
	}
	sources := make([]string, len(sent))
	for i, item := range sent {
		if item.ID <= 0 {
			return fmt.Errorf("Telegram returned no message ID")
		}
		sources[i] = telegram.SourceID(t.id, meta.Telegram.ChatID, item.ID)
		if meta.Telegram.BusinessConnectionID != "" {
			sources[i] = telegram.BusinessSourceID(t.id, meta.Telegram.BusinessConnectionID, meta.Telegram.ChatID, item.ID)
		}
	}
	if err := t.sourceUpdater.RecordTelegramSend(message.UUID, sources); err != nil {
		t.lo.Error("error storing telegram message ids", "message_uuid", message.UUID, "error", err)
	}
	return nil
}
