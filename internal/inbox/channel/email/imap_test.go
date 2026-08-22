package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/emersion/go-message/mail"
	"github.com/jhillyerd/enmime/v2"
)

func TestEmail_extractUUIDFromReplyAddress(t *testing.T) {
	e := &Email{}

	testCases := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "Valid reply address with UUID",
			address:  "support+550e8400-e29b-41d4-a716-446655440000@example.com",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "Reply address with angle brackets",
			address:  "<support+123e4567-e89b-42d3-a456-426614174000@example.com>",
			expected: "123e4567-e89b-42d3-a456-426614174000",
		},
		{
			name:     "No plus sign in address",
			address:  "support@example.com",
			expected: "",
		},
		{
			name:     "Plus sign but no UUID",
			address:  "support+test@example.com",
			expected: "",
		},
		{
			name:     "Invalid UUID format",
			address:  "support+550e8400-e29b-41d4-a716-44665544000X@example.com",
			expected: "550e8400-e29b-41d4-a716-44665544000X", // extractUUIDFromReplyAddress uses simple format check
		},
		{
			name:     "Empty address",
			address:  "",
			expected: "",
		},
		{
			name:     "UUID too short",
			address:  "support+550e8400-e29b-41d4-a716-4466554400@example.com",
			expected: "",
		},
		{
			name:     "UUID too long",
			address:  "support+550e8400-e29b-41d4-a716-4466554400000@example.com",
			expected: "",
		},
		{
			name:     "Multiple plus signs",
			address:  "support+test+550e8400-e29b-41d4-a716-446655440000@example.com",
			expected: "", // "test+550e8400-e29b-41d4-a716-446655440000" is not 36 chars, so validation fails
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := e.extractUUIDFromReplyAddress(tc.address)
			if result != tc.expected {
				t.Errorf("extractUUIDFromReplyAddress(%q) = %q; expected %q", tc.address, result, tc.expected)
			}
		})
	}
}

// TestGoIMAPMessageIDParsing shows how go-imap fails to parse malformed Message-IDs
// and demonstrates the fallback solution.
// go-imap uses mail.Header.MessageID() which strictly follows RFC 5322 and returns
// empty strings for Message-IDs with multiple @ symbols.
//
// This caused emails to be dropped since we require Message-IDs for deduplication.
// References:
// - https://community.mailcow.email/d/701-multiple-at-in-message-id/5
// - https://github.com/emersion/go-message/issues/154#issuecomment-1425634946
func TestGoIMAPMessageIDParsing(t *testing.T) {
	testCases := []struct {
		input            string
		expectedIMAP     string
		expectedFallback string
		name             string
	}{
		{"<normal@example.com>", "normal@example.com", "normal@example.com", "normal message ID"},
		{"<malformed@@example.com>", "", "malformed@@example.com", "double @ - IMAP fails, fallback works"},
		{"<001c01d710db$a8137a50$f83a6ef0$@jones.smith@example.com>", "", "001c01d710db$a8137a50$f83a6ef0$@jones.smith@example.com", "mailcow-style - IMAP fails, fallback works"},
		{"<test@@@domain.com>", "", "test@@@domain.com", "triple @ - IMAP fails, fallback works"},
		{"  <abc123@example.com>  ", "abc123@example.com", "abc123@example.com", "with whitespace - both handle correctly"},
		{"abc123@example.com", "", "abc123@example.com", "no angle brackets - IMAP fails, fallback works"},
		{"", "", "", "empty input"},
		{"<>", "", "", "empty brackets"},
		{"<CAFnQjQFhY8z@mail.example.com@gateway.company.com>", "", "CAFnQjQFhY8z@mail.example.com@gateway.company.com", "gateway-style - IMAP fails, fallback works"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test go-imap parsing behavior
			var h mail.Header
			h.Set("Message-Id", tc.input)
			imapResult, _ := h.MessageID()

			if imapResult != tc.expectedIMAP {
				t.Errorf("IMAP parsing of %q: expected %q, got %q", tc.input, tc.expectedIMAP, imapResult)
			}

			// Test fallback solution
			if tc.input != "" {
				rawEmail := "From: test@example.com\nMessage-ID: " + tc.input + "\n\nBody"
				envelope, err := enmime.ReadEnvelope(strings.NewReader(rawEmail))
				if err != nil {
					t.Fatal(err)
				}

				fallbackResult := extractMessageIDFromHeaders(envelope)
				if fallbackResult != tc.expectedFallback {
					t.Errorf("Fallback extraction of %q: expected %q, got %q", tc.input, tc.expectedFallback, fallbackResult)
				}

				// Critical check: ensure fallback works when IMAP fails
				if imapResult == "" && tc.expectedFallback != "" && fallbackResult == "" {
					t.Errorf("CRITICAL: Both IMAP and fallback failed for %q - would drop email!", tc.input)
				}
			}
		})
	}
}

