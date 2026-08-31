package attachment

import (
	"net/textproto"
	"reflect"
	"testing"
)

func TestMakeHeader(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		contentID   string
		fileName    string
		encoding    string
		disposition string
		want        textproto.MIMEHeader
	}{
		{
			name:        "inline attachment",
			contentType: "image/jpeg",
			contentID:   "123",
			fileName:    "test.jpg",
			encoding:    "base64",
			disposition: "inline",
			want: textproto.MIMEHeader{
				"Content-Disposition":       []string{"inline"},
				"Content-Id":                []string{"<123>"},
				"Content-Type":              []string{"image/jpeg; name=test.jpg"},
				"Content-Transfer-Encoding": []string{"base64"},
			},
		},
		{
			name:        "regular attachment",
			contentType: "application/pdf",
			contentID:   "",
			fileName:    "doc.pdf",
			encoding:    "base64",
			disposition: "attachment",
			want: textproto.MIMEHeader{
				"Content-Disposition":       []string{"attachment; filename=doc.pdf"},
				"Content-Type":              []string{"application/pdf; name=doc.pdf"},
				"Content-Transfer-Encoding": []string{"base64"},
			},
		},
		{
			name:        "content type with params keeps them",
			contentType: "text/plain; charset=utf-8",
			contentID:   "",
			fileName:    "notes.txt",
			encoding:    "base64",
			disposition: "attachment",
			want: textproto.MIMEHeader{
				"Content-Disposition":       []string{"attachment; filename=notes.txt"},
				"Content-Type":              []string{"text/plain; charset=utf-8; name=notes.txt"},
				"Content-Transfer-Encoding": []string{"base64"},
			},
		},
		{
			name:        "default values",
			contentType: "",
			contentID:   "",
			fileName:    "file.txt",
			encoding:    "",
			disposition: "",
			want: textproto.MIMEHeader{
				"Content-Disposition":       []string{"attachment; filename=file.txt"},
				"Content-Type":              []string{"application/octet-stream; name=file.txt"},
				"Content-Transfer-Encoding": []string{"base64"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeHeader(tt.contentType, tt.contentID, tt.fileName, tt.encoding, tt.disposition)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakeHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttachments_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    Attachments
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			want:    Attachments{},
			wantErr: false,
		},
		{
			name:  "valid json",
			input: []byte(`[{"name":"test.jpg","size":1024,"content_type":"image/jpeg","disposition":"attachment"}]`),
			want: Attachments{
				{
					Name:        "test.jpg",
					Size:        1024,
					ContentType: "image/jpeg",
					Disposition: "attachment",
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   "not bytes",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a Attachments
			err := a.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Attachments.Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(a, tt.want) {
				t.Errorf("Attachments.Scan() = %v, want %v", a, tt.want)
			}
		})
	}
}
