package main

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/attachment"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
)

func TestResolveAttachmentCIDs(t *testing.T) {
	tests := []struct {
		name    string
		msg     cmodels.Message
		rootURL string
		want    string
	}{
		{
			name: "replace single cid with attachment url",
			msg: cmodels.Message{
				Content: `<div><img src="cid:conv_abc123" alt="photo.png"></div>`,
				Attachments: attachment.Attachments{
					{ContentID: "conv_abc123", URL: "https://s3.example.com/photo.png?sig=xxx"},
				},
			},
			rootURL: "",
			want:    `<div><img src="https://s3.example.com/photo.png?sig=xxx" alt="photo.png"></div>`,
		},
		{
			name: "replace multiple cids",
			msg: cmodels.Message{
				Content: `<div><img src="cid:img1"><img src="cid:img2"></div>`,
				Attachments: attachment.Attachments{
					{ContentID: "img1", URL: "https://s3.example.com/img1.png"},
					{ContentID: "img2", URL: "https://s3.example.com/img2.png"},
				},
			},
			rootURL: "",
			want:    `<div><img src="https://s3.example.com/img1.png"><img src="https://s3.example.com/img2.png"></div>`,
		},
		{
			name: "same cid referenced multiple times",
			msg: cmodels.Message{
				Content: `<div><img src="cid:dup1"><p>text</p><img src="cid:dup1"></div>`,
				Attachments: attachment.Attachments{
					{ContentID: "dup1", URL: "https://s3.example.com/dup.png"},
				},
			},
			rootURL: "",
			want:    `<div><img src="https://s3.example.com/dup.png"><p>text</p><img src="https://s3.example.com/dup.png"></div>`,
		},
		{
			name: "skip attachment with empty content_id",
			msg: cmodels.Message{
				Content: `<div>no cids here</div>`,
				Attachments: attachment.Attachments{
					{ContentID: "", URL: "https://s3.example.com/file.pdf"},
				},
			},
			rootURL: "",
			want:    `<div>no cids here</div>`,
		},
		{
			name: "skip attachment with empty url",
			msg: cmodels.Message{
				Content: `<div><img src="cid:orphan"></div>`,
				Attachments: attachment.Attachments{
					{ContentID: "orphan", URL: ""},
				},
			},
			rootURL: "",
			want:    `<div><img src="cid:orphan"></div>`,
		},
		{
			name: "no attachments",
			msg: cmodels.Message{
				Content:     `<div><img src="cid:unknown"></div>`,
				Attachments: attachment.Attachments{},
			},
			rootURL: "",
			want:    `<div><img src="cid:unknown"></div>`,
		},
		{
			name: "resolve /uploads/ double quotes to absolute url",
			msg: cmodels.Message{
				Content: `<div><img src="/uploads/abc-uuid"></div>`,
			},
			rootURL: "https://desk.example.com",
			want:    `<div><img src="https://desk.example.com/uploads/abc-uuid"></div>`,
		},
		{
			name: "resolve /uploads/ single quotes to absolute url",
			msg: cmodels.Message{
				Content: `<div><img src='/uploads/abc-uuid'></div>`,
			},
			rootURL: "https://desk.example.com",
			want:    `<div><img src='https://desk.example.com/uploads/abc-uuid'></div>`,
		},
		{
			name: "skip /uploads/ resolution when rootURL is empty",
			msg: cmodels.Message{
				Content: `<div><img src="/uploads/abc-uuid"></div>`,
			},
			rootURL: "",
			want:    `<div><img src="/uploads/abc-uuid"></div>`,
		},
		{
			name: "cid replacement and /uploads/ resolution together",
			msg: cmodels.Message{
				Content: `<div><img src="cid:inline1"><img src="/uploads/existing-uuid"></div>`,
				Attachments: attachment.Attachments{
					{ContentID: "inline1", URL: "https://s3.example.com/inline1.png"},
				},
			},
			rootURL: "https://desk.example.com",
			want:    `<div><img src="https://s3.example.com/inline1.png"><img src="https://desk.example.com/uploads/existing-uuid"></div>`,
		},
		{
			name: "does not replace cid in plain text outside src attribute",
			msg: cmodels.Message{
				Content: `<div>Reference: cid:img1 is the image id</div>`,
				Attachments: attachment.Attachments{
					{ContentID: "img1", URL: "https://s3.example.com/img1.png"},
				},
			},
			rootURL: "",
			want:    `<div>Reference: https://s3.example.com/img1.png is the image id</div>`,
		},
		{
			name: "rootURL with trailing slash is preserved as-is",
			msg: cmodels.Message{
				Content: `<img src="/uploads/uuid1">`,
			},
			rootURL: "https://desk.example.com/",
			want:    `<img src="https://desk.example.com//uploads/uuid1">`,
		},
		{
			name: "real email html with cid in nested divs and gmail quote",
			msg: cmodels.Message{
				Content: `<div><div dir="ltr">Attachments and inline.<br><br><div>Thanks.</div>` +
					`<img src="cid:conv_ii_abc1" alt="screenshot.png" width="562" height="74">` +
					`<img src="cid:conv_ii_abc2" alt="logo.png" width="562" height="73"><br><br>` +
					`<div><br></div></div><br><div class="gmail_quote gmail_quote_container">` +
					`<blockquote class="gmail_quote">quoted text</blockquote></div></div>`,
				Attachments: attachment.Attachments{
					{ContentID: "conv_ii_abc1", URL: "https://s3.example.com/screenshot.png?sig=aaa"},
					{ContentID: "conv_ii_abc2", URL: "https://s3.example.com/logo.png?sig=bbb"},
				},
			},
			rootURL: "",
			want: `<div><div dir="ltr">Attachments and inline.<br><br><div>Thanks.</div>` +
				`<img src="https://s3.example.com/screenshot.png?sig=aaa" alt="screenshot.png" width="562" height="74">` +
				`<img src="https://s3.example.com/logo.png?sig=bbb" alt="logo.png" width="562" height="73"><br><br>` +
				`<div><br></div></div><br><div class="gmail_quote gmail_quote_container">` +
				`<blockquote class="gmail_quote">quoted text</blockquote></div></div>`,
		},
		{
			name: "cid in css background-image inline style",
			msg: cmodels.Message{
				Content: `<div style="background-image: url(cid:bg_img)">content</div>`,
				Attachments: attachment.Attachments{
					{ContentID: "bg_img", URL: "https://s3.example.com/bg.png"},
				},
			},
			rootURL: "",
			want:    `<div style="background-image: url(https://s3.example.com/bg.png)">content</div>`,
		},
		{
			name: "empty content",
			msg: cmodels.Message{
				Content: "",
				Attachments: attachment.Attachments{
					{ContentID: "img1", URL: "https://s3.example.com/img1.png"},
				},
			},
			rootURL: "https://desk.example.com",
			want:    "",
		},
		{
			name: "content_id with special characters",
			msg: cmodels.Message{
				Content: `<img src="cid:conv_uuid_ii_mnoeigle3">`,
				Attachments: attachment.Attachments{
					{ContentID: "conv_uuid_ii_mnoeigle3", URL: "https://s3.example.com/76584918.png"},
				},
			},
			rootURL: "",
			want:    `<img src="https://s3.example.com/76584918.png">`,
		},
		{
			name: "cid reference that does not match any attachment is left as-is",
			msg: cmodels.Message{
				Content: `<img src="cid:unknown_ref"><img src="cid:known_ref">`,
				Attachments: attachment.Attachments{
					{ContentID: "known_ref", URL: "https://s3.example.com/known.png"},
				},
			},
			rootURL: "",
			want:    `<img src="cid:unknown_ref"><img src="https://s3.example.com/known.png">`,
		},
		{
			name: "mixed /uploads/ in src and in text content",
			msg: cmodels.Message{
				Content: `<div>See /uploads/readme.txt for details</div><img src="/uploads/abc-uuid">`,
			},
			rootURL: "https://desk.example.com",
			want:    `<div>See /uploads/readme.txt for details</div><img src="https://desk.example.com/uploads/abc-uuid">`,
		},
		{
			name: "multiple /uploads/ references with mixed quotes",
			msg: cmodels.Message{
				Content: `<img src="/uploads/uuid1"><img src='/uploads/uuid2'>`,
			},
			rootURL: "https://desk.example.com",
			want:    `<img src="https://desk.example.com/uploads/uuid1"><img src='https://desk.example.com/uploads/uuid2'>`,
		},
		{
			name: "attachment url with ampersands in presigned url",
			msg: cmodels.Message{
				Content: `<img src="cid:conv_img1">`,
				Attachments: attachment.Attachments{
					{ContentID: "conv_img1", URL: "https://s3.ap-south-1.amazonaws.com/bucket/uuid?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Expires=300&X-Amz-Signature=abc123"},
				},
			},
			rootURL: "",
			want:    `<img src="https://s3.ap-south-1.amazonaws.com/bucket/uuid?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Expires=300&X-Amz-Signature=abc123">`,
		},
		{
			name: "cid is case sensitive",
			msg: cmodels.Message{
				Content: `<img src="CID:img1"><img src="cid:img1"><img src="Cid:img1">`,
				Attachments: attachment.Attachments{
					{ContentID: "img1", URL: "https://s3.example.com/img1.png"},
				},
			},
			rootURL: "",
			want:    `<img src="CID:img1"><img src="https://s3.example.com/img1.png"><img src="Cid:img1">`,
		},
		{
			name: "content with html entities around cid",
			msg: cmodels.Message{
				Content: `<img src="cid:img1" alt="image &amp; logo">`,
				Attachments: attachment.Attachments{
					{ContentID: "img1", URL: "https://s3.example.com/img1.png"},
				},
			},
			rootURL: "",
			want:    `<img src="https://s3.example.com/img1.png" alt="image &amp; logo">`,
		},
		{
			name: "cid in html comment is still replaced",
			msg: cmodels.Message{
				Content: `<!-- cid:img1 reference --><img src="cid:img1">`,
				Attachments: attachment.Attachments{
					{ContentID: "img1", URL: "https://s3.example.com/img1.png"},
				},
			},
			rootURL: "",
			want:    `<!-- https://s3.example.com/img1.png reference --><img src="https://s3.example.com/img1.png">`,
		},
		{
			name: "nil attachments",
			msg: cmodels.Message{
				Content:     `<img src="cid:img1">`,
				Attachments: nil,
			},
			rootURL: "https://desk.example.com",
			want:    `<img src="cid:img1">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolveAttachmentCIDs(&tt.msg, tt.rootURL)
			if tt.msg.Content != tt.want {
				t.Errorf("resolveAttachmentCIDs()\n got  = %s\n want = %s", tt.msg.Content, tt.want)
			}
		})
	}
}
