package models

import (
	"strings"
	"testing"
)

// The reserved name is how the CSAT template is found, and how the delete guard recognises it.
func TestCSATTemplateName(t *testing.T) {
	name := CSATTemplateName(15)
	if name != "libredesk_csat_15" {
		t.Fatalf("unexpected name %q", name)
	}
	if !strings.HasPrefix(name, CSATTemplateNamePrefix) {
		t.Fatalf("%q must carry the reserved prefix", name)
	}
	if CSATTemplateName(15) == CSATTemplateName(16) {
		t.Fatal("each inbox needs its own template name")
	}
}
