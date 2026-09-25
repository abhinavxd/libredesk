package models

import "testing"

func TestNormalizeContentType(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"image/png", "image/png"},
		{"IMAGE/SVG+XML", "image/svg+xml"},
		{"image/svg+xml; charset=utf-8", "image/svg+xml"},
		{"image/svg+xml; x", "image/svg+xml"},
		{"Image/Svg+Xml; x", "image/svg+xml"},
		{`image/svg+xml ; a="b`, "image/svg+xml"},
		{"text/plain,image/svg+xml", ContentTypeOctetStream},
		{"image", ContentTypeOctetStream},
		{"", ContentTypeOctetStream},
		{"garbage", ContentTypeOctetStream},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := NormalizeContentType(tt.in); got != tt.want {
				t.Fatalf("NormalizeContentType(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestContentDisposition(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"image/png", DispositionInline},
		{"video/mp4", DispositionInline},
		{"application/pdf", DispositionInline},
		{"Application/PDF; x", DispositionInline},
		{"image/svg+xml", DispositionAttachment},
		{"image/svg+xml; x", DispositionAttachment},
		{"IMAGE/SVG+XML", DispositionAttachment},
		{"image/x+xml", DispositionAttachment},
		{"video/x+xml", DispositionAttachment},
		{"application/xhtml+xml", DispositionAttachment},
		{"text/html", DispositionAttachment},
		{"text/xml", DispositionAttachment},
		{"text/plain,image/svg+xml", DispositionAttachment},
		{"", DispositionAttachment},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := ContentDisposition(tt.in); got != tt.want {
				t.Fatalf("ContentDisposition(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
