package media

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/image"
	"github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// MaxMessageMedia bounds work and metadata accepted from a browser per message or draft.
const MaxMessageMedia = 100

func mediaPermissionError() error {
	return envelope.NewError(envelope.PermissionError, "Attachment is unavailable or belongs to another user", nil)
}

// GetPendingForUser resolves only files uploaded by this caller. IDs are never capabilities.
func (m *Manager) GetPendingForUser(ids []int, userID int) ([]models.Media, error) {
	if userID <= 0 || len(ids) > MaxMessageMedia {
		return nil, mediaPermissionError()
	}
	out := make([]models.Media, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id <= 0 {
			return nil, mediaPermissionError()
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		item, err := m.Get(id, "")
		if err != nil || item.Model.String != models.ModelMessages || item.ModelID.Int > 0 || !item.UploadedBy.Valid || item.UploadedBy.Int != userID {
			return nil, mediaPermissionError()
		}
		out = append(out, item)
	}
	return out, nil
}

// LinkMessageMediaTx validates and locks every requested file before claiming it.
// Browser uploads require their uploader; userID zero is reserved for server ingestion
// and can claim only explicitly supplied media IDs with no browser uploader.
// Quoted inline files may remain linked to older messages in the same conversation.
func (m *Manager) LinkMessageMediaTx(tx *sqlx.Tx, messageID int, media []models.Media, inlineUUIDs []string, userID int) error {
	if len(media) == 0 && len(inlineUUIDs) == 0 {
		return nil
	}
	if userID > 0 && len(media)+len(inlineUUIDs) > MaxMessageMedia {
		return mediaPermissionError()
	}
	var conversationID int
	if err := tx.Get(&conversationID, `SELECT conversation_id FROM conversation_messages WHERE id=$1`, messageID); err != nil {
		return err
	}
	ids := make([]int, 0, len(media))
	for _, item := range media {
		ids = append(ids, item.ID)
	}
	var candidates []models.Media
	if err := tx.Select(&candidates, `SELECT * FROM media WHERE id=ANY($1::int[]) OR uuid=ANY($2::uuid[]) ORDER BY id FOR UPDATE`, pq.Array(ids), pq.Array(inlineUUIDs)); err != nil {
		return err
	}
	foundIDs, foundUUIDs := map[int]bool{}, map[string]bool{}
	claimIDs, claimInline := []int{}, []string{}
	for _, item := range candidates {
		explicit := slices.Contains(ids, item.ID)
		inline := slices.Contains(inlineUUIDs, item.UUID)
		foundIDs[item.ID], foundUUIDs[item.UUID] = true, true
		if item.Model.String != models.ModelMessages {
			return mediaPermissionError()
		}
		if item.ModelID.Int > 0 {
			var sourceConversation int
			err := tx.Get(&sourceConversation, `SELECT conversation_id FROM conversation_messages WHERE id=$1`, item.ModelID.Int)
			if err != nil || explicit || sourceConversation != conversationID {
				return mediaPermissionError()
			}
			continue
		}
		if userID > 0 {
			if !item.UploadedBy.Valid || item.UploadedBy.Int != userID {
				return mediaPermissionError()
			}
		} else if item.UploadedBy.Valid || !explicit {
			return mediaPermissionError()
		}
		claimIDs = append(claimIDs, item.ID)
		if inline {
			claimInline = append(claimInline, item.UUID)
		}
	}
	for _, id := range ids {
		if !foundIDs[id] {
			return mediaPermissionError()
		}
	}
	for _, uuid := range inlineUUIDs {
		if !foundUUIDs[uuid] {
			return mediaPermissionError()
		}
	}
	if len(claimIDs) == 0 {
		return nil
	}
	res, err := tx.Stmtx(m.queries.LinkMessageMedia).Exec(messageID, pq.Array(claimIDs), pq.Array(claimInline))
	if err != nil {
		return fmt.Errorf("linking media to message %d: %w", messageID, err)
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count != int64(len(claimIDs)) {
		return mediaPermissionError()
	}
	return nil
}

// deletePendingMessageMedia keeps the row lock through blob deletion, so a draft
// save, review submission or send either establishes its reference first or sees
// the file as gone.
func (m *Manager) deletePendingMessageMedia(id int) error {
	tx, err := m.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var item models.Media
	err = tx.Get(&item, `SELECT * FROM media WHERE id=$1 FOR UPDATE`, id)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var eligible bool
	err = tx.Get(&eligible, `SELECT
  (model_type='messages' OR model_type IS NULL)
  AND NOT EXISTS (SELECT 1 FROM conversation_draft_media dm WHERE dm.media_id=media.id)
  AND NOT EXISTS (SELECT 1 FROM outbound_review_media rm WHERE rm.media_id=media.id)
  AND ((COALESCE(model_id,0)=0 AND created_at < NOW()-INTERVAL '7 days')
    OR (model_id>0 AND created_at < NOW()-INTERVAL '24 hours' AND NOT EXISTS (SELECT 1 FROM conversation_messages cm WHERE cm.id=media.model_id)))
  FROM media WHERE id=$1`, id)
	if err != nil || !eligible {
		return err
	}
	if err := m.deleteBlob(item.UUID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM media WHERE id=$1`, id); err != nil {
		return err
	}
	if strings.HasPrefix(item.ContentType, "image/") {
		_ = m.deleteBlob(image.ThumbPrefix + item.UUID)
	}
	return tx.Commit()
}

func (m *Manager) deleteBlob(name string) error {
	err := m.store.Delete(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
