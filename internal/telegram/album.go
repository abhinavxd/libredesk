package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/image"
)

const MaxAlbumSize = 10

type InputMedia struct {
	Type      string `json:"type"`
	Media     string `json:"media"`
	Caption   string `json:"caption,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"`
}

func (c *Client) SendAlbum(ctx context.Context, token string, chatID int64, text string, files attachment.Attachments, options SendOptions) ([]Message, error) {
	if len(files) < 2 || len(files) > MaxAlbumSize || len(options.Buttons) > 0 {
		return nil, fmt.Errorf("Telegram albums require 2 to 10 attachments and no buttons")
	}
	for attempt := 0; ; attempt++ {
		messages, err := c.sendAlbum(ctx, token, chatID, text, files, options)
		var apiErr *APIError
		if attempt >= 2 || !errors.As(err, &apiErr) || apiErr.Code != http.StatusTooManyRequests {
			return messages, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(max(1, apiErr.Parameters.RetryAfter)) * time.Second):
		}
	}
}

func (c *Client) sendAlbum(ctx context.Context, token string, chatID int64, text string, files attachment.Attachments, options SendOptions) ([]Message, error) {
	media := make([]InputMedia, len(files))
	family := ""
	documents := false
	for i, file := range files {
		if len(file.Content) > MaxUploadBytes {
			return nil, fmt.Errorf("file exceeds the 50 MB upload limit")
		}
		kind := MediaType(file)
		group := kind
		if kind == "photo" || kind == "video" {
			group = "visual"
		}
		if kind == "voice" || kind == "animation" || (family != "" && family != group) {
			documents = true
		}
		family = group
		media[i] = InputMedia{Type: kind, Media: "attach://file" + strconv.Itoa(i)}
	}
	if documents {
		for i := range media {
			media[i].Type = "document"
		}
	}
	media[0].Caption = text
	media[0].ParseMode = options.ParseMode
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	if options.BusinessConnectionID != "" {
		writer.WriteField("business_connection_id", options.BusinessConnectionID)
	}
	if options.ThreadID > 0 {
		writer.WriteField("message_thread_id", strconv.FormatInt(options.ThreadID, 10))
	}
	if options.ReplyToMessageID > 0 {
		value, _ := json.Marshal(ReplyParameters{MessageID: options.ReplyToMessageID, AllowSendingWithoutReply: true})
		writer.WriteField("reply_parameters", string(value))
	}
	value, _ := json.Marshal(media)
	writer.WriteField("media", string(value))
	for i, file := range files {
		part, _ := writer.CreateFormFile("file"+strconv.Itoa(i), file.Name)
		part.Write(file.Content)
	}
	writer.Close()
	var messages []Message
	err := c.request(ctx, token, "sendMediaGroup", writer.FormDataContentType(), &body, &messages)
	return messages, err
}

func MediaType(file attachment.Attachment) string {
	switch strings.Split(file.ContentType, ";")[0] {
	case "image/jpeg", "image/png":
		if len(file.Content) <= 10*1024*1024 {
			width, height, err := image.GetDimensions(bytes.NewReader(file.Content))
			if err == nil && width > 0 && height > 0 && width+height <= 10000 && max(width, height) <= 20*min(width, height) {
				return "photo"
			}
		}
	case "video/mp4":
		return "video"
	case "audio/mpeg", "audio/mp4":
		return "audio"
	case "audio/ogg":
		return "voice"
	case "image/gif":
		return "animation"
	}
	return "document"
}
