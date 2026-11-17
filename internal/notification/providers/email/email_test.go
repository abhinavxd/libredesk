package email

import (
	"bytes"
	"testing"

	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/jhillyerd/enmime/v2"
	"github.com/stretchr/testify/require"
)

func TestEmailSubjectWhitespace(t *testing.T) {
	sender := &Email{from: "support@example.com"}
	for _, tt := range []struct {
		name, subject, want string
	}{
		{"line breaks", "New reply\n\nPlease help", "New reply Please help"},
		{"CRLF", "New reply\r\nPlease help", "New reply Please help"},
		{"CR", "New reply\rPlease help", "New reply Please help"},
		{"whitespace", " \tNew\t  reply\n", "New reply"},
		{"Unicode", "日本語\nCafé\u2028🙂", "日本語 Café 🙂"},
		{"single line", "New reply #127", "New reply #127"},
		{"empty", "", ""},
		{"blank", "\r\n\t ", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			email := sender.prepareEmail(tt.subject, "Hello\n\nthere\n", []string{"agent@example.com"}, notifier.Message{ContentType: "plain"})
			require.Equal(t, "Hello\n\nthere\n", string(email.Text))
			raw, err := email.Bytes()
			require.NoError(t, err)
			message, err := enmime.ReadEnvelope(bytes.NewReader(raw))
			require.NoError(t, err)
			require.Equal(t, tt.want, message.GetHeader("Subject"))
			require.Equal(t, "Hello\r\n\r\nthere\r\n", message.Text)
		})
	}
}
