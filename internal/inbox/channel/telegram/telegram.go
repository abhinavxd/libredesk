package telegram

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

const (
	ChannelTelegram = "telegram"
	telegramAPIBase = "https://api.telegram.org/bot"
)

type Config struct {
	BotToken     string `json:"bot_token"`
	BotName      string `json:"bot_name"`
	WebhookURL   string `json:"webhook_url"`
	WebhookSecret string `json:"webhook_secret"`
}

type Telegram struct {
	id           int
	config       Config
	lo           *logf.Logger
	messageStore inbox.MessageStore
	userStore    inbox.UserStore
	httpClient   *http.Client
}

type Opts struct {
	ID     int
	Config Config
	Lo     *logf.Logger
}

func New(store inbox.MessageStore, userStore inbox.UserStore, opts Opts) (*Telegram, error) {
	return &Telegram{
		id:           opts.ID,
		config:       opts.Config,
		lo:           opts.Lo,
		messageStore: store,
		userStore:    userStore,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (t *Telegram) Identifier() int {
	return t.id
}

func (t *Telegram) Receive(ctx context.Context) error {
	return nil
}

func (t *Telegram) Send(message models.OutboundMessage) error {
	chatID := t.extractChatID(message.Meta)
	if chatID == "" {
		return fmt.Errorf("telegram chat_id not found in message meta for message %s", message.UUID)
	}

	content := message.TextContent
	if content == "" {
		content = message.Content
	}

	if len(message.Attachments) > 0 {
		var lastErr error
		for i, att := range message.Attachments {
			caption := ""
			if i == 0 {
				caption = content
			}
			if err := t.sendAttachment(chatID, att, caption); err != nil {
				t.lo.Error("error sending telegram attachment", "error", err, "attachment", att.Name)
				lastErr = err
			}
		}
		return lastErr
	}

	if content != "" {
		if err := t.sendMessage(chatID, content); err != nil {
			return fmt.Errorf("sending telegram message: %w", err)
		}
	}

	return nil
}

func (t *Telegram) Close() error {
	return nil
}

func (t *Telegram) FromAddress() string {
	return t.config.BotName
}

func (t *Telegram) ReplyToAddress() string {
	return ""
}

func (t *Telegram) Channel() string {
	return ChannelTelegram
}

// WebhookSecret returns the secret used to verify incoming webhook requests.
func (t *Telegram) WebhookSecret() string {
	if t.config.WebhookSecret != "" {
		return t.config.WebhookSecret
	}
	// Derive a secret from bot token using HMAC-SHA256.
	h := hmac.New(sha256.New, []byte("libredesk-telegram-webhook"))
	h.Write([]byte(t.config.BotToken))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// VerifyWebhook checks the X-Telegram-Bot-Api-Secret-Token header.
// If the header is empty, verification is skipped (webhook was set without secret_token).
func (t *Telegram) VerifyWebhook(secretToken string) bool {
	if secretToken == "" {
		return true
	}
	return secretToken == t.WebhookSecret()
}

func (t *Telegram) SetWebhook(url string) error {
	apiURL := fmt.Sprintf("%s%s/setWebhook", telegramAPIBase, t.config.BotToken)
	payload, _ := json.Marshal(map[string]string{
		"url":          url,
		"secret_token": t.WebhookSecret(),
	})

	resp, err := t.httpClient.Post(apiURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("setting telegram webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram setWebhook failed: %s", string(body))
	}
	return nil
}

func (t *Telegram) ProcessWebhookUpdate(update Update) error {
	msg := update.Message
	if msg == nil {
		return nil
	}

	if msg.From == nil || msg.Chat == nil {
		return nil
	}

	// Use a synthetic email to identify the Telegram contact instead of external_user_id.
	// Format: {telegram_user_id}@telegram
	telegramEmail := fmt.Sprintf("%d@telegram", msg.From.ID)

	contact := models.IncomingContact{
		FirstName: msg.From.FirstName,
		LastName:  msg.From.LastName,
		Email:     null.StringFrom(telegramEmail),
	}

	if avatarBytes, mime := t.downloadUserAvatar(msg.From.ID); len(avatarBytes) > 0 {
		contact.AvatarContent = avatarBytes
		contact.AvatarMIME = mime
	}

	content := msg.Text
	if content == "" && msg.Caption != "" {
		content = msg.Caption
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"telegram_chat_id":    strconv.FormatInt(msg.Chat.ID, 10),
		"telegram_user_id":    strconv.FormatInt(msg.From.ID, 10),
		"telegram_message_id": msg.MessageID,
		"telegram_username":   msg.From.Username,
	})

	incomingMsg := models.IncomingMessage{
		Channel:     ChannelTelegram,
		InboxID:     t.id,
		Contact:     contact,
		SourceID:    null.StringFrom(fmt.Sprintf("telegram_%d_%d", msg.Chat.ID, msg.MessageID)),
		Content:     content,
		ContentType: models.ContentTypeText,
		Meta:        meta,
	}

	if attachments := t.extractAttachments(msg); len(attachments) > 0 {
		incomingMsg.Attachments = attachments
		if incomingMsg.Content == "" {
			incomingMsg.Content = "[attachment]"
		}
	}

	if err := t.messageStore.EnqueueIncoming(incomingMsg); err != nil {
		return fmt.Errorf("enqueuing telegram message: %w", err)
	}

	t.lo.Info("telegram message enqueued", "chat_id", msg.Chat.ID, "message_id", msg.MessageID)
	return nil
}

func (t *Telegram) extractAttachments(msg *Message) []attachment.Attachment {
	var attachments []attachment.Attachment

	if len(msg.Photo) > 0 {
		photo := msg.Photo[len(msg.Photo)-1]
		if att, err := t.downloadFile(photo.FileID, "photo.jpg", "image/jpeg"); err == nil {
			attachments = append(attachments, att)
		} else {
			t.lo.Error("error downloading telegram photo", "error", err)
		}
	}

	if msg.Document != nil {
		name := msg.Document.FileName
		if name == "" {
			name = "document"
		}
		mime := msg.Document.MimeType
		if mime == "" {
			mime = "application/octet-stream"
		}
		if att, err := t.downloadFile(msg.Document.FileID, name, mime); err == nil {
			attachments = append(attachments, att)
		} else {
			t.lo.Error("error downloading telegram document", "error", err)
		}
	}

	if msg.Video != nil {
		name := msg.Video.FileName
		if name == "" {
			name = "video.mp4"
		}
		mime := msg.Video.MimeType
		if mime == "" {
			mime = "video/mp4"
		}
		if att, err := t.downloadFile(msg.Video.FileID, name, mime); err == nil {
			attachments = append(attachments, att)
		} else {
			t.lo.Error("error downloading telegram video", "error", err)
		}
	}

	if msg.Voice != nil {
		mime := msg.Voice.MimeType
		if mime == "" {
			mime = "audio/ogg"
		}
		if att, err := t.downloadFile(msg.Voice.FileID, "voice.ogg", mime); err == nil {
			attachments = append(attachments, att)
		} else {
			t.lo.Error("error downloading telegram voice", "error", err)
		}
	}

	if msg.Audio != nil {
		name := msg.Audio.FileName
		if name == "" {
			name = "audio.mp3"
		}
		mime := msg.Audio.MimeType
		if mime == "" {
			mime = "audio/mpeg"
		}
		if att, err := t.downloadFile(msg.Audio.FileID, name, mime); err == nil {
			attachments = append(attachments, att)
		} else {
			t.lo.Error("error downloading telegram audio", "error", err)
		}
	}

	if msg.Sticker != nil {
		// For animated (.tgs) or video (.webm) stickers, use the thumbnail if available.
		// Static stickers (.webp) can be downloaded directly.
		if msg.Sticker.IsAnimated || msg.Sticker.IsVideo {
			if msg.Sticker.Thumbnail != nil {
				if att, err := t.downloadFile(msg.Sticker.Thumbnail.FileID, "sticker.jpg", "image/jpeg"); err == nil {
					attachments = append(attachments, att)
				} else {
					t.lo.Error("error downloading telegram sticker thumbnail", "error", err)
				}
			} else {
				// No thumbnail available, set emoji as content placeholder.
				// The message content will show the emoji.
			}
		} else {
			if att, err := t.downloadFile(msg.Sticker.FileID, "sticker.webp", "image/webp"); err == nil {
				attachments = append(attachments, att)
			} else {
				t.lo.Error("error downloading telegram sticker", "error", err)
			}
		}
	}

	return attachments
}

func (t *Telegram) downloadFile(fileID, fileName, mimeType string) (attachment.Attachment, error) {
	fileURL, err := t.getFileURL(fileID)
	if err != nil {
		return attachment.Attachment{}, err
	}

	resp, err := t.httpClient.Get(fileURL)
	if err != nil {
		return attachment.Attachment{}, fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return attachment.Attachment{}, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return attachment.Attachment{}, fmt.Errorf("reading file content: %w", err)
	}

	return attachment.Attachment{
		Name:        fileName,
		Size:        len(content),
		Content:     content,
		ContentType: mimeType,
		Disposition: attachment.DispositionAttachment,
	}, nil
}

func (t *Telegram) downloadUserAvatar(userID int64) ([]byte, string) {
	apiURL := fmt.Sprintf("%s%s/getUserProfilePhotos?user_id=%d&limit=1", telegramAPIBase, t.config.BotToken, userID)

	resp, err := t.httpClient.Get(apiURL)
	if err != nil {
		return nil, ""
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			TotalCount int `json:"total_count"`
			Photos     [][]struct {
				FileID string `json:"file_id"`
			} `json:"photos"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ""
	}

	if !result.OK || result.Result.TotalCount == 0 || len(result.Result.Photos) == 0 || len(result.Result.Photos[0]) == 0 {
		return nil, ""
	}

	photoSizes := result.Result.Photos[0]
	fileID := photoSizes[len(photoSizes)-1].FileID

	fileURL, err := t.getFileURL(fileID)
	if err != nil {
		return nil, ""
	}

	dlResp, err := t.httpClient.Get(fileURL)
	if err != nil {
		return nil, ""
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return nil, ""
	}

	content, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return nil, ""
	}

	mime := "image/jpeg"
	if strings.HasSuffix(fileURL, ".png") {
		mime = "image/png"
	} else if strings.HasSuffix(fileURL, ".webp") {
		mime = "image/webp"
	}

	return content, mime
}

func (t *Telegram) getFileURL(fileID string) (string, error) {
	apiURL := fmt.Sprintf("%s%s/getFile?file_id=%s", telegramAPIBase, t.config.BotToken, fileID)

	resp, err := t.httpClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("getting telegram file: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding telegram file response: %w", err)
	}

	if !result.OK || result.Result.FilePath == "" {
		return "", fmt.Errorf("telegram getFile failed for file_id: %s", fileID)
	}

	return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", t.config.BotToken, result.Result.FilePath), nil
}

func (t *Telegram) extractChatID(meta json.RawMessage) string {
	if len(meta) == 0 {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(meta, &m); err != nil {
		return ""
	}
	if cid, ok := m["telegram_chat_id"].(string); ok {
		return cid
	}
	if cid, ok := m["telegram_chat_id"].(float64); ok {
		return strconv.FormatInt(int64(cid), 10)
	}
	return ""
}

func (t *Telegram) sendAttachment(chatID string, att attachment.Attachment, caption string) error {
	if len(att.Content) > 0 {
		if isImageContentType(att.ContentType) {
			return t.uploadMultipart(
				fmt.Sprintf("%s%s/sendPhoto", telegramAPIBase, t.config.BotToken),
				chatID, "photo", att.Content, att.Name, caption,
			)
		}
		return t.uploadMultipart(
			fmt.Sprintf("%s%s/sendDocument", telegramAPIBase, t.config.BotToken),
			chatID, "document", att.Content, att.Name, caption,
		)
	}
	if att.URL != "" {
		if isImageContentType(att.ContentType) {
			return t.sendMediaJSON(
				fmt.Sprintf("%s%s/sendPhoto", telegramAPIBase, t.config.BotToken),
				chatID, "photo", att.URL, caption,
			)
		}
		return t.sendMediaJSON(
			fmt.Sprintf("%s%s/sendDocument", telegramAPIBase, t.config.BotToken),
			chatID, "document", att.URL, caption,
		)
	}
	return nil
}

func (t *Telegram) sendMessage(chatID, text string) error {
	apiURL := fmt.Sprintf("%s%s/sendMessage", telegramAPIBase, t.config.BotToken)
	payload, _ := json.Marshal(map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	})

	resp, err := t.httpClient.Post(apiURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("sending telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram sendMessage failed: %s", string(body))
	}
	return nil
}

func (t *Telegram) sendMediaJSON(apiURL, chatID, field, mediaURL, caption string) error {
	payload := map[string]interface{}{
		"chat_id": chatID,
		field:     mediaURL,
	}
	if caption != "" {
		payload["caption"] = caption
	}

	body, _ := json.Marshal(payload)
	resp, err := t.httpClient.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API failed: %s", string(respBody))
	}
	return nil
}

func (t *Telegram) uploadMultipart(apiURL, chatID, fieldName string, content []byte, fileName, caption string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("chat_id", chatID)
	if caption != "" {
		writer.WriteField("caption", caption)
	}

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return fmt.Errorf("creating form file: %w", err)
	}
	part.Write(content)
	writer.Close()

	resp, err := t.httpClient.Post(apiURL, writer.FormDataContentType(), body)
	if err != nil {
		return fmt.Errorf("uploading file to telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram upload failed: %s", string(respBody))
	}
	return nil
}

func isImageContentType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp":
		return true
	}
	return false
}
