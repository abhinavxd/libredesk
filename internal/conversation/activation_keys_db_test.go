package conversation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jakedolan443/fernmail/internal/activationkey"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	wmodels "github.com/jakedolan443/fernmail/internal/webhook/models"
	"github.com/zerodha/logf"
)

type keyFixture struct {
	reviewFixture
	keys  *activationkey.Manager
	appID int
}

func newKeyFixture(t *testing.T, name string) keyFixture {
	t.Helper()
	f := newReviewFixture(t, name)
	lo := logf.New(logf.Opts{})
	keys, err := activationkey.New(activationkey.Opts{DB: f.db, Lo: &lo, EncryptionKey: strings.Repeat("k", 32)})
	if err != nil {
		t.Fatal(err)
	}
	f.m.SetActivationKeys(keys)
	if _, err := keys.UpdateSettings(activationkey.Settings{Enabled: true, MaxKeysPerEmail: 2, LowStockThreshold: 1}); err != nil {
		t.Fatal(err)
	}
	app, err := keys.CreateApp("Star Game")
	if err != nil {
		t.Fatal(err)
	}
	return keyFixture{reviewFixture: f, keys: keys, appID: app.ID}
}

func keyChip(id string, appID int) string {
	return fmt.Sprintf(`<span data-type="activation-key" class="ld-activation-key" data-id="%s" data-app-id="%d">&lt;KEY_ID_%s&gt;</span>`, id, appID, id)
}

func (f keyFixture) send(content string) (models.Message, error) {
	return f.m.QueueReply(nil, f.conversation.InboxID, f.reviewer.ID, f.conversation.ContactID, f.conversation.UUID,
		content, []string{f.sender}, nil, nil, map[string]any{})
}

func TestContributorKeyIsTakenOnlyOnApprovalAndNeverReturns(t *testing.T) {
	f := newKeyFixture(t, "keys_review")
	submitted, err := f.reply(t, `<p>Your key: `+keyChip("100000001", f.appID)+`</p>`)
	if err != nil {
		t.Fatal(err)
	}
	if s := submitted.ActivationKeys; s == nil || s.Count != 1 || s.AppName != "Star Game" {
		t.Fatalf("submission key summary = %+v", s)
	}
	if queue, _ := f.m.ListReviews(f.reviewer, true); len(queue) != 1 || queue[0].ActivationKeys == nil || queue[0].ActivationKeys.Count != 1 {
		t.Fatalf("reviewer queue doesn't report the key: %+v", queue)
	}

	// An empty pool refuses the approval and leaves the submission waiting.
	if _, _, err := f.m.ApproveReview(submitted.UUID, f.reviewer); err == nil || !strings.Contains(err.Error(), "No keys available for Star Game") {
		t.Fatalf("approval with an empty pool: %v", err)
	}
	if review, _ := f.m.GetReview(submitted.UUID); review.Status != models.ReviewStatusPending {
		t.Fatalf("review status after failed approval = %s", review.Status)
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM conversation_messages WHERE type='outgoing'`); n != 0 {
		t.Fatalf("failed approval queued %d messages", n)
	}

	if _, err := f.keys.Import(f.appID, "ABCDE-FGHIJ-KLMNO\nZZZZZ-YYYYY-XXXXX", 0); err != nil {
		t.Fatal(err)
	}
	_, message, err := f.m.ApproveReview(submitted.UUID, f.reviewer)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := f.m.GetMessage(message.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored.Content+stored.TextContent, "ABCDE") || !strings.Contains(stored.Content, `data-id="100000001"`) {
		t.Fatalf("stored message must hold only the placeholder: %q", stored.Content)
	}
	var bound struct {
		ID         int64  `db:"id"`
		Recipients string `db:"recipients"`
		SentBy     int    `db:"sent_by"`
		ApprovedBy int    `db:"approved_by"`
	}
	if err := f.db.Get(&bound, `SELECT id, array_to_string(recipients, ',') AS recipients, sent_by, approved_by FROM activation_keys
		WHERE status='activated' AND message_id=$1 AND placeholder_id='100000001'`, stored.ID); err != nil {
		t.Fatalf("key not bound to the message: %v", err)
	}
	if bound.Recipients != f.sender || bound.SentBy != f.contributor.ID || bound.ApprovedBy != f.reviewer.ID {
		t.Fatalf("activation record = %+v", bound)
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM activation_keys WHERE status='redeemable'`); n != 1 {
		t.Fatalf("redeemable keys left = %d", n)
	}

	// The worker writes the key into the outgoing copy only.
	keys, err := f.m.prepareActivationKeys(&stored)
	if err != nil {
		t.Fatal(err)
	}
	if err := keys.apply(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stored.Content, "<strong>ABCDE-FGHIJ-KLMNO</strong>") || strings.Contains(stored.Content, "activation-key") {
		t.Fatalf("outgoing content = %q", stored.Content)
	}

	// Activated is final, whatever path tries to undo it.
	if _, err := f.db.Exec(`UPDATE activation_keys SET status='redeemable' WHERE id=$1`, bound.ID); err == nil {
		t.Fatal("an activated key went back to redeemable")
	}
	if _, err := f.db.Exec(`DELETE FROM activation_keys WHERE id=$1`, bound.ID); err == nil {
		t.Fatal("an activated key was deleted")
	}
	if err := f.keys.Void(bound.ID, f.reviewer.ID); err == nil {
		t.Fatal("an activated key was voided")
	}

	// A customer's reply quoting the key is hidden from previews, webhooks
	// and viewers who may not see keys.
	hooks := &recordingWebhooks{}
	f.m.webhookStore = hooks
	quote := "Thanks! It says ABCDE-FGHIJ-KLMNO is already used?"
	reply, err := f.m.CreateContactMessage(nil, f.conversation.ContactID, f.conversation.UUID, quote, models.ContentTypeText, false, "")
	if err != nil {
		t.Fatal(err)
	}
	var preview string
	if err := f.db.Get(&preview, `SELECT last_message FROM conversations WHERE id=$1`, f.conversation.ID); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(preview, "ABCDE") || !strings.Contains(preview, activationkey.MaskedKey) {
		t.Fatalf("conversation preview = %q", preview)
	}
	if len(hooks.messages) != 1 || strings.Contains(hooks.messages[0].Content+hooks.messages[0].TextContent, "ABCDE") ||
		!strings.Contains(hooks.messages[0].Content, activationkey.MaskedKey) {
		t.Fatalf("message.created webhook = %+v", hooks.messages)
	}
	if !strings.Contains(reply.Content, "ABCDE-FGHIJ-KLMNO") {
		t.Fatalf("the stored reply lost the quoted key: %q", reply.Content)
	}

	// The same key quoted in another conversation, e.g. a forward, is hidden too.
	forwardUUID, forwardContact, err := f.m.StartConversation(f.addressID, f.conversation.InboxID, "Fwd: key", "friend@example.test")
	if err != nil {
		t.Fatal(err)
	}
	forward, err := f.m.CreateContactMessage(nil, forwardContact, forwardUUID, "<p>Fwd: your key is abcde-fghij-klmno</p>", models.ContentTypeHTML, false, "")
	if err != nil {
		t.Fatal(err)
	}
	messages := []models.Message{reply, forward}
	if err := f.m.MaskActivationKeys(messages); err != nil {
		t.Fatal(err)
	}
	for _, message := range messages {
		if strings.Contains(strings.ToUpper(message.Content+message.TextContent), "ABCDE") {
			t.Fatalf("masked message = %q / %q", message.Content, message.TextContent)
		}
	}

	// A key still in the pool isn't sent, so it isn't hidden.
	unsent := []models.Message{{Content: "ZZZZZ-YYYYY-XXXXX"}}
	if err := f.m.MaskActivationKeys(unsent); err != nil || unsent[0].Content != "ZZZZZ-YYYYY-XXXXX" {
		t.Fatalf("unsent key = %q, %v", unsent[0].Content, err)
	}
}

