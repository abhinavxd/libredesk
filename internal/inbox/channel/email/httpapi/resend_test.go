package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func signSvix(t *testing.T, secret, msgID string, ts time.Time, body []byte) http.Header {
	t.Helper()
	secretBytes, err := base64.StdEncoding.DecodeString(secret)
	require.NoError(t, err)

	timestamp := strconv.FormatInt(ts.Unix(), 10)
	signedContent := fmt.Sprintf("%s.%s.%s", msgID, timestamp, body)
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedContent))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	h := http.Header{}
	h.Set("svix-id", msgID)
	h.Set("svix-timestamp", timestamp)
	h.Set("svix-signature", "v1,"+sig)
	return h
}

func TestVerifySvixSignature(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("test-secret-key-32-bytes-long!!"))
	body := []byte(`{"type":"email.received"}`)

	t.Run("valid signature", func(t *testing.T) {
		headers := signSvix(t, secret, "msg_123", time.Now(), body)
		err := verifySvixSignature(secret, headers, body)
		assert.NoError(t, err)
	})

	t.Run("whsec_ prefix stripped", func(t *testing.T) {
		headers := signSvix(t, secret, "msg_123", time.Now(), body)
		err := verifySvixSignature("whsec_"+secret, headers, body)
		assert.NoError(t, err)
	})

	t.Run("tampered body", func(t *testing.T) {
		headers := signSvix(t, secret, "msg_123", time.Now(), body)
		err := verifySvixSignature(secret, headers, []byte(`{"type":"tampered"}`))
		assert.Error(t, err)
	})

	t.Run("wrong secret", func(t *testing.T) {
		headers := signSvix(t, secret, "msg_123", time.Now(), body)
		wrongSecret := base64.StdEncoding.EncodeToString([]byte("wrong-secret-key-32-bytes-long!"))
		err := verifySvixSignature(wrongSecret, headers, body)
		assert.Error(t, err)
	})

	t.Run("expired timestamp", func(t *testing.T) {
		headers := signSvix(t, secret, "msg_123", time.Now().Add(-1*time.Hour), body)
		err := verifySvixSignature(secret, headers, body)
		assert.Error(t, err)
	})

	t.Run("missing headers", func(t *testing.T) {
		err := verifySvixSignature(secret, http.Header{}, body)
		assert.Error(t, err)
	})
}

func TestResendSend(t *testing.T) {
	var capturedReq resendSendRequest
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedReq))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resendSendResponse{ID: "sent-id-123"})
	}))
	defer srv.Close()

	provider := newResendProvider(imodels.HTTPAPIConfig{APIKey: "re_test_key"})
	provider.baseURL = srv.URL

	msg := OutboundEmail{
		From:    "support@example.com",
		To:      []string{"user@example.com"},
		Subject: "Test",
		HTML:    "<p>hi</p>",
		Headers: map[string]string{"Message-ID": "<abc@example.com>"},
		Attachments: []Attachment{
			{Filename: "file.txt", Content: []byte("hello"), ContentType: "text/plain"},
		},
	}

	msgID, err := provider.Send(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, "sent-id-123", msgID)
	assert.Equal(t, "Bearer re_test_key", capturedAuth)
	assert.Equal(t, msg.From, capturedReq.From)
	assert.Equal(t, msg.To, capturedReq.To)
	assert.Equal(t, "<abc@example.com>", capturedReq.Headers["Message-ID"])
	require.Len(t, capturedReq.Attachments, 1)
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("hello")), capturedReq.Attachments[0].Content)

	assert.Equal(t, "resend", provider.Name())
}

func TestResendSendAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(resendErrorResponse{Name: "validation_error", Message: "invalid `to` field"})
	}))
	defer srv.Close()

	provider := newResendProvider(imodels.HTTPAPIConfig{APIKey: "re_test_key"})
	provider.baseURL = srv.URL

	_, err := provider.Send(context.Background(), OutboundEmail{From: "a@example.com", Subject: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid `to` field")
}

func TestParseResendInboundPayload(t *testing.T) {
	body := []byte(`{
		"type": "email.received",
		"data": {
			"email_id": "email_abc123",
			"from": "Jane Doe <jane@example.com>",
			"to": ["support+conv-11111111-1111-4111-8111-111111111111@example.com"],
			"subject": "Re: Help",
			"html": "<p>reply</p>",
			"text": "reply",
			"headers": {"In-Reply-To": "<orig@example.com>", "References": "<orig@example.com> <mid2@example.com>"},
			"attachments": [{"filename": "a.png", "content_type": "image/png", "content": "aGVsbG8="}]
		}
	}`)

	inbound, err := parseResendInboundPayload(body)
	require.NoError(t, err)
	assert.Equal(t, "email_abc123", inbound.MessageID)
	assert.Equal(t, "Jane Doe <jane@example.com>", inbound.From)
	assert.Equal(t, "Re: Help", inbound.Subject)
	assert.Equal(t, "orig@example.com", inbound.InReplyTo)
	assert.Equal(t, []string{"orig@example.com", "mid2@example.com"}, inbound.References)
	require.Len(t, inbound.Attachments, 1)
	assert.Equal(t, "a.png", inbound.Attachments[0].Filename)
	assert.Equal(t, []byte("hello"), inbound.Attachments[0].Content)
}

func TestParseResendInboundPayloadRejectsNonInboundEvents(t *testing.T) {
	for _, eventType := range []string{"email.delivered", "email.bounced", "email.sent", ""} {
		body := []byte(`{"type":"` + eventType + `","data":{"email_id":"email_x"}}`)
		_, err := parseResendInboundPayload(body)
		require.Error(t, err, "event type %q should be rejected", eventType)
	}
}

func TestParseResendHeadersListShape(t *testing.T) {
	raw := json.RawMessage(`[{"name":"In-Reply-To","value":"<x@example.com>"}]`)
	headers := parseResendHeaders(raw)
	assert.Equal(t, "<x@example.com>", headers["In-Reply-To"])
}
