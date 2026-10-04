package conversation

import (
	"strings"
	"testing"

	"github.com/jakedolan443/fernmail/internal/activationkey"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jmoiron/sqlx"
)

type fakeKeyStore struct {
	keys map[string]string
}

func (f fakeKeyStore) Validate([]activationkey.Placeholder) error          { return nil }
func (f fakeKeyStore) AllocateTx(*sqlx.Tx, activationkey.Allocation) error { return nil }
func (f fakeKeyStore) MessageKeys(int) (map[string]string, error)          { return f.keys, nil }
func (f fakeKeyStore) MaskSent(...*string) error                           { return nil }
func (f fakeKeyStore) AppNames([]int) (map[int]string, error)              { return nil, nil }

// The exact markup the composer's editor serializes.
const editorChip = `<span data-type="activation-key" class="ld-activation-key" data-id="510338604" data-app-id="3">&lt;KEY_ID_510338604&gt;</span>`

func TestActivationKeysAreWrittenOnlyAfterRendering(t *testing.T) {
	m := &Manager{activationKeys: fakeKeyStore{keys: map[string]string{"510338604": "ABCDE-FGHIJ-KLMNO"}}}
	message := models.Message{ID: 1, Content: `<p>Here you go: ` + editorChip + `</p><p>&lt;KEY_ID_999999999&gt;</p>`}

	keys, err := m.prepareActivationKeys(&message)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(message.Content, "ABCDE") || strings.Contains(message.Content, "activation-key") {
		t.Fatalf("key or placeholder visible before rendering: %q", message.Content)
	}
	// Stand-in for the email template wrapping the content.
	message.Content = "<html><body>" + message.Content + "</body></html>"
	if err := keys.apply(&message); err != nil {
		t.Fatal(err)
	}
	want := `<html><body><p>Here you go: <strong>ABCDE-FGHIJ-KLMNO</strong></p><p>&lt;KEY_ID_999999999&gt;</p></body></html>`
	if message.Content != want {
		t.Fatalf("got  %q\nwant %q", message.Content, want)
	}
}

func TestActivationKeyMismatchFailsDelivery(t *testing.T) {
	for name, tc := range map[string]struct {
		content string
		keys    map[string]string
	}{
		"placeholder without a key":   {`<p>` + editorChip + `</p>`, map[string]string{}},
		"key without a placeholder":   {`<p>no chip</p>`, map[string]string{"510338604": "ABCDE-FGHIJ-KLMNO"}},
		"key for another placeholder": {`<p>` + editorChip + `</p>`, map[string]string{"111111111": "ABCDE-FGHIJ-KLMNO"}},
		"damaged placeholder":         {`<p><span data-type="activation-key" data-id="510338604">x</p>`, map[string]string{}},
	} {
		m := &Manager{activationKeys: fakeKeyStore{keys: tc.keys}}
		message := models.Message{ID: 1, Content: tc.content}
		if _, err := m.prepareActivationKeys(&message); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestActivationKeyMarkerMustSurviveRenderingExactlyOnce(t *testing.T) {
	m := &Manager{activationKeys: fakeKeyStore{keys: map[string]string{"510338604": "ABCDE-FGHIJ-KLMNO"}}}
	message := models.Message{ID: 1, Content: `<p>` + editorChip + `</p>`}
	keys, err := m.prepareActivationKeys(&message)
	if err != nil {
		t.Fatal(err)
	}
	duplicated := models.Message{Content: message.Content + message.Content}
	if err := keys.apply(&duplicated); err == nil {
		t.Fatal("a marker rendered twice must not send the key twice")
	}
	dropped := models.Message{Content: "<p>template lost the content</p>"}
	if err := keys.apply(&dropped); err == nil {
		t.Fatal("a marker lost in rendering must fail delivery")
	}
}

func TestMessagesWithoutKeysPassThrough(t *testing.T) {
	m := &Manager{}
	message := models.Message{ID: 1, Content: `<p>Plain reply</p>`}
	keys, err := m.prepareActivationKeys(&message)
	if err != nil || keys != nil || message.Content != `<p>Plain reply</p>` {
		t.Fatalf("got %v %v %q", keys, err, message.Content)
	}
	if err := keys.apply(&message); err != nil {
		t.Fatal(err)
	}
}

func TestPlaceholdersWithoutKeyDistributionAreRefused(t *testing.T) {
	m := &Manager{}
	message := models.Message{ID: 1, Content: `<p>` + editorChip + `</p>`}
	if _, err := m.prepareActivationKeys(&message); err == nil {
		t.Fatal("expected an error without a key store")
	}
	if err := m.ValidateActivationKeys(message.Content); err == nil {
		t.Fatal("expected validation to refuse placeholders without a key store")
	}
}
