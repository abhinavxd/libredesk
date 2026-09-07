package attachment

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/textproto"
)

const (
	DispositionInline     = "inline"
	DispositionAttachment = "attachment"
)

// Attachment represents a file or blob attachment that can be sent or received on a message.
type Attachment struct {
	Name         string               `json:"name"`
	Size         int                  `json:"size"`
	Content      []byte               `json:"content"`
	ContentID    string               `json:"content_id"`
	ContentType  string               `json:"content_type"`
	Disposition  string               `json:"disposition"`
	UUID         string               `json:"uuid"`
	URL          string               `json:"url"`
	ThumbnailURL string               `json:"thumbnail_url"`
	Header       textproto.MIMEHeader `json:"-"`
}

type Attachments []Attachment

func (a *Attachments) Scan(value interface{}) error {
	if value == nil {
		*a = make(Attachments, 0)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Attachments.Scan: type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, a)
}

func (a Attachments) MarshalJSON() ([]byte, error) {
	if a == nil {
		a = make(Attachments, 0)
	}
	return json.Marshal([]Attachment(a))
}

// MakeHeader creates a MIME header for email attachments or inline content.
func MakeHeader(contentType, contentID, fileName, encoding, disposition string) textproto.MIMEHeader {
	if encoding == "" {
		encoding = "base64"
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if disposition == "" {
		disposition = "attachment"
	}

	h := textproto.MIMEHeader{}

	if disposition == "inline" {
		h.Set("Content-Disposition", "inline")
		h.Set("Content-ID", "<"+contentID+">")
	} else {
		h.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": fileName}))
	}

	// Round-trip through ParseMediaType so existing params (e.g. charset) survive alongside name.
	if mt, params, err := mime.ParseMediaType(contentType); err == nil {
		params["name"] = fileName
		h.Set("Content-Type", mime.FormatMediaType(mt, params))
	} else {
		h.Set("Content-Type", contentType)
	}
	h.Set("Content-Transfer-Encoding", encoding)

	return h
}
