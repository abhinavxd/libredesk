package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
)

const (
	APIURL           = "https://api.telegram.org"
	MaxDownloadBytes = 20 * 1024 * 1024
	MaxUploadBytes   = 50 * 1024 * 1024
	MaxTextLength    = 4096
	MaxCaptionLength = 1024
)

var ErrFileTooLarge = errors.New("telegram file exceeds the 20 MB download limit")

type APIError struct {
	Code        int    `json:"error_code"`
	Description string `json:"description"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

type SendOptions struct {
	Buttons              []Button
	ReplyToMessageID     int64
	BusinessConnectionID string
	ThreadID             int64
	ParseMode            string `json:"parse_mode,omitempty"`
}

type ReplyParameters struct {
	MessageID                int64 `json:"message_id"`
	AllowSendingWithoutReply bool  `json:"allow_sending_without_reply"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New() *Client {
	return &Client{baseURL: APIURL, http: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (e *APIError) Error() string { return fmt.Sprintf("Telegram: %s (%d)", e.Description, e.Code) }

func (c *Client) SetBaseURL(baseURL string) { c.baseURL = strings.TrimRight(baseURL, "/") }

func (c *Client) GetMe(ctx context.Context, token string) (User, error) {
	var user User
	err := c.call(ctx, token, "getMe", map[string]any{}, &user)
	return user, err
}

func (c *Client) GetProfilePhoto(ctx context.Context, token string, userID int64) (string, error) {
	var result struct {
		Photos [][]File `json:"photos"`
	}
	if err := c.call(ctx, token, "getUserProfilePhotos", map[string]any{"user_id": userID, "limit": 1}, &result); err != nil {
		return "", err
	}
	if len(result.Photos) == 0 || len(result.Photos[0]) == 0 {
		return "", nil
	}
	photos := result.Photos[0]
	return photos[len(photos)-1].ID, nil
}

func (c *Client) SetWebhook(ctx context.Context, token, callback, secret string) error {
	return c.call(ctx, token, "setWebhook", map[string]any{
		"url": callback, "secret_token": secret, "allowed_updates": []string{"message", "edited_message", "business_message", "edited_business_message", "callback_query"}, "max_connections": 10,
	}, nil)
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, token, queryID string) error {
	return c.call(ctx, token, "answerCallbackQuery", map[string]string{"callback_query_id": queryID}, nil)
}

func (c *Client) DeleteWebhook(ctx context.Context, token string) error {
	return c.call(ctx, token, "deleteWebhook", map[string]any{}, nil)
}

func (c *Client) Send(ctx context.Context, token string, chatID int64, text string, attachments attachment.Attachments, options SendOptions) (int64, error) {
	return withRetry(ctx, func() (int64, error) {
		return c.send(ctx, token, chatID, text, attachments, options)
	})
}

func (c *Client) Download(ctx context.Context, token, fileID string) (File, []byte, error) {
	var file File
	if err := c.call(ctx, token, "getFile", map[string]string{"file_id": fileID}, &file); err != nil {
		return file, nil, err
	}
	if file.Size > MaxDownloadBytes {
		return file, nil, ErrFileTooLarge
	}
	if file.Path == "" || strings.HasPrefix(file.Path, "/") || path.Clean(file.Path) != file.Path || strings.Contains(file.Path, "..") {
		return file, nil, fmt.Errorf("invalid telegram file path")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/file/bot"+token+"/"+file.Path, nil)
	if err != nil {
		return file, nil, fmt.Errorf("building telegram download request")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return file, nil, redactError(err, token)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return file, nil, &APIError{Code: resp.StatusCode, Description: "file download failed"}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxDownloadBytes+1))
	if err != nil {
		return file, nil, redactError(err, token)
	}
	if len(body) > MaxDownloadBytes {
		return file, nil, ErrFileTooLarge
	}
	return file, body, nil
}

func (c *Client) send(ctx context.Context, token string, chatID int64, text string, attachments attachment.Attachments, options SendOptions) (int64, error) {
	var message Message
	if err := ValidateButtons(options.Buttons); err != nil {
		return 0, err
	}
	if len(attachments) == 0 {
		payload := map[string]any{"chat_id": chatID, "text": text}
		if keyboard := MakeKeyboard(options.Buttons); keyboard != nil {
			payload["reply_markup"] = keyboard
		}
		if options.ReplyToMessageID > 0 {
			payload["reply_parameters"] = ReplyParameters{MessageID: options.ReplyToMessageID, AllowSendingWithoutReply: true}
		}
		if options.BusinessConnectionID != "" {
			payload["business_connection_id"] = options.BusinessConnectionID
		}
		if options.ThreadID > 0 {
			payload["message_thread_id"] = options.ThreadID
		}
		if options.ParseMode != "" {
			payload["parse_mode"] = options.ParseMode
		}
		err := c.call(ctx, token, "sendMessage", payload, &message)
		return message.ID, err
	}
	if len(attachments) != 1 {
		return 0, fmt.Errorf("send one attachment per message")
	}
	att := attachments[0]
	if len(att.Content) > MaxUploadBytes {
		return 0, fmt.Errorf("file exceeds the 50 MB upload limit")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	writer.WriteField("caption", text)
	writeOptions(writer, options)
	if options.ParseMode != "" {
		writer.WriteField("parse_mode", options.ParseMode)
	}
	if keyboard := MakeKeyboard(options.Buttons); keyboard != nil {
		value, _ := json.Marshal(keyboard)
		writer.WriteField("reply_markup", string(value))
	}
	field := MediaType(att)
	method := "send" + strings.ToUpper(field[:1]) + field[1:]

	part, _ := writer.CreateFormFile(field, att.Name)
	part.Write(att.Content)
	writer.Close()
	err := c.request(ctx, token, method, writer.FormDataContentType(), &body, &message)
	return message.ID, err
}

func (c *Client) call(ctx context.Context, token, method string, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.request(ctx, token, method, "application/json", bytes.NewReader(body), result)
}

func (c *Client) request(ctx context.Context, token, method, contentType string, body io.Reader, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/bot"+token+"/"+method, body)
	if err != nil {
		return fmt.Errorf("building telegram request")
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := c.http.Do(req)
	if err != nil {
		return redactError(err, token)
	}
	defer resp.Body.Close()
	var response struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		APIError
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2*1024*1024)).Decode(&response); err != nil {
		return fmt.Errorf("invalid Telegram response (HTTP %d)", resp.StatusCode)
	}
	if !response.OK || resp.StatusCode != http.StatusOK {
		if response.Code == 0 {
			response.Code = resp.StatusCode
		}
		response.Description = strings.ReplaceAll(response.Description, token, "[redacted]")
		return &response.APIError
	}
	if result != nil {
		return json.Unmarshal(response.Result, result)
	}
	return nil
}

func withRetry[T any](ctx context.Context, send func() (T, error)) (T, error) {
	for attempt := 0; ; attempt++ {
		result, err := send()
		var apiErr *APIError
		if attempt >= 2 || !errors.As(err, &apiErr) || apiErr.Code != http.StatusTooManyRequests {
			return result, err
		}
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-time.After(time.Duration(max(1, apiErr.Parameters.RetryAfter)) * time.Second):
		}
	}
}

func writeOptions(writer *multipart.Writer, options SendOptions) {
	if options.BusinessConnectionID != "" {
		writer.WriteField("business_connection_id", options.BusinessConnectionID)
	}
	if options.ThreadID > 0 {
		writer.WriteField("message_thread_id", strconv.FormatInt(options.ThreadID, 10))
	}
	if options.ReplyToMessageID > 0 {
		reply, _ := json.Marshal(ReplyParameters{MessageID: options.ReplyToMessageID, AllowSendingWithoutReply: true})
		writer.WriteField("reply_parameters", string(reply))
	}
}

func redactError(err error, token string) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	return errors.New(strings.ReplaceAll(err.Error(), token, "[redacted]"))
}
