package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

const testEncryptionKey = "01234567890123456789012345678901"

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	tr, err := i18n.New([]byte(`{"_.code":"en","_.name":"English"}`))
	if err != nil {
		t.Fatalf("i18n init: %v", err)
	}
	lo := logf.New(logf.Opts{})
	return &Manager{lo: &lo, i18n: tr, encryptionKey: testEncryptionKey}
}

func decodeToolAuth(t *testing.T, raw types.JSONText) models.ToolAuth {
	t.Helper()
	var auth models.ToolAuth
	if err := json.Unmarshal(raw, &auth); err != nil {
		t.Fatalf("unmarshal auth: %v", err)
	}
	return auth
}

func TestPrepareToolAuthNewTool(t *testing.T) {
	m := newTestManager(t)
	raw, _ := json.Marshal(models.ToolAuth{Headers: []models.ToolAuthHeader{
		{Key: "Authorization", Value: "Bearer secret"},
		{Key: "X-Api-Key", Value: "apikey123"},
	}})

	stored, err := m.prepareToolAuth(types.JSONText(raw), nil)
	if err != nil {
		t.Fatalf("prepareToolAuth: %v", err)
	}
	auth := decodeToolAuth(t, stored)
	if len(auth.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(auth.Headers))
	}
	for _, h := range auth.Headers {
		if !crypto.IsEncrypted(h.Value) {
			t.Fatalf("header %q value not encrypted: %q", h.Key, h.Value)
		}
		plain, err := crypto.Decrypt(h.Value, testEncryptionKey)
		if err != nil {
			t.Fatalf("decrypt %q: %v", h.Key, err)
		}
		if h.Key == "Authorization" && plain != "Bearer secret" {
			t.Fatalf("Authorization roundtrip mismatch: %q", plain)
		}
		if h.Key == "X-Api-Key" && plain != "apikey123" {
			t.Fatalf("X-Api-Key roundtrip mismatch: %q", plain)
		}
	}
}

func TestPrepareToolAuthEditKeepsDummyChangesOther(t *testing.T) {
	m := newTestManager(t)
	existingRaw, _ := json.Marshal(models.ToolAuth{Headers: []models.ToolAuthHeader{
		{Key: "Authorization", Value: "Bearer secret"},
		{Key: "X-Api-Key", Value: "apikey123"},
	}})
	existing, err := m.prepareToolAuth(types.JSONText(existingRaw), nil)
	if err != nil {
		t.Fatalf("prepareToolAuth (initial): %v", err)
	}

	dummy := strings.Repeat(stringutil.PasswordDummy, 10)
	incomingRaw, _ := json.Marshal(models.ToolAuth{Headers: []models.ToolAuthHeader{
		{Key: "Authorization", Value: dummy},
		{Key: "X-Api-Key", Value: "new-apikey"},
	}})

	updated, err := m.prepareToolAuth(types.JSONText(incomingRaw), existing)
	if err != nil {
		t.Fatalf("prepareToolAuth (update): %v", err)
	}
	auth := decodeToolAuth(t, updated)
	if len(auth.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(auth.Headers))
	}
	for _, h := range auth.Headers {
		plain, err := crypto.Decrypt(h.Value, testEncryptionKey)
		if err != nil {
			t.Fatalf("decrypt %q: %v", h.Key, err)
		}
		switch h.Key {
		case "Authorization":
			if plain != "Bearer secret" {
				t.Fatalf("Authorization should be unchanged, got %q", plain)
			}
		case "X-Api-Key":
			if plain != "new-apikey" {
				t.Fatalf("X-Api-Key should be updated, got %q", plain)
			}
		default:
			t.Fatalf("unexpected header %q", h.Key)
		}
	}
}

func TestPrepareToolAuthEditRemovesHeader(t *testing.T) {
	m := newTestManager(t)
	existingRaw, _ := json.Marshal(models.ToolAuth{Headers: []models.ToolAuthHeader{
		{Key: "Authorization", Value: "Bearer secret"},
		{Key: "X-Api-Key", Value: "apikey123"},
	}})
	existing, err := m.prepareToolAuth(types.JSONText(existingRaw), nil)
	if err != nil {
		t.Fatalf("prepareToolAuth (initial): %v", err)
	}

	incomingRaw, _ := json.Marshal(models.ToolAuth{Headers: []models.ToolAuthHeader{
		{Key: "Authorization", Value: strings.Repeat(stringutil.PasswordDummy, 10)},
	}})

	updated, err := m.prepareToolAuth(types.JSONText(incomingRaw), existing)
	if err != nil {
		t.Fatalf("prepareToolAuth (remove): %v", err)
	}
	auth := decodeToolAuth(t, updated)
	if len(auth.Headers) != 1 || auth.Headers[0].Key != "Authorization" {
		t.Fatalf("expected only Authorization header to remain, got %+v", auth.Headers)
	}
}

func TestMaskToolAuthMasksAllHeaders(t *testing.T) {
	m := newTestManager(t)
	raw, _ := json.Marshal(models.ToolAuth{Headers: []models.ToolAuthHeader{
		{Key: "Authorization", Value: "Bearer secret"},
		{Key: "X-Api-Key", Value: "apikey123"},
	}})
	stored, err := m.prepareToolAuth(types.JSONText(raw), nil)
	if err != nil {
		t.Fatalf("prepareToolAuth: %v", err)
	}

	masked := maskToolAuth(stored)
	auth := decodeToolAuth(t, masked)
	if len(auth.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(auth.Headers))
	}
	dummy := strings.Repeat(stringutil.PasswordDummy, 10)
	for _, h := range auth.Headers {
		if h.Value != dummy {
			t.Fatalf("header %q value not masked: %q", h.Key, h.Value)
		}
	}
}
