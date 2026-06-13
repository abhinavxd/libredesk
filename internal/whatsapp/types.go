// Package whatsapp provides a client for the WhatsApp Cloud API and helpers for parsing Meta webhook payloads.
package whatsapp

import "time"

const DefaultAPIVersion = "v21.0"

// Account holds the per-inbox Meta Graph API credentials, already decrypted at the call site.
type Account struct {
	PhoneNumberID string
	WABAID        string
	AccessToken   string
	AppSecret     string
	APIVersion    string
}

// Version returns the API version, falling back to the default.
func (a Account) Version() string {
	if a.APIVersion == "" {
		return DefaultAPIVersion
	}
	return a.APIVersion
}

// MetaAPIError is a structured Graph API error response implementing the error interface.
type MetaAPIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Type       string `json:"type"`
	Code       int    `json:"code"`
	Subcode    int    `json:"error_subcode"`
	UserMsg    string `json:"error_user_msg"`
	FBTraceID  string `json:"fbtrace_id"`
}

func (e *MetaAPIError) Error() string {
	if e.UserMsg != "" {
		return e.UserMsg
	}
	return e.Message
}

// metaErrorEnvelope is the top-level wrapper Meta returns on error responses.
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

// SendResponse is the shape Meta returns from the messages endpoint.
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

// MediaInfo describes a media object fetched from Meta.
type MediaInfo struct {
	URL              string `json:"url"`
	MimeType         string `json:"mime_type"`
	SHA256           string `json:"sha256"`
	FileSize         int64  `json:"file_size"`
	ID               string `json:"id"`
	MessagingProduct string `json:"messaging_product"`
}

// UploadMediaResponse is returned by Meta after a media upload.
type UploadMediaResponse struct {
	ID string `json:"id"`
}

// MetaTemplate is the template shape Meta returns when listing or after submission.
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

// TemplateComponent is a single component (header/body/footer/buttons) in a template.
type TemplateComponent struct {
	Type    string           `json:"type"`
	Format  string           `json:"format,omitempty"`
	Text    string           `json:"text,omitempty"`
	Example map[string]any   `json:"example,omitempty"`
	Buttons []TemplateButton `json:"buttons,omitempty"`
}

// TemplateButton is a button on a template.
type TemplateButton struct {
	Type        string   `json:"type"`
	Text        string   `json:"text,omitempty"`
	URL         string   `json:"url,omitempty"`
	PhoneNumber string   `json:"phone_number,omitempty"`
	Example     []string `json:"example,omitempty"`
}

// TemplateSubmission is the payload sent to Meta when creating a template.
type TemplateSubmission struct {
	Name       string              `json:"name"`
	Language   string              `json:"language"`
	Category   string              `json:"category"`
	Components []TemplateComponent `json:"components"`
}

// templateListResponse is the paginated response from list templates.
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

// ParsedMessage is the flat shape used internally for an inbound WhatsApp message.
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
}

// ParsedStatus is the flat shape used internally for a status webhook event.
type ParsedStatus struct {
	MessageID   string
	Status      string
	Timestamp   time.Time
	RecipientID string
	ErrorCode   int
	Subcode     int
	UserMsg     string
	FBTraceID   string
}

// ParsedTemplateStatus is the flat shape for a template status update event.
type ParsedTemplateStatus struct {
	WABAID         string
	Event          string
	TemplateName   string
	Language       string
	Reason         string
	MetaTemplateID string
}