// TestCollectAttachments_OutlookInlineImages covers Outlook inline images with a Content-ID but no Content-Disposition, which enmime routes to OtherParts.
func TestCollectAttachments_OutlookInlineImages(t *testing.T) {
	const wantCID = "report-chart@example.com"

	f, err := os.Open(filepath.Join("testdata", "outlook-inline-images.eml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	envelope, err := enmime.ReadEnvelope(f)
	if err != nil {
		t.Fatal(err)
	}

	// The fixture must exercise the OtherParts path, not Inlines/Attachments.
	if len(envelope.OtherParts) != 1 {
		t.Fatalf("expected 1 other part, got %d", len(envelope.OtherParts))
	}

	got := collectAttachments(envelope)

	// The cid referenced in the HTML must resolve to a collected, non-empty inline attachment.
	var att *attachment.Attachment
	for i := range got {
		if got[i].ContentID == wantCID {
			att = &got[i]
			break
		}
	}
	if att == nil {
		t.Fatalf("cid:%s referenced in HTML but not collected as an attachment (got %d attachments)", wantCID, len(got))
	}
	if att.Disposition != attachment.DispositionInline {
		t.Errorf("expected inline disposition, got %q", att.Disposition)
	}
	if att.ContentType != "image/png" {
		t.Errorf("expected content type image/png, got %q", att.ContentType)
	}
	if att.Size == 0 || len(att.Content) == 0 {
		t.Errorf("expected non-empty attachment content, got size=%d len=%d", att.Size, len(att.Content))
	}
}

// TestCollectAttachments_Fixtures runs real EMLs end-to-end through enmime and collectAttachments.
func TestCollectAttachments_Fixtures(t *testing.T) {
	type want struct {
		name        string
		contentID   string
		contentType string
		disposition string
	}
	tests := []struct {
		fixture string
		want    []want
	}{
		{
			// DSN parts (message/delivery-status, text/rfc822-headers) are transport noise.
			fixture: "bounce-dsn.eml",
			want:    nil,
		},
		{
			// The signature blob has no filename and no cid.
			fixture: "pgp-signed.eml",
			want:    nil,
		},
		{
			// The text/calendar alternative body is dropped; the named .ics attachment is kept.
			fixture: "calendar-invite.eml",
			want: []want{
				{name: "invite.ics", contentType: "application/ics", disposition: attachment.DispositionAttachment},
			},
		},
		{
			// Inline image with a cid but no name= param gets a made-up filename.
			fixture: "nameless-inline-image.eml",
			want: []want{
				{name: "attachment.png", contentID: "screenshot@example.com", contentType: "image/png", disposition: attachment.DispositionInline},
			},
		},
		{
			// Explicit filename="" (Gmail) still gets a name based on the content type.
			fixture: "empty-filename-attachments.eml",
			want: []want{
				{name: "attachment.png", contentID: "shot-1@example.com", contentType: "image/png", disposition: attachment.DispositionAttachment},
				{name: "attachment.gif", contentID: "shot-2@example.com", contentType: "image/gif", disposition: attachment.DispositionAttachment},
			},
		},
		{
			// Apple Mail inline disposition without a cid (and no text body) is a plain attachment.
			fixture: "apple-inline-no-cid.eml",
			want: []want{
				{name: "photo.png", contentType: "image/png", disposition: attachment.DispositionAttachment},
				{name: "document.pdf", contentType: "application/pdf", disposition: attachment.DispositionAttachment},
			},
		},
		{
			// Explicit attachment disposition wins over the cid.
			fixture: "attachment-with-cid.eml",
			want: []want{
				{name: "photo.png", contentID: "photo-cid@example.com", contentType: "image/png", disposition: attachment.DispositionAttachment},
			},
		},
		{
			fixture: "mixed-attachments.eml",
			want: []want{
				{name: "logo.png", contentID: "logo@example.com", contentType: "image/png", disposition: attachment.DispositionInline},
				{name: "invoice.pdf", contentType: "application/pdf", disposition: attachment.DispositionAttachment},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			f, err := os.Open(filepath.Join("testdata", tt.fixture))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			envelope, err := enmime.ReadEnvelope(f)
			if err != nil {
				t.Fatal(err)
			}

			got := collectAttachments(envelope)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d attachments, got %d: %+v", len(tt.want), len(got), got)
			}

			byName := map[string]attachment.Attachment{}
			for _, a := range got {
				byName[a.Name] = a
			}
			for _, w := range tt.want {
				a, ok := byName[w.name]
				if !ok {
					t.Errorf("attachment %q not collected", w.name)
					continue
				}
				if a.ContentID != w.contentID {
					t.Errorf("%s: expected cid %q, got %q", w.name, w.contentID, a.ContentID)
				}
				if a.ContentType != w.contentType {
					t.Errorf("%s: expected content type %q, got %q", w.name, w.contentType, a.ContentType)
				}
				if a.Disposition != w.disposition {
					t.Errorf("%s: expected disposition %q, got %q", w.name, w.disposition, a.Disposition)
				}
				if a.Size == 0 || len(a.Content) == 0 {
					t.Errorf("%s: expected non-empty content, got size=%d len=%d", w.name, a.Size, len(a.Content))
				}
			}
		})
	}
}

// TestCollectAttachments_Dispositions verifies disposition and filtering across the three enmime buckets.
func TestCollectAttachments_Dispositions(t *testing.T) {
	env := &enmime.Envelope{
		Attachments: []*enmime.Part{
			{FileName: "doc.pdf", ContentType: "application/pdf", ContentID: "", Content: []byte("x")},
		},
		Inlines: []*enmime.Part{
			{FileName: "logo.png", ContentType: "image/png", ContentID: "logo@id", Content: []byte("x")},
		},
		OtherParts: []*enmime.Part{
			{FileName: "inline.png", ContentType: "image/png", ContentID: "inline@id", Content: []byte("x")},
			{FileName: "noid.png", ContentType: "image/png", ContentID: "", Content: []byte("x")},
			{FileName: "", ContentType: "image/png", ContentID: "nameless@id", Content: []byte("x")},
			{FileName: "", ContentType: "message/delivery-status", ContentID: "", Content: []byte("x")},
		},
	}

	got := collectAttachments(env)
	if len(got) != 5 {
		t.Fatalf("expected 5 attachments, got %d", len(got))
	}

	byCID := map[string]attachment.Attachment{}
	byName := map[string]attachment.Attachment{}
	for _, a := range got {
		byName[a.Name] = a
		byCID[a.ContentID] = a
	}

	cases := map[string]string{
		"doc.pdf":    attachment.DispositionAttachment, // real attachment
		"logo.png":   attachment.DispositionInline,     // inline w/ cid
		"inline.png": attachment.DispositionInline,     // other part w/ cid -> inline
		"noid.png":   attachment.DispositionAttachment, // other part w/o cid but named -> attachment
	}
	for name, want := range cases {
		if got := byName[name].Disposition; got != want {
			t.Errorf("%s: expected disposition %q, got %q", name, want, got)
		}
	}

	// A nameless part with a cid gets a made-up filename; a nameless cid-less part is dropped.
	nameless, ok := byCID["nameless@id"]
	if !ok {
		t.Fatal("nameless part with cid was not collected")
	}
	if nameless.Name == "" || nameless.Disposition != attachment.DispositionInline {
		t.Errorf("nameless cid part: expected synthesized name and inline disposition, got name=%q disposition=%q", nameless.Name, nameless.Disposition)
	}
	for _, a := range got {
		if a.ContentType == "message/delivery-status" {
			t.Errorf("delivery-status part should have been dropped, got %+v", a)
		}
	}
}

// TestEdgeCasesMessageID tests additional edge cases for Message-ID extraction.
func TestEdgeCasesMessageID(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name: "no Message-ID header",
			email: `From: test@example.com
To: inbox@test.com
Subject: Test

Body`,
			expected: "",
		},
		{
			name: "malformed header syntax",
			email: `From: test@example.com
Message-ID: malformed-no-brackets@@domain.com
To: inbox@test.com

Body`,
			expected: "malformed-no-brackets@@domain.com",
		},
		{
			name: "multiple Message-ID headers (first wins)",
			email: `From: test@example.com
Message-ID: <first@example.com>
Message-ID: <second@@example.com>
To: inbox@test.com

Body`,
			expected: "first@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope, err := enmime.ReadEnvelope(strings.NewReader(tt.email))
			if err != nil {
				t.Fatal(err)
			}

			result := extractMessageIDFromHeaders(envelope)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestMimeParser_Charsets asserts that the declared charset is honoured across transfer encodings.
func TestMimeParser_Charsets(t *testing.T) {
	const headers = "From: a@example.com\r\nTo: b@example.com\r\nSubject: test\r\n"

	tests := []struct {
		name        string
		raw         string
		wantText    string
		wantHTML    string
		wantSubject string
	}{
		{
			name: "quoted printable utf8 long mostly ascii body",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"Bom dia. Segue em anexo o documento solicitado ontem pela manha. Qualquer duvida, estou a disposicao para ajudar. Obrigado e a=C3=AD?\r\n",
			wantText: "Bom dia. Segue em anexo o documento solicitado ontem pela manha. Qualquer duvida, estou a disposicao para ajudar. Obrigado e aí?\r\n",
		},
		{
			name: "quoted printable utf8 short body",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\nSim, pra mim deu certo, e a=C3=AD?\r\n",
			wantText: "Sim, pra mim deu certo, e aí?\r\n",
		},
		{
			name: "quoted printable soft line break",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\nn=C3=A3o quero mais=\r\n problemas\r\n",
			wantText: "não quero mais problemas\r\n",
		},
		{
			name: "base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nT2zDoSwgdHVkbyBiZW0/IFNlZ3VlIG8gcmVsYXTDs3JpbyBlbSBhbmV4by4=\r\n",
			wantText: "Olá, tudo bem? Segue o relatório em anexo.",
		},
		{
			name: "8bit utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: 8bit\r\n\r\nOlá, tudo bem?\r\n",
			wantText: "Olá, tudo bem?\r\n",
		},
		{
			name: "7bit ascii",
			raw: headers + "Content-Type: text/plain; charset=\"US-ASCII\"\r\n" +
				"Content-Transfer-Encoding: 7bit\r\n\r\nPlease find the invoice attached.\r\n",
			wantText: "Please find the invoice attached.\r\n",
		},
		{
			name:     "no content type declared",
			raw:      headers + "\r\nPlain body with no content type header at all.\r\n",
			wantText: "Plain body with no content type header at all.\r\n",
		},
		{
			name: "declared iso-8859-1",
			raw: headers + "Content-Type: text/plain; charset=\"ISO-8859-1\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\ne a=ED?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name: "declared windows-1252",
			raw: headers + "Content-Type: text/plain; charset=\"windows-1252\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n=93quoted=94 and =80\r\n",
			wantText: "“quoted” and €\r\n",
		},
		{
			name: "multipart alternative utf8",
			raw: headers + "Content-Type: multipart/alternative; boundary=\"bnd\"\r\n\r\n" +
				"--bnd\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\nSim, e a=C3=AD?\r\n" +
				"--bnd\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n<div dir=3D\"ltr\">Sim, e a=C3=AD?</div>\r\n" +
				"--bnd--\r\n",
			wantText: "Sim, e aí?",
			wantHTML: "<div dir=\"ltr\">Sim, e aí?</div>",
		},
		{
			name: "encoded word subject base64",
			raw: "From: a@example.com\r\nTo: b@example.com\r\n" +
				"Subject: =?UTF-8?B?UmVsYXTDs3JpbyBtZW5zYWw=?=\r\n" +
				"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\nbody\r\n",
			wantText:    "body\r\n",
			wantSubject: "Relatório mensal",
		},
		{
			name: "french quoted printable utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"Bonjour, veuillez trouver ci-joint le re=C3=A7u. =C3=80 bient=C3=B4t, merci=\r\n beaucoup pour votre fid=C3=A9lit=C3=A9 !\r\n",
			wantText: "Bonjour, veuillez trouver ci-joint le reçu. À bientôt, merci beaucoup pour votre fidélité !\r\n",
		},
		{
			name: "german quoted printable utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"Gr=C3=BC=C3=9Fe aus M=C3=BCnchen, die =C3=84nderung wurde durchgef=C3=BChrt=\r\n. Vielen Dank f=C3=BCr Ihre Geduld!\r\n",
			wantText: "Grüße aus München, die Änderung wurde durchgeführt. Vielen Dank für Ihre Geduld!\r\n",
		},
		{
			name: "spanish quoted printable utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"=C2=BFC=C3=B3mo est=C3=A1s? El a=C3=B1o pasado compr=C3=A9 el se=C3=B1uelo =\r\ny funcion=C3=B3 de maravilla.\r\n",
			wantText: "¿Cómo estás? El año pasado compré el señuelo y funcionó de maravilla.\r\n",
		},
		{
			name: "russian quoted printable utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"=D0=97=D0=B4=D1=80=D0=B0=D0=B2=D1=81=D1=82=D0=B2=D1=83=D0=B9=D1=82=D0=B5! =\r\n=D0=9F=D0=BE=D0=B4=D1=81=D0=BA=D0=B0=D0=B6=D0=B8=D1=82=D0=B5, =D0=BF=D0=BE=\r\n=D0=B6=D0=B0=D0=BB=D1=83=D0=B9=D1=81=D1=82=D0=B0, =D0=BA=D0=B0=D0=BA =D0=BD=\r\n=D0=B0=D1=81=D1=82=D1=80=D0=BE=D0=B8=D1=82=D1=8C =D0=BF=D0=BE=D1=87=D1=82=\r\n=D0=BE=D0=B2=D1=8B=D0=B9 =D1=8F=D1=89=D0=B8=D0=BA?\r\n",
			wantText: "Здравствуйте! Подскажите, пожалуйста, как настроить почтовый ящик?\r\n",
		},
		{
			name: "japanese base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"44GK5LiW6Kmx44Gr44Gq44Gj44Gm44GK44KK44G+44GZ44CC5re75LuY44OV44Kh44Kk44Or44KS44GU56K66KqN44GP44Gg44GV44GE44CC\r\n",
			wantText: "お世話になっております。添付ファイルをご確認ください。",
		},
		{
			name: "chinese base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"5oKo5aW977yM6K+35p+l5pS26ZmE5Lu25Lit55qE5Y+R56Wo77yM6LCi6LCi77yB\r\n",
			wantText: "您好，请查收附件中的发票，谢谢！",
		},
		{
			name: "korean base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"7JWI64WV7ZWY7IS47JqULiDssqjrtoAg7YyM7J287J2EIO2ZleyduO2VtCDso7zshLjsmpQu\r\n",
			wantText: "안녕하세요. 첨부 파일을 확인해 주세요.",
		},
		{
			name: "emoji quoted printable utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"Thanks a lot! =F0=9F=8E=89=F0=9F=91=8D Works great now =F0=9F=9A=80\r\n",
			wantText: "Thanks a lot! 🎉👍 Works great now 🚀\r\n",
		},
		{
			name: "lowercase charset name",
			raw: headers + "Content-Type: text/plain; charset=utf-8\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\ne a=C3=AD?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name: "declared shift_jis base64",
			raw: headers + "Content-Type: text/plain; charset=\"Shift_JIS\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\ngqiQophigsmCyILBgsSCqILogtyCt4FC\r\n",
			wantText: "お世話になっております。",
		},
		{
			name: "declared iso-2022-jp base64",
			raw: headers + "Content-Type: text/plain; charset=\"ISO-2022-JP\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nGyRCJCpAJE9DJEskSiRDJEYkKiRqJF4kOSEjGyhC\r\n",
			wantText: "お世話になっております。",
		},
		{
			name: "declared euc-kr base64",
			raw: headers + "Content-Type: text/plain; charset=\"EUC-KR\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nvsiz58fPvLy/5C4gyK7AziC6zsW5teW4s7TPtNku\r\n",
			wantText: "안녕하세요. 확인 부탁드립니다.",
		},
		{
			name: "declared gb2312 base64",
			raw: headers + "Content-Type: text/plain; charset=\"GB2312\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nxPq6w6Osx+uy6crVuL28/qGj\r\n",
			wantText: "您好，请查收附件。",
		},
		{
			name: "declared koi8-r base64",
			raw: headers + "Content-Type: text/plain; charset=\"KOI8-R\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n+sTSwdfT1NfVytTFLCDQ0s/XxdLY1MUg18zP1sXOycUu\r\n",
			wantText: "Здравствуйте, проверьте вложение.",
		},
		{
			name: "declared windows-1251 base64",
			raw: headers + "Content-Type: text/plain; charset=\"windows-1251\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nx+Tw4OLx8uLz6fLlLCDv8O7i5fD88uUg4uvu5uXt6OUu\r\n",
			wantText: "Здравствуйте, проверьте вложение.",
		},
		{
			name: "declared iso-8859-7 greek base64",
			raw: headers + "Content-Type: text/plain; charset=\"ISO-8859-7\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nyuHr5+zd8eEsIOX19+Hx6fP0/iDw7+v9Lg==\r\n",
			wantText: "Καλημέρα, ευχαριστώ πολύ.",
		},
		{
			name: "declared iso-8859-9 turkish base64",
			raw: headers + "Content-Type: text/plain; charset=\"ISO-8859-9\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nTWVyaGFiYSwgaWxnaWxpIGRvc3lhIGVrdGVkaXIuIFRl/mVra/xybGVyLCBpeWkg52Fs/f5tYWxhci4=\r\n",
			wantText: "Merhaba, ilgili dosya ektedir. Teşekkürler, iyi çalışmalar.",
		},
		{
			name: "unknown charset passes utf8 through",
			raw: headers + "Content-Type: text/plain; charset=\"x-nonsense\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\ne a=C3=AD?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name: "malformed doubled charset param salvaged",
			raw: headers + "Content-Type: text/plain; charset=\"charset=UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\ne a=C3=AD?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name: "chinese quoted printable utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"=E6=82=A8=E5=A5=BD=EF=BC=8C=E6=84=9F=E8=B0=A2=E6=82=A8=E7=9A=84=E6=9D=A5=E4=\r\n=BF=A1=EF=BC=8C=E6=88=91=E4=BB=AC=E4=BC=9A=E5=B0=BD=E5=BF=AB=E5=9B=9E=E5=A4=\r\n=8D=E3=80=82\r\n",
			wantText: "您好，感谢您的来信，我们会尽快回复。\r\n",
		},
		{
			name: "declared big5 traditional chinese base64",
			raw: headers + "Content-Type: text/plain; charset=\"Big5\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nsXqmbqFBvdCsZKasqv6l86FBwcLBwqFJ\r\n",
			wantText: "您好，請查收附件，謝謝！",
		},
		{
			name: "declared gbk base64",
			raw: headers + "Content-Type: text/plain; charset=\"GBK\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nxPq6w6Osx+uy6crVuL28/sDvtcSxqLzbtaWjrNC70LujoQ==\r\n",
			wantText: "您好，请查收附件里的报价单，谢谢！",
		},
		{
			name: "declared euc-jp base64",
			raw: headers + "Content-Type: text/plain; charset=\"EUC-JP\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\npKrApM/DpMukyqTDpMakqqTqpN6kuaGj\r\n",
			wantText: "お世話になっております。",
		},
		{
			name: "hindi base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"4KSo4KSu4KS44KWN4KSk4KWHLCDgpJXgpYPgpKrgpK/gpL4g4KS44KSC4KSy4KSX4KWN4KSoIOCkq+CkvOCkvuCkh+CksiDgpKbgpYfgpJbgpYfgpILgpaQg4KSn4KSo4KWN4KSv4KS14KS+4KSmIQ==\r\n",
			wantText: "नमस्ते, कृपया संलग्न फ़ाइल देखें। धन्यवाद!",
		},
		{
			name: "arabic base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"2YXYsdit2KjYp9mL2Iwg2YrYsdis2Ykg2KfZhNin2LfZhNin2Lkg2LnZhNmJINin2YTZhdix2YHZgi4g2LTZg9ix2KfZiyDYrNiy2YrZhNin2Ysh\r\n",
			wantText: "مرحباً، يرجى الاطلاع على المرفق. شكراً جزيلاً!",
		},
		{
			name: "declared windows-1256 arabic base64",
			raw: headers + "Content-Type: text/plain; charset=\"windows-1256\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n49HNyMehINTf0ccgzNLt4ccu\r\n",
			wantText: "مرحبا، شكرا جزيلا.",
		},
		{
			name: "hebrew base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"16nXnNeV150sINeQ16DXkCDXqNeQ15Qg15DXqiDXlNen15XXkdelINeU157XpteV16jXoy4g16rXldeT15Qg16jXkdeUIQ==\r\n",
			wantText: "שלום, אנא ראה את הקובץ המצורף. תודה רבה!",
		},
		{
			name: "declared iso-8859-8 hebrew base64",
			raw: headers + "Content-Type: text/plain; charset=\"ISO-8859-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n+ezl7Swg+uXj5CD44eQu\r\n",
			wantText: "שלום, תודה רבה.",
		},
		{
			name: "thai base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"4Liq4Lin4Lix4Liq4LiU4Li14LiE4Lij4Lix4LiaIOC4geC4o+C4uOC4k+C4suC4leC4o+C4p+C4iOC4quC4reC4muC5hOC4n+C4peC5jOC5geC4meC4miDguILguK3guJrguITguLjguJPguITguKPguLHguJo=\r\n",
			wantText: "สวัสดีครับ กรุณาตรวจสอบไฟล์แนบ ขอบคุณครับ",
		},
		{
			name: "declared windows-874 thai base64",
			raw: headers + "Content-Type: text/plain; charset=\"windows-874\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nysfRyrTVpMPRuiCizbqk2LOkw9G6\r\n",
			wantText: "สวัสดีครับ ขอบคุณครับ",
		},
		{
			name: "vietnamese quoted printable utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"Xin ch=C3=A0o, vui l=C3=B2ng ki=E1=BB=83m tra t=E1=BB=87p =C4=91=C3=ADnh k=\r\n=C3=A8m. C=E1=BA=A3m =C6=A1n nhi=E1=BB=81u!\r\n",
			wantText: "Xin chào, vui lòng kiểm tra tệp đính kèm. Cảm ơn nhiều!\r\n",
		},
		{
			name: "declared iso-8859-2 polish base64",
			raw: headers + "Content-Type: text/plain; charset=\"ISO-8859-2\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nRHppZfEgZG9icnksIHcgemGzsWN6bmlrdSBwcnplc3mzYW0gZmFrdHVy6i4gUG96ZHJhd2lhbSE=\r\n",
			wantText: "Dzień dobry, w załączniku przesyłam fakturę. Pozdrawiam!",
		},
		{
			name: "czech quoted printable utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"Dobr=C3=BD den, v p=C5=99=C3=ADloze zas=C3=ADl=C3=A1m po=C5=BEadovan=C3=A9 =\r\ndokumenty. D=C4=9Bkuji!\r\n",
			wantText: "Dobrý den, v příloze zasílám požadované dokumenty. Děkuji!\r\n",
		},
		{
			name: "declared koi8-u ukrainian base64",
			raw: headers + "Content-Type: text/plain; charset=\"KOI8-U\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n5M/C0s/HzyDEztEsIMTRy9XAINrBINemxNDP16bE2C4=\r\n",
			wantText: "Доброго дня, дякую за відповідь.",
		},
		{
			name: "greek quoted printable utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"=CE=9A=CE=B1=CE=BB=CE=B7=CE=BC=CE=AD=CF=81=CE=B1 =CF=83=CE=B1=CF=82, =CE=B5=\r\n=CF=80=CE=B9=CF=83=CF=85=CE=BD=CE=AC=CF=80=CF=84=CF=89 =CF=84=CE=BF =CE=B1=\r\n=CF=81=CF=87=CE=B5=CE=AF=CE=BF. =CE=95=CF=85=CF=87=CE=B1=CF=81=CE=B9=CF=83=\r\n=CF=84=CF=8E =CF=80=CE=BF=CE=BB=CF=8D!\r\n",
			wantText: "Καλημέρα σας, επισυνάπτω το αρχείο. Ευχαριστώ πολύ!\r\n",
		},
		{
			name: "mixed cjk and latin quoted printable utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n" +
				"Hello =E4=BD=A0=E5=A5=BD =E3=81=93=E3=82=93=E3=81=AB=E3=81=A1=E3=81=AF =EC=\r\n=95=88=EB=85=95=ED=95=98=EC=84=B8=EC=9A=94 =C3=87a va? Gr=C3=BC=C3=9Fe!\r\n",
			wantText: "Hello 你好 こんにちは 안녕하세요 Ça va? Grüße!\r\n",
		},
		{
			name: "declared gb18030 base64",
			raw: headers + "Content-Type: text/plain; charset=\"GB18030\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nxOO6w6Os1eLKx0dCMTgwMzCx4MLrtcSy4srUz/vPoqGj\r\n",
			wantText: "你好，这是GB18030编码的测试消息。",
		},
		{
			name: "declared windows-1250 czech base64",
			raw: headers + "Content-Type: text/plain; charset=\"windows-1250\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nRG9icv0gZGVuLCBk7Gt1amkgemEgb2Rwb3bs7y4=\r\n",
			wantText: "Dobrý den, děkuji za odpověď.",
		},
		{
			name: "declared iso-8859-15 euro sign base64",
			raw: headers + "Content-Type: text/plain; charset=\"ISO-8859-15\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nUHJpeCA6IDQyIKQgc2V1bGVtZW50Lg==\r\n",
			wantText: "Prix : 42 € seulement.",
		},
		{
			name: "farsi base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"2LPZhNin2YXYjCDZhNi32YHYp9mLINmB2KfbjNmEINm+24zZiNiz2Kog2LHYpyDYqNix2LHYs9uMINqp2YbbjNivLiDZhdiq2LTaqdix2YUh\r\n",
			wantText: "سلام، لطفاً فایل پیوست را بررسی کنید. متشکرم!",
		},
		{
			name: "tamil base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"4K614K6j4K6V4K+N4K6V4K6u4K+NLCDgrofgrqPgr4jgrqrgr43grqrgr4jgrprgr40g4K6a4K6w4K6/4K6q4K6+4K6w4K+N4K6V4K+N4K6V4K614K+B4K6u4K+NLiDgrqjgrqngr43grrHgrr8h\r\n",
			wantText: "வணக்கம், இணைப்பைச் சரிபார்க்கவும். நன்றி!",
		},
		{
			name: "bengali base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"4Kao4Kau4Ka44KeN4KaV4Ka+4KawLCDgprjgpoLgpq/gp4HgppXgp43gpqTgpr/gpp/gpr8g4Kam4KeH4KaW4KeB4Kao4KWkIOCmp+CmqOCnjeCmr+CmrOCmvuCmpiE=\r\n",
			wantText: "নমস্কার, সংযুক্তিটি দেখুন। ধন্যবাদ!",
		},
		{
			name: "georgian base64 utf8",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n" +
				"4YOS4YOQ4YOb4YOQ4YOg4YOv4YOd4YOR4YOQLCDhg5Lhg5vhg5Dhg5Phg5rhg53hg5Hhg5cg4YOe4YOQ4YOh4YOj4YOu4YOY4YOh4YOX4YOV4YOY4YOhIQ==\r\n",
			wantText: "გამარჯობა, გმადლობთ პასუხისთვის!",
		},
		{
			name: "declared utf-16 with le bom base64",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-16\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n//5PAGwA4QAsACAAdABlAHMAdABlACAAVQBUAEYALQAxADYALgA=\r\n",
			wantText: "\ufeffOlá, teste UTF-16.",
		},
		{
			name: "utf8 bom preserved in body",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\n77u/T2zDoSBjb20gQk9N\r\n",
			wantText: "\ufeffOlá com BOM",
		},
		{
			name: "empty charset param",
			raw: headers + "Content-Type: text/plain; charset=\"\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\ne a=C3=AD?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name: "charset alias utf8 without hyphen",
			raw: headers + "Content-Type: text/plain; charset=utf8\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\ne a=C3=AD?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name: "charset alias latin1",
			raw: headers + "Content-Type: text/plain; charset=latin1\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\ne a=ED?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name:     "charset alias ansi_x3.4-1968",
			raw:      headers + "Content-Type: text/plain; charset=\"ansi_x3.4-1968\"\r\n\r\nplain ascii here\r\n",
			wantText: "plain ascii here\r\n",
		},
		{
			name: "uppercase transfer encoding",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: QUOTED-PRINTABLE\r\n\r\ne a=C3=AD?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name: "base64 with whitespace and missing padding",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: base64\r\n\r\nT2zDoSwg  dHVkbyBiZW0/IFNl\r\nZ3VlIG8gcmVsYXTDs3JpbyBlbSBhbmV4by4\r\n",
			wantText: "Olá, tudo bem? Segue o relatório em anexo.",
		},
		{
			name: "quoted printable lowercase hex",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\ne a=c3=ad?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name: "quoted printable invalid escape and trailing equals",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\n100=G1% sure=\r\n",
			wantText: "100=G1% sure",
		},
		{
			name: "quoted printable multibyte split across soft break",
			raw: headers + "Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\ne a=C3=\r\n=AD?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name: "format flowed params after charset",
			raw: headers + "Content-Type: text/plain; charset=UTF-8; format=flowed; delsp=yes\r\n" +
				"Content-Transfer-Encoding: quoted-printable\r\n\r\ne a=C3=AD?\r\n",
			wantText: "e aí?\r\n",
		},
		{
			name: "subparts with different charsets latin1 plain utf8 html",
			raw: headers + "Content-Type: multipart/alternative; boundary=\"b\"\r\n\r\n" +
				"--b\r\nContent-Type: text/plain; charset=\"ISO-8859-1\"\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\ne a=ED?\r\n" +
				"--b\r\nContent-Type: text/html; charset=\"UTF-8\"\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n<div>e a=C3=AD?</div>\r\n--b--\r\n",
			wantText: "e aí?",
			wantHTML: "<div>e aí?</div>",
		},
		{
			name: "subparts with different charsets utf8 plain shift_jis html",
			raw: headers + "Content-Type: multipart/alternative; boundary=\"b\"\r\n\r\n" +
				"--b\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\nContent-Transfer-Encoding: base64\r\n\r\n44GK5LiW6Kmx44Gr44Gq44Gj44Gm44GK44KK44G+44GZ44CC\r\n" +
				"--b\r\nContent-Type: text/html; charset=\"Shift_JIS\"\r\nContent-Transfer-Encoding: base64\r\n\r\nPGh0bWw+PGJvZHk+gqiMqZDPguCC6IKgguiCqoLGgqSCsoK0gqKC3IK3gUI8L2JvZHk+PC9odG1sPg==\r\n--b--\r\n",
			wantText: "お世話になっております。",
			wantHTML: "<html><body>お見積もりありがとうございます。</body></html>",
		},
		{
			name: "html meta-only charset shift_jis detected",
			raw: headers + "Content-Type: text/html\r\nContent-Transfer-Encoding: base64\r\n\r\n" +
				"PGh0bWw+PGhlYWQ+PG1ldGEgY2hhcnNldD0ic2hpZnRfamlzIj48L2hlYWQ+PGJvZHk+gqiMqZDPguCC6IKgguiCqoLGgqSCsoK0gqKC3IK3gUI8L2JvZHk+PC9odG1sPg==\r\n",
			wantText: "お見積もりありがとうございます。",
			wantHTML: "<html><head><meta charset=\"shift_jis\"></head><body>お見積もりありがとうございます。</body></html>",
		},
		{
			name: "subject encoded word gb2312",
			raw: "From: a@example.com\r\nTo: b@example.com\r\nSubject: =?GB2312?B?sai827Wl?=\r\n" +
				"Content-Type: text/plain\r\n\r\nbody\r\n",
			wantText:    "body\r\n",
			wantSubject: "报价单",
		},
		{
			name: "subject encoded word koi8-r",
			raw: "From: a@example.com\r\nTo: b@example.com\r\nSubject: =?KOI8-R?B?896j1A==?=\r\n" +
				"Content-Type: text/plain\r\n\r\nbody\r\n",
			wantText:    "body\r\n",
			wantSubject: "Счёт",
		},
		{
			name: "subject folded adjacent encoded words",
			raw: "From: a@example.com\r\nTo: b@example.com\r\nSubject: =?UTF-8?Q?Relat=C3=B3rio_?=\r\n =?UTF-8?Q?mensal_de_agosto?=\r\n" +
				"Content-Type: text/plain\r\n\r\nbody\r\n",
			wantText:    "body\r\n",
			wantSubject: "Relatório mensal de agosto",
		},
		{
			name: "encoded word subject quoted printable",
			raw: "From: a@example.com\r\nTo: b@example.com\r\n" +
				"Subject: =?UTF-8?Q?Relat=C3=B3rio_mensal?=\r\n" +
				"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\nbody\r\n",
			wantText:    "body\r\n",
			wantSubject: "Relatório mensal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope, err := mimeParser.ReadEnvelope(strings.NewReader(tt.raw))
			if err != nil {
				t.Fatalf("reading envelope: %v", err)
			}
			if envelope.Text != tt.wantText {
				t.Errorf("text: got %q, want %q", envelope.Text, tt.wantText)
			}
			if envelope.HTML != tt.wantHTML {
				t.Errorf("html: got %q, want %q", envelope.HTML, tt.wantHTML)
			}
			if tt.wantSubject != "" && envelope.GetHeader("Subject") != tt.wantSubject {
				t.Errorf("subject: got %q, want %q", envelope.GetHeader("Subject"), tt.wantSubject)
			}
		})
	}
}
