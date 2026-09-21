package main

import (
	"encoding/json"
	"strings"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
)

func TestSetMessageDisplayPreservesOriginal(t *testing.T) {
	content := `<p>Hello</p><img src="https://images.example/pixel">`
	msg := cmodels.Message{Content: content, ContentType: cmodels.ContentTypeHTML}
	setMessageDisplay(&msg, resourcepolicy.Policy{}, nil)
	if msg.Content != content || msg.Display == nil || msg.Display.BlockedImages != 1 {
		t.Fatalf("unexpected message: %#v", msg)
	}
	p, err := resourcepolicy.New(resourcepolicy.Config{Mode: resourcepolicy.Allowlist, AllowedDomains: []string{"images.example"}})
	if err != nil {
		t.Fatal(err)
	}
	setMessageDisplay(&msg, p, nil)
	if msg.Content != content || msg.Display.BlockedImages != 1 || strings.Contains(msg.Display.HTML, `<img`) {
		t.Fatalf("display without gateway must block external images: %#v", msg.Display)
	}
	data, err := json.Marshal(msg)
	if err != nil || !strings.Contains(string(data), `"display":`) {
		t.Fatalf("missing display in response: %s, %v", data, err)
	}
}

func TestSetMessageDisplayEscapesPlainText(t *testing.T) {
	msg := cmodels.Message{Content: `<img src="https://evil.example/x">`, ContentType: cmodels.ContentTypeText}
	setMessageDisplay(&msg, resourcepolicy.Policy{}, nil)
	if strings.Contains(msg.Display.HTML, "<img") || msg.Display.BlockedImages != 0 {
		t.Fatalf("plain text treated as HTML: %#v", msg.Display)
	}
}
