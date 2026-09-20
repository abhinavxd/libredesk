package livechat

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/logf"
)

type imageTestUsers struct{ inbox.UserStore }

func (imageTestUsers) GetAgent(int, string) (umodels.User, error) {
	return umodels.User{}, nil
}

func TestSendDisplaysAuthorizedInlineAttachment(t *testing.T) {
	lo := logf.New(logf.Opts{})
	lc, err := New(nil, imageTestUsers{}, Opts{Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{Channel: make(chan []byte, 1)}
	lc.clients["1"] = []*Client{client}
	const id = "12345678-1234-4234-9234-123456789abc"
	const imageURL = "https://desk.example/uploads/" + id + "?sig=current"
	content := `<p>Screenshot:</p><img src="cid:screenshot"><img src="https://tracker.example/pixel">`
	err = lc.Send(models.OutboundMessage{
		MessageReceiverID: 1,
		ContentType:       models.ContentTypeHTML,
		Content:           content,
		Attachments: attachment.Attachments{{
			UUID: id, ContentID: "screenshot", ContentType: "image/png", URL: imageURL,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Data models.ChatMessage `json:"data"`
	}
	if err := json.Unmarshal(<-client.Channel, &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Content != content || response.Data.Display.BlockedImages != 1 ||
		!strings.Contains(response.Data.Display.HTML, `src="`+imageURL+`"`) ||
		strings.Contains(response.Data.Display.HTML, "https://tracker.example") {
		t.Fatalf("unexpected widget message: %#v", response.Data)
	}
}
