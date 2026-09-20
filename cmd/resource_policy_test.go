package main

import (
	"strings"
	"testing"
)

func TestDecodeResourcePolicy(t *testing.T) {
	for _, body := range []string{
		`{"mode":"block_all","allowed_domains":[]}`,
		`{"mode":"load_on_receipt","allowed_domains":[]}`,
		`{"mode":"allowlist","allowed_domains":["IMAGES.example.com"]}`,
	} {
		if _, err := decodeResourcePolicy([]byte(body)); err != nil {
			t.Errorf("valid request rejected: %v", err)
		}
	}
	for _, body := range []string{
		``, `null`, `{}`, `[]`, `{"mode":true}`,
		`{"mode":"allow_all"}`, `{"mode":"allowlist","allowed_domains":["*"]}`,
		`{"mode":"allowlist","allowed_domains":[null]}`,
		`{"mode":"allowlist","allowed_domains":"example.com"}`,
		`{"mode":"allowlist","allowed_domain":["example.com"]}`,
		`{"mode":"block_all"} {"mode":"allowlist"}`, `{"mode":"block_all"} trailing`,
		strings.Repeat(" ", 32769),
	} {
		if _, err := decodeResourcePolicy([]byte(body)); err == nil {
			t.Errorf("invalid request accepted: %q", body)
		}
	}
}
