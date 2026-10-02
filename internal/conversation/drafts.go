package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	mediamanager "github.com/jakedolan443/fernmail/internal/media"
	mmodels "github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func (m *Manager) UpsertConversationDraft(conversationID, userID int, draftType, content string, meta json.RawMessage) (models.ConversationDraft, error) {
	var draft models.ConversationDraft
	if len(meta) == 0 || string(meta) == "null" {
		meta = json.RawMessage(`{}`)
	}
	if len(content) > 1024*1024 || len(meta) > 32*1024 {
		return draft, envelope.NewError(envelope.InputError, "Draft exceeds the allowed size", nil)
	}
	content, _ = m.normalizeInlineUploads(content, conversationID, userID)
	ids, inline, err := draftMediaReferences(content, meta)
	if err != nil {
		return draft, m.draftError(err)
	}
	tx, err := m.db.Beginx()
	if err != nil {
		return draft, m.draftError(err)
	}
	defer tx.Rollback()
	if err := tx.Stmtx(m.q.UpsertConversationDraft).Get(&draft, conversationID, userID, draftType, content, meta); err != nil {
		return draft, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	files, err := lockDraftMedia(tx, conversationID, userID, ids, inline)
	if err != nil {
		return draft, m.draftError(err)
	}
	if _, err := tx.Exec(`DELETE FROM conversation_draft_media WHERE draft_id=$1`, draft.ID); err != nil {
		return draft, m.draftError(err)
	}
	for _, file := range files {
		if _, err := tx.Exec(`INSERT INTO conversation_draft_media(draft_id,media_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, draft.ID, file.ID); err != nil {
			return draft, m.draftError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return draft, m.draftError(err)
	}
	m.prepareDraft(&draft)
	return draft, nil
}

func (m *Manager) GetAllUserDrafts(userID int) ([]models.ConversationDraft, error) {
	var drafts = make([]models.ConversationDraft, 0)
	if err := m.q.GetAllUserDrafts.Select(&drafts, userID); err != nil {
		m.lo.Error("error fetching user drafts", "user_id", userID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for i := range drafts {
		m.prepareDraft(&drafts[i])
	}
	return drafts, nil
}

// DeleteConversationDraft deletes a draft for a conversation by ID or UUID. An empty draftType deletes all types.
func (m *Manager) DeleteConversationDraft(conversationID int, uuid string, userID int, draftType string) error {
	var uuidParam any
	if uuid != "" {
		uuidParam = uuid
	}

	if _, err := m.q.DeleteConversationDraft.Exec(conversationID, uuidParam, userID, draftType); err != nil {
		m.lo.Error("error deleting conversation draft", "conversation_id", conversationID, "uuid", uuid, "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	return nil
}

// DeleteStaleDrafts deletes drafts older than the specified retention period.
func (m *Manager) DeleteStaleDrafts(ctx context.Context, retentionPeriod time.Duration) error {
	cutoff := time.Now().Add(-retentionPeriod)
	res, err := m.q.DeleteStaleDrafts.ExecContext(ctx, cutoff)
	if err != nil {
		m.lo.Error("error deleting stale drafts", "error", err)
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected > 0 {
		m.lo.Info("deleted stale drafts", "count", rowsAffected)
	}

	return nil
}

// resolveDraftInlineCIDs rewrites inline cid: refs to media URLs, resolving only unattached media or media linked to the draft's own conversation.
func (m *Manager) resolveDraftInlineCIDs(conversationID, userID int, content string) string {
	cids := extractInlineContentIDs(content)
	for _, cid := range cids {
		if !strings.HasPrefix(cid, "ldsk-") {
			continue
		}
		parsed, err := uuid.Parse(strings.TrimPrefix(cid, "ldsk-"))
		if err != nil {
			continue
		}
		media, err := m.mediaStore.GetDraftInlineMedia(parsed.String(), conversationID, userID)
		if err != nil {
			continue
		}
		content = strings.ReplaceAll(content, "cid:"+cid, m.mediaStore.GetURL(media.UUID, media.ContentType, media.Filename))
	}
	return content
}

// draftMediaReferences trusts only IDs, never filenames, sizes, URLs, or ownership
// supplied in browser metadata. The authoritative rows are checked while locked.
func draftMediaReferences(content string, meta json.RawMessage) ([]int, []string, error) {
	var parsed struct {
		Attachments []struct {
			ID int `json:"id"`
		} `json:"attachments"`
	}
	if len(meta) > 0 && string(meta) != "null" {
		if err := json.Unmarshal(meta, &parsed); err != nil {
			return nil, nil, envelope.NewError(envelope.InputError, "Invalid draft attachments", nil)
		}
	}
	ids, inline := []int{}, []string{}
	for _, item := range parsed.Attachments {
		if item.ID <= 0 {
			return nil, nil, envelope.NewError(envelope.InputError, "Invalid draft attachment", nil)
		}
		ids = append(ids, item.ID)
	}
	for _, cid := range extractInlineContentIDs(content) {
		raw := strings.TrimPrefix(cid, "ldsk-")
		if cid == raw {
			continue
		}
		parsed, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		inline = append(inline, parsed.String())
	}
	if len(ids)+len(inline) > mediamanager.MaxMessageMedia {
		return nil, nil, envelope.NewError(envelope.InputError, "Too many draft attachments", nil)
	}
	return ids, inline, nil
}

func lockDraftMedia(tx *sqlx.Tx, conversationID, userID int, ids []int, inline []string) ([]mmodels.Media, error) {
	files := []mmodels.Media{}
	if len(ids)+len(inline) == 0 {
		return files, nil
	}
	if err := tx.Select(&files, `SELECT * FROM media WHERE id=ANY($1::int[]) OR uuid=ANY($2::uuid[]) ORDER BY id FOR UPDATE`, pq.Array(ids), pq.Array(inline)); err != nil {
		return nil, err
	}
	foundIDs, foundUUIDs := map[int]bool{}, map[string]bool{}
	denied := func() error {
		return envelope.NewError(envelope.PermissionError, "Draft attachment is unavailable or belongs to another user", nil)
	}
	for _, file := range files {
		foundIDs[file.ID], foundUUIDs[file.UUID] = true, true
		if file.Model.String != mmodels.ModelMessages {
			return nil, denied()
		}
		if file.ModelID.Int == 0 {
			if !file.UploadedBy.Valid || file.UploadedBy.Int != userID {
				return nil, denied()
			}
		} else {
			var belongs bool
			if err := tx.Get(&belongs, `SELECT EXISTS(SELECT 1 FROM conversation_messages WHERE id=$1 AND conversation_id=$2)`, file.ModelID.Int, conversationID); err != nil {
				return nil, err
			}
			if !belongs {
				return nil, denied()
			}
		}
	}
	for _, id := range ids {
		if !foundIDs[id] {
			return nil, denied()
		}
	}
	for _, id := range inline {
		if !foundUUIDs[id] {
			return nil, denied()
		}
	}
	return files, nil
}

func (m *Manager) prepareDraft(draft *models.ConversationDraft) {
	draft.Content = m.resolveDraftInlineCIDs(int(draft.ConversationID), int(draft.UserID), draft.Content)
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(draft.Meta, &metadata); err != nil || metadata == nil {
		return
	}
	var entries []struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(metadata["attachments"], &entries); err != nil {
		return
	}
	files := []mmodels.Media{}
	for _, entry := range entries {
		var file mmodels.Media
		err := m.db.Get(&file, `SELECT m.* FROM media m LEFT JOIN conversation_messages cm ON cm.id=m.model_id AND m.model_type='messages'
   WHERE m.id=$1 AND m.model_type='messages' AND ((COALESCE(m.model_id,0)=0 AND m.uploaded_by=$2) OR cm.conversation_id=$3)`, entry.ID, draft.UserID, draft.ConversationID)
		if err != nil {
			continue
		}
		file.URL = m.mediaStore.GetURL(file.UUID, file.ContentType, file.Filename)
		files = append(files, file)
	}
	metadata["attachments"], _ = json.Marshal(files)
	draft.Meta, _ = json.Marshal(metadata)
}

func (m *Manager) draftError(err error) error {
	var env envelope.Error
	if errors.As(err, &env) {
		return env
	}
	m.lo.Error("error saving conversation draft", "error", err)
	return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
}
