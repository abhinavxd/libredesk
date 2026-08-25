// Package whatsapp provides a client for the WhatsApp Cloud API and helpers for parsing Meta webhook payloads.
package whatsapp

import "time"

const DefaultAPIVersion = "v25.0"

// Account holds the per-inbox Meta Graph API credentials, already decrypted at the call site.
type Account struct {
	PhoneNumberID string
	WABAID        string
	AccessToken   string
	AppSecret     string
	APIVersion    string
}

type MetaAPIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Type       string `json:"type"`
	Code       int    `json:"code"`
	Subcode    int    `json:"error_subcode"`
	UserMsg    string `json:"error_user_msg"`
	FBTraceID  string `json:"fbtrace_id"`
}

type metaErrorEnvelope struct {
	Error struct {
		Message      string `json:"message"`
		Type         string `json:"type"`
		Code         int    `json:"code"`
		ErrorSubcode int    `json:"error_subcode"`
		ErrorUserMsg string `json:"error_user_msg"`
		FBTraceID    string `json:"fbtrace_id"`
	} `json:"error"`
}

type SendResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WAID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID            string `json:"id"`
		MessageStatus string `json:"message_status"`
	} `json:"messages"`
}

type MediaInfo struct {
	URL              string `json:"url"`
	MimeType         string `json:"mime_type"`
	SHA256           string `json:"sha256"`
	FileSize         int64  `json:"file_size"`
	ID               string `json:"id"`
	MessagingProduct string `json:"messaging_product"`
}

type UploadMediaResponse struct {
	ID string `json:"id"`
}

type MetaTemplate struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Language       string              `json:"language"`
	Category       string              `json:"category"`
	Status         string              `json:"status"`
	Components     []TemplateComponent `json:"components"`
	QualityScore   any                 `json:"quality_score,omitempty"`
	RejectedReason string              `json:"rejected_reason,omitempty"`
}

type TemplateComponent struct {
	Type    string           `json:"type"`
	Format  string           `json:"format,omitempty"`
	Text    string           `json:"text,omitempty"`
	Example map[string]any   `json:"example,omitempty"`
	Buttons []TemplateButton `json:"buttons,omitempty"`
}

type TemplateButton struct {
	Type        string   `json:"type"`
	Text        string   `json:"text,omitempty"`
	URL         string   `json:"url,omitempty"`
	PhoneNumber string   `json:"phone_number,omitempty"`
	Example     []string `json:"example,omitempty"`
}

type TemplateSubmission struct {
	Name            string              `json:"name"`
	Language        string              `json:"language"`
	Category        string              `json:"category"`
	ParameterFormat string              `json:"parameter_format,omitempty"`
	Components      []TemplateComponent `json:"components"`
}

// TemplateEdit is the payload sent to Meta when editing a template; name and language are immutable on Meta so they are omitted.
type TemplateEdit struct {
	Category        string              `json:"category,omitempty"`
	ParameterFormat string              `json:"parameter_format,omitempty"`
	Components      []TemplateComponent `json:"components"`
}

type phoneNumberListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
}

type templateListResponse struct {
	Data   []MetaTemplate `json:"data"`
	Paging struct {
		Cursors struct {
			Before string `json:"before"`
			After  string `json:"after"`
		} `json:"cursors"`
		Next string `json:"next"`
	} `json:"paging"`
}

type ParsedMessage struct {
	From          string
	ID            string
	Timestamp     time.Time
	Type          string
	Text          string
	ButtonReplyID string
	ListReplyID   string
	MediaID       string
	MediaMimeType string
	Caption       string
	Filename      string
	ContactName   string
	PhoneNumberID string
	ContextID     string
	SystemType    string
	SystemNewWAID string
}

type ParsedStatus struct {
	MessageID string
	Status    string
	Timestamp time.Time
	UserMsg   string
}

type ParsedTemplateStatus struct {
	WABAID         string
	Event          string
	TemplateName   string
	Language       string
	Reason         string
	MetaTemplateID string
}

func (a Account) Version() string {
	if a.APIVersion == "" {
		return DefaultAPIVersion
	}
	return a.APIVersion
}

func (e *MetaAPIError) Error() string {
	if e.UserMsg != "" {
		return e.UserMsg
	}
	return e.Message
}
