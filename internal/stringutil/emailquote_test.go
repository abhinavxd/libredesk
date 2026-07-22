package stringutil

import "testing"

func TestHTML2TextNoQuotes(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "gmail",
			html: `<div dir="ltr">Thanks, that worked!</div><br><div class="gmail_quote gmail_quote_container"><div dir="ltr" class="gmail_attr">On Mon, Jul 20, 2026 at 2:14 PM Support &lt;support@example.com&gt; wrote:<br></div><blockquote class="gmail_quote" style="margin:0px 0px 0px 0.8ex"><div dir="ltr">Please try restarting the app.</div></blockquote></div>`,
			want: "Thanks, that worked!",
		},
		{
			name: "outlook divRplyFwdMsg with trailing siblings",
			html: `<div>Reply text here</div><div id="appendonsend"></div><hr style="display:inline-block;width:98%"><div id="divRplyFwdMsg"><b>From:</b> Support &lt;support@example.com&gt;</div><div>Old message body</div><div>More old body</div>`,
			want: "Reply text here",
		},
		{
			name: "outlook OLK_SRC_BODY_SECTION",
			html: `<div>New reply</div><div id="OLK_SRC_BODY_SECTION"><div>quoted</div></div><div>trailing quote</div>`,
			want: "New reply",
		},
		{
			name: "outlook message header class",
			html: `<div>Answer</div><div class="OutlookMessageHeader">From: someone</div><div>old thread</div>`,
			want: "Answer",
		},
		{
			name: "yahoo",
			html: `<div>My reply</div><div class="yahoo_quoted"><div>On Monday, Support wrote:</div><div>old</div></div>`,
			want: "My reply",
		},
		{
			name: "apple mail blockquote with dangling attribution",
			html: `<div>Got it, thanks.</div><div><br></div><div>On 5 Jan 2026, at 10:00, Support wrote:</div><blockquote type="cite"><div>previous message</div></blockquote>`,
			want: "Got it, thanks.",
		},
		{
			name: "protonmail",
			html: `<div><div>Hello with quoted.</div><div class="protonmail_signature_block"><div class="protonmail_signature_block-proton">Sent with <a href="https://proton.me/mail/home">Proton Mail</a> secure email.</div></div><div class="protonmail_quote">On Wednesday, July 22nd, 2026 at 9:06 PM, Abhinav &lt;user@example.com&gt; wrote:<br><blockquote class="protonmail_quote" type="cite"><p>Hello!</p></blockquote><br></div></div>`,
			want: "Hello with quoted.\nSent with Proton Mail secure email.",
		},
		{
			name: "quote only returns empty",
			html: `<div class="gmail_quote gmail_quote_container"><blockquote class="gmail_quote"><div>forwarded content</div></blockquote></div>`,
			want: "",
		},
		{
			name: "no quotes passes through",
			html: `<div>Hello, I need help with my order.</div>`,
			want: "Hello, I need help with my order.",
		},
		{
			name: "nested blockquote chain",
			html: `<div>Top reply</div><blockquote><div>level one</div><blockquote><div>level two</div></blockquote></blockquote>`,
			want: "Top reply",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTML2TextNoQuotes(tt.html); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTrimPlainTextQuotes(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "gmail style plaintext",
			text: "New question here\n\nOn Tue, Jul 21, 2026 at 9:00 AM Support <support@example.com> wrote:\n> old reply\n> more old reply",
			want: "New question here",
		},
		{
			name: "original message marker",
			text: "My answer\n\n-----Original Message-----\n> quoted",
			want: "My answer",
		},
		{
			name: "quote only returns empty",
			text: "On Tue wrote:\n> everything quoted",
			want: "",
		},
		{
			name: "inline bottom quotes kept",
			text: "> what is your order id?\nIt is 12345",
			want: "> what is your order id?\nIt is 12345",
		},
		{
			name: "no quotes passes through",
			text: "Just a normal message",
			want: "Just a normal message",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimPlainTextQuotes(tt.text); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