// recordingWebhooks keeps the messages sent to message webhooks.
type recordingWebhooks struct {
	receiptWebhookStore
	messages []models.Message
}

func (r *recordingWebhooks) TriggerEvent(_ wmodels.WebhookEvent, data any) {
	if message, ok := data.(*models.Message); ok {
		r.messages = append(r.messages, *message)
	}
}

func TestKeyPlaceholderRulesOnDirectSend(t *testing.T) {
	f := newKeyFixture(t, "keys_direct")
	if _, err := f.keys.Import(f.appID, "AAAAA-11111-AAAAA\nBBBBB-22222-BBBBB\nCCCCC-33333-CCCCC", 0); err != nil {
		t.Fatal(err)
	}
	other, err := f.keys.CreateApp("Other Game")
	if err != nil {
		t.Fatal(err)
	}
	refused := map[string]string{
		"over the cap":       keyChip("100000001", f.appID) + keyChip("100000002", f.appID) + keyChip("100000003", f.appID),
		"two games":          keyChip("100000001", f.appID) + keyChip("100000002", other.ID),
		"repeated chip":      keyChip("100000001", f.appID) + keyChip("100000001", f.appID),
		"unknown game":       keyChip("100000001", 9999),
		"damaged chip":       `<span data-type="activation-key" data-id="100000001">x</span>`,
		"chip without an id": `<span data-type="activation-key" data-app-id="1">x</span>`,
	}
	for name, content := range refused {
		if _, err := f.send(`<p>` + content + `</p>`); err == nil {
			t.Errorf("%s: sent", name)
		}
	}
	if _, err := f.m.SendPrivateNote(nil, f.reviewer.ID, f.conversation.UUID, `<p>`+keyChip("100000001", f.appID)+`</p>`, nil); err == nil {
		t.Error("a private note took a key")
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM activation_keys WHERE status='activated'`); n != 0 {
		t.Fatalf("refused emails activated %d keys", n)
	}

	// Lookalike text is just text.
	plain, err := f.send(`<p>&lt;KEY_ID_100000001&gt;</p>`)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM activation_keys WHERE message_id=$1`, plain.ID); n != 0 {
		t.Fatal("typed placeholder text took a key")
	}

	two, err := f.send(`<p>` + keyChip("100000001", f.appID) + ` and ` + keyChip("100000002", f.appID) + `</p>`)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM activation_keys WHERE status='activated' AND message_id=$1`, two.ID); n != 2 {
		t.Fatalf("two chips activated %d keys", n)
	}

	// Archived games and a disabled feature refuse new keys.
	if _, err := f.keys.UpdateApp(f.appID, "Star Game", true); err != nil {
		t.Fatal(err)
	}
	if _, err := f.send(`<p>` + keyChip("100000004", f.appID) + `</p>`); err == nil {
		t.Error("an archived game handed out a key")
	}
	if _, err := f.keys.UpdateApp(f.appID, "Star Game", false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.keys.UpdateSettings(activationkey.Settings{Enabled: false, MaxKeysPerEmail: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.send(`<p>` + keyChip("100000005", f.appID) + `</p>`); err == nil || !strings.Contains(err.Error(), "turned off") {
		t.Errorf("disabled key distribution: %v", err)
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM activation_keys WHERE status='redeemable'`); n != 1 {
		t.Fatalf("redeemable keys left = %d, want 1", n)
	}
}
