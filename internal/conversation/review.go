package conversation

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	mmodels "github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/jakedolan443/fernmail/internal/stringutil"
	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	wsmodels "github.com/jakedolan443/fernmail/internal/ws/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

// Contributor emails wait in outbound_reviews until an Admin or Agent approves
// or denies them. Approval queues the real message in the same transaction
// that records the decision, so a submission is sent at most once and never
// silently lost; a denial returns it to its author as an editable draft.

const (
	maxReviewRecipients = 50
	maxReviewNote       = 2000
	maxReviewSubject    = 998
)

// ErrReviewDecided means another reviewer, or the author, acted first.
var ErrReviewDecided = errors.New("review already decided")

const reviewSelect = `SELECT r.id, r.uuid::text AS uuid, r.created_at, r.updated_at, r.kind, r.status, r.author_id,
	concat_ws(' ', au.first_name, NULLIF(au.last_name, '')) AS author_name,
	r.address_id, a.address, a.display_name AS address_name, a.inbox_id,
	r.conversation_id, c.uuid::text AS conversation_uuid, COALESCE(c.subject, '') AS conversation_subject,
	r.subject, r.content, r."to", r.cc, r.bcc, r.reviewer_id,
	NULLIF(concat_ws(' ', rv.first_name, NULLIF(rv.last_name, '')), '') AS reviewer_name,
	r.reviewed_at, r.decision_note, r.dismissed_at, m.uuid::text AS message_uuid
FROM outbound_reviews r
JOIN users au ON au.id = r.author_id
JOIN email_addresses a ON a.id = r.address_id
LEFT JOIN conversations c ON c.id = r.conversation_id
LEFT JOIN users rv ON rv.id = r.reviewer_id
LEFT JOIN conversation_messages m ON m.id = r.message_id`

func (m *Manager) reviewError(err error) error {
	var envErr envelope.Error
	if errors.As(err, &envErr) {
		return err
	}
	if errors.Is(err, ErrReviewDecided) {
		return envelope.NewError(envelope.ConflictError, "This email has already been reviewed or withdrawn.", nil)
	}
	m.lo.Error("review queue error", "error", err)
	return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
}

// normalizeRecipients trims, de-duplicates and validates one recipient field.
func normalizeRecipients(field string, values []string) ([]string, error) {
	out := []string{}
	for _, value := range stringutil.RemoveEmpty(values) {
		value = strings.TrimSpace(value)
		if !stringutil.ValidEmail(value) {
			return nil, envelope.NewError(envelope.InputError, fmt.Sprintf("%q in %s is not a valid email address.", value, field), nil)
		}
		if !slices.ContainsFunc(out, func(existing string) bool { return strings.EqualFold(existing, value) }) {
			out = append(out, value)
		}
	}
	if len(out) > maxReviewRecipients {
		return nil, envelope.NewError(envelope.InputError, fmt.Sprintf("%s can have at most %d recipients.", field, maxReviewRecipients), nil)
	}
	return out, nil
}

// NormalizeOutgoing validates the parts every outgoing email shares, whether
// it is sent directly or submitted for review.
func NormalizeOutgoing(in models.ReviewInput, needsSubject bool) (models.ReviewInput, error) {
	var err error
	if in.To, err = normalizeRecipients("To", in.To); err != nil {
		return in, err
	}
	if in.CC, err = normalizeRecipients("CC", in.CC); err != nil {
		return in, err
	}
	if in.BCC, err = normalizeRecipients("BCC", in.BCC); err != nil {
		return in, err
	}
	if len(in.To) == 0 {
		return in, envelope.NewError(envelope.InputError, "Add at least one recipient in To.", nil)
	}
	in.Subject = strings.TrimSpace(in.Subject)
	if needsSubject && in.Subject == "" {
		return in, envelope.NewError(envelope.InputError, "Add a subject.", nil)
	}
	if len(in.Subject) > maxReviewSubject || strings.ContainsAny(in.Subject, "\r\n") {
		return in, envelope.NewError(envelope.InputError, "The subject is too long or contains a line break.", nil)
	}
	if len(in.Content) > 1024*1024 {
		return in, envelope.NewError(envelope.InputError, "The email is too long.", nil)
	}
	hasText := strings.TrimSpace(stringutil.HTML2Text(in.Content)) != ""
	if !hasText && len(in.Media) == 0 && !strings.Contains(in.Content, "<img") {
		return in, envelope.NewError(envelope.InputError, "The email is empty.", nil)
	}
	return in, nil
}

// SubmitReview holds a Contributor's reply (ConversationID set) or new email for review.
func (m *Manager) SubmitReview(authorID int, in models.ReviewInput) (models.Review, error) {
	kind := models.ReviewKindReply
	if in.ConversationID == 0 {
		kind = models.ReviewKindNew
	}
	in, err := NormalizeOutgoing(in, kind == models.ReviewKindNew)
	if err != nil {
		return models.Review{}, err
	}
	if kind == models.ReviewKindReply {
		in.Subject = ""
	}
	content, inline := m.normalizeInlineUploads(in.Content, in.ConversationID, authorID)
	tx, err := m.db.Beginx()
	if err != nil {
		return models.Review{}, m.reviewError(err)
	}
	defer tx.Rollback()
	var conversationID null.Int
	if in.ConversationID > 0 {
		conversationID = null.IntFrom(in.ConversationID)
	}
	var id int
	var uuid string
	err = tx.QueryRow(`INSERT INTO outbound_reviews (kind, author_id, address_id, conversation_id, subject, content, "to", cc, bcc)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, uuid::text`,
		kind, authorID, in.AddressID, conversationID, in.Subject, content, pq.Array(in.To), pq.Array(in.CC), pq.Array(in.BCC)).Scan(&id, &uuid)
	if err != nil {
		if strings.Contains(err.Error(), "index_unique_outbound_reviews_pending_reply") {
			return models.Review{}, envelope.NewError(envelope.ConflictError, "You already have a reply waiting for review in this conversation. Withdraw it to make changes.", nil)
		}
		return models.Review{}, m.reviewError(err)
	}
	if err := linkReviewMedia(tx, id, in.ConversationID, authorID, in.Media, inline); err != nil {
		return models.Review{}, m.reviewError(err)
	}
	if kind == models.ReviewKindReply {
		// A fresh submission supersedes any earlier denial notice in this conversation.
		if _, err := tx.Exec(`UPDATE outbound_reviews SET dismissed_at = NOW()
			WHERE author_id = $1 AND conversation_id = $2 AND id <> $3 AND status IN ('denied', 'withdrawn') AND dismissed_at IS NULL`,
			authorID, in.ConversationID, id); err != nil {
			return models.Review{}, m.reviewError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return models.Review{}, m.reviewError(err)
	}
	review, err := m.GetReview(uuid)
	if err == nil {
		m.broadcastReview(wsmodels.MessageTypeReviewCreated, review)
	}
	return review, err
}

// linkReviewMedia protects the submission's own uploads from cleanup. Files
// quoted from earlier messages in the conversation are already linked there.
func linkReviewMedia(tx *sqlx.Tx, reviewID, conversationID, authorID int, attachments []mmodels.Media, inline []string) error {
	ids := make([]int, 0, len(attachments))
	for _, file := range attachments {
		ids = append(ids, file.ID)
	}
	files, err := lockDraftMedia(tx, conversationID, authorID, ids, inline)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM outbound_review_media WHERE review_id = $1`, reviewID); err != nil {
		return err
	}
	for _, file := range files {
		if file.ModelID.Int > 0 {
			continue
		}
		isInline := !slices.Contains(ids, file.ID)
		if _, err := tx.Exec(`INSERT INTO outbound_review_media (review_id, media_id, inline) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, reviewID, file.ID, isInline); err != nil {
			return err
		}
	}
	return nil
}

// GetReview returns one submission with its held attachments.
func (m *Manager) GetReview(uuid string) (models.Review, error) {
	var review models.Review
	err := m.db.Get(&review, reviewSelect+` WHERE r.uuid = $1::uuid`, uuid)
	if errors.Is(err, sql.ErrNoRows) || (err != nil && strings.Contains(err.Error(), "invalid input syntax for type uuid")) {
		return review, envelope.NewError(envelope.NotFoundError, "This email is no longer waiting for review.", nil)
	}
	if err != nil {
		return review, m.reviewError(err)
	}
	if err := m.decorateReviews([]*models.Review{&review}); err != nil {
		return review, m.reviewError(err)
	}
	return review, nil
}

// ListReviews returns a reviewer's queue (pending submissions in addresses
// they can read), or a Contributor's own pending and returned submissions.
func (m *Manager) ListReviews(viewer umodels.User, asReviewer bool) ([]models.Review, error) {
	reviews := []models.Review{}
	var err error
	if asReviewer {
		// Only mail that could still be sent: its author must still be able to use the address.
		err = m.db.Select(&reviews, reviewSelect+` WHERE r.status = 'pending' AND r.author_id <> $1
			AND can_access_email_address(r.address_id, $1) AND can_access_email_address(r.address_id, r.author_id)
			ORDER BY r.created_at, r.id`, viewer.ID)
	} else {
		err = m.db.Select(&reviews, reviewSelect+` WHERE r.author_id = $1
			AND (r.status = 'pending' OR (r.kind = 'new' AND r.status IN ('denied', 'withdrawn')))
			AND can_access_email_address(r.address_id, $1) ORDER BY r.updated_at DESC, r.id DESC`, viewer.ID)
	}
	if err != nil {
		return nil, m.reviewError(err)
	}
	ptrs := make([]*models.Review, len(reviews))
	for i := range reviews {
		ptrs[i] = &reviews[i]
	}
	if err := m.decorateReviews(ptrs); err != nil {
		return nil, m.reviewError(err)
	}
	return reviews, nil
}

// CountReviews drives the sidebar badge.
func (m *Manager) CountReviews(viewer umodels.User, asReviewer bool) (models.ReviewCounts, error) {
	var counts models.ReviewCounts
	var err error
	if asReviewer {
		err = m.db.Get(&counts, `SELECT count(*) AS pending, 0 AS returned FROM outbound_reviews r
			WHERE r.status = 'pending' AND r.author_id <> $1 AND can_access_email_address(r.address_id, $1)
			AND can_access_email_address(r.address_id, r.author_id)`, viewer.ID)
	} else {
		err = m.db.Get(&counts, `SELECT count(*) FILTER (WHERE r.status = 'pending') AS pending,
			count(*) FILTER (WHERE r.kind = 'new' AND r.status IN ('denied', 'withdrawn')) AS returned
			FROM outbound_reviews r WHERE r.author_id = $1 AND can_access_email_address(r.address_id, $1)`, viewer.ID)
	}
	if err != nil {
		return counts, m.reviewError(err)
	}
	return counts, nil
}

// ConversationReviews returns what a conversation thread should show: pending
// submissions for reviewers, and the viewer's own pending submission or
// undismissed denial.
func (m *Manager) ConversationReviews(conversationID int, viewer umodels.User, asReviewer bool) ([]models.Review, error) {
	reviews := []models.Review{}
	err := m.db.Select(&reviews, reviewSelect+` WHERE r.conversation_id = $1 AND (
			(r.status = 'pending' AND $3 AND can_access_email_address(r.address_id, $2))
			OR (r.author_id = $2 AND (r.status = 'pending' OR (r.status = 'denied' AND r.dismissed_at IS NULL)))
		) ORDER BY r.created_at, r.id`, conversationID, viewer.ID, asReviewer)
	if err != nil {
		return nil, m.reviewError(err)
	}
	ptrs := make([]*models.Review, len(reviews))
	for i := range reviews {
		ptrs[i] = &reviews[i]
	}
	if err := m.decorateReviews(ptrs); err != nil {
		return nil, m.reviewError(err)
	}
	return reviews, nil
}

func (m *Manager) decorateReviews(reviews []*models.Review) error {
	if len(reviews) == 0 {
		return nil
	}
	ids := make([]int, 0, len(reviews))
	byID := make(map[int]*models.Review, len(reviews))
	for _, review := range reviews {
		review.Preview = strings.TrimSpace(stringutil.HTML2Text(review.Content))
		if len(review.Preview) > 280 {
			review.Preview = strings.TrimSpace(review.Preview[:280]) + "…"
		}
		review.Attachments = []models.ReviewAttachment{}
		ids = append(ids, review.ID)
		byID[review.ID] = review
	}
	var rows []struct {
		ReviewID int `db:"review_id"`
		models.ReviewAttachment
	}
	if err := m.db.Select(&rows, `SELECT rm.review_id, rm.inline, md.id, md.uuid::text AS uuid, md.filename, md.content_type, md.size
		FROM outbound_review_media rm JOIN media md ON md.id = rm.media_id
		WHERE rm.review_id = ANY($1::bigint[]) ORDER BY md.id`, pq.Array(ids)); err != nil {
		return err
	}
	for _, row := range rows {
		attachment := row.ReviewAttachment
		attachment.URL = m.mediaStore.GetURL(attachment.UUID, attachment.ContentType, attachment.Filename)
		byID[row.ReviewID].Attachments = append(byID[row.ReviewID].Attachments, attachment)
	}
	return nil
}

func (m *Manager) reviewAttachments(reviewID int) ([]mmodels.Media, error) {
	files := []mmodels.Media{}
	err := m.db.Select(&files, `SELECT md.* FROM media md JOIN outbound_review_media rm ON rm.media_id = md.id
		WHERE rm.review_id = $1 AND NOT rm.inline ORDER BY md.id`, reviewID)
	return files, err
}

// ApproveReview queues the submission exactly as if its author had sent it.
// A new email gets its conversation now; the review row records the result
// in the same transaction as the message, so only one approval can win.
func (m *Manager) ApproveReview(uuid string, reviewer umodels.User) (models.Review, models.Message, error) {
	review, err := m.GetReview(uuid)
	if err != nil {
		return review, models.Message{}, err
	}
	if review.Status != models.ReviewStatusPending {
		return review, models.Message{}, m.reviewError(ErrReviewDecided)
	}
	// Mail is sent as its author; delivery would refuse it if they lost access.
	var authorAllowed bool
	if err := m.db.Get(&authorAllowed, `SELECT can_access_email_address($1, $2)`, review.AddressID, review.AuthorID); err != nil {
		return review, models.Message{}, m.reviewError(err)
	}
	if !authorAllowed {
		return review, models.Message{}, envelope.NewError(envelope.InputError, "The author no longer has access to this address, so this email can't be sent.", nil)
	}
	attachments, err := m.reviewAttachments(review.ID)
	if err != nil {
		return review, models.Message{}, m.reviewError(err)
	}
	meta := map[string]any{"review_uuid": review.UUID, "approved_by": reviewer.ID, "approved_by_name": strings.TrimSpace(reviewer.FullName())}
	record := func(tx *sqlx.Tx, message *models.Message) error {
		res, err := tx.Exec(`UPDATE outbound_reviews SET status = 'approved', reviewer_id = $2, reviewed_at = NOW(), updated_at = NOW(),
			message_id = $3, conversation_id = COALESCE(conversation_id, $4) WHERE id = $1 AND status = 'pending'`,
			review.ID, reviewer.ID, message.ID, message.ConversationID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return ErrReviewDecided
		}
		_, err = tx.Exec(`DELETE FROM outbound_review_media WHERE review_id = $1`, review.ID)
		return err
	}

	var message models.Message
	switch review.Kind {
	case models.ReviewKindReply:
		conversation, err := m.GetConversation(int(review.ConversationID.Int), "", "")
		if err != nil {
			return review, message, err
		}
		message, err = m.queueReply(attachments, conversation.InboxID, review.AuthorID, conversation.ContactID, conversation.UUID,
			review.Content, review.To, review.CC, review.BCC, meta, record)
		if err != nil {
			return review, message, m.reviewError(err)
		}
	case models.ReviewKindNew:
		conversationUUID, contactID, err := m.StartConversation(review.AddressID, review.InboxID, review.Subject, review.To[0])
		if err != nil {
			return review, message, err
		}
		message, err = m.queueReply(attachments, review.InboxID, review.AuthorID, contactID, conversationUUID,
			review.Content, review.To, review.CC, review.BCC, meta, record)
		if err != nil {
			// The provisional conversation never held a committed message.
			if delErr := m.DeleteConversation(conversationUUID); delErr != nil {
				m.lo.Error("error removing conversation after failed approval", "uuid", conversationUUID, "error", delErr)
			}
			return review, message, m.reviewError(err)
		}
	default:
		return review, message, m.reviewError(fmt.Errorf("unknown review kind %q", review.Kind))
	}
	decided, err := m.GetReview(uuid)
	if err != nil {
		return review, message, err
	}
	m.broadcastReview(wsmodels.MessageTypeReviewUpdated, decided)
	return decided, message, nil
}

// StartConversation creates an empty outgoing conversation on an address for
// a new email and returns its UUID and the recipient contact. The caller
// queues the first message, and deletes the conversation if that fails.
func (m *Manager) StartConversation(addressID, inboxID int, subject, firstRecipient string) (string, int, error) {
	contact := umodels.User{Email: null.StringFrom(firstRecipient), FirstName: recipientName(firstRecipient)}
	if err := m.userStore.ResolveEmailSender(&contact); err != nil {
		m.lo.Error("error resolving recipient contact", "error", err)
		return "", 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	_, uuid, err := m.CreateConversation(contact.ID, inboxID, addressID, "", time.Now(), subject, false, map[string]any{}, nil, 0, 0)
	if err != nil {
		m.lo.Error("error creating outgoing conversation", "error", err)
		return "", 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return uuid, contact.ID, nil
}

// recipientName derives a display name from an address's local part.
func recipientName(email string) string {
	local, _, _ := strings.Cut(email, "@")
	if local == "" {
		return email
	}
	return local
}

// ReturnReview denies (reviewer set) or withdraws (reviewer nil) a pending
// submission. A reply goes back into its author's composer as a draft with its
// recipients and attachments; a new email stays editable from the Review list.
func (m *Manager) ReturnReview(uuid string, reviewer *umodels.User, note string) (models.Review, error) {
	note = strings.TrimSpace(note)
	if len(note) > maxReviewNote {
		return models.Review{}, envelope.NewError(envelope.InputError, fmt.Sprintf("The reason can be at most %d characters.", maxReviewNote), nil)
	}
	status := models.ReviewStatusWithdrawn
	var reviewerID null.Int
	if reviewer != nil {
		status, reviewerID = models.ReviewStatusDenied, null.IntFrom(reviewer.ID)
	}
	tx, err := m.db.Beginx()
	if err != nil {
		return models.Review{}, m.reviewError(err)
	}
	defer tx.Rollback()
	var row struct {
		ID             int            `db:"id"`
		Kind           string         `db:"kind"`
		AuthorID       int            `db:"author_id"`
		ConversationID null.Int       `db:"conversation_id"`
		Content        string         `db:"content"`
		To             pq.StringArray `db:"to"`
		CC             pq.StringArray `db:"cc"`
		BCC            pq.StringArray `db:"bcc"`
	}
	err = tx.Get(&row, `UPDATE outbound_reviews SET status = $2, reviewer_id = $3, reviewed_at = NOW(), decision_note = $4, updated_at = NOW()
		WHERE uuid = $1::uuid AND status = 'pending'
		RETURNING id, kind, author_id, conversation_id, content, "to", cc, bcc`, uuid, status, reviewerID, note)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Review{}, m.reviewError(ErrReviewDecided)
	}
	if err != nil {
		return models.Review{}, m.reviewError(err)
	}
	if row.Kind == models.ReviewKindReply {
		if err := m.restoreReviewDraft(tx, row.ID, int(row.ConversationID.Int), row.AuthorID, row.Content, row.To, row.CC, row.BCC); err != nil {
			return models.Review{}, m.reviewError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return models.Review{}, m.reviewError(err)
	}
	review, err := m.GetReview(uuid)
	if err == nil {
		m.broadcastReview(wsmodels.MessageTypeReviewUpdated, review)
	}
	return review, err
}

// restoreReviewDraft moves a returned reply into its author's reply draft. The
// composer is locked while a reply is pending, so the draft is normally empty;
// anything already there is kept above the returned text.
func (m *Manager) restoreReviewDraft(tx *sqlx.Tx, reviewID, conversationID, authorID int, content string, to, cc, bcc []string) error {
	var files []struct {
		mmodels.Media
		Inline bool `db:"inline"`
	}
	if err := tx.Select(&files, `SELECT md.*, rm.inline FROM media md JOIN outbound_review_media rm ON rm.media_id = md.id
		WHERE rm.review_id = $1 ORDER BY md.id`, reviewID); err != nil {
		return err
	}
	var existing struct {
		Content string          `db:"content"`
		Meta    json.RawMessage `db:"meta"`
	}
	err := tx.Get(&existing, `SELECT content, meta FROM conversation_drafts WHERE conversation_id = $1 AND user_id = $2 AND type = 'reply' FOR UPDATE`, conversationID, authorID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	meta := map[string]any{}
	if len(existing.Meta) > 0 {
		_ = json.Unmarshal(existing.Meta, &meta)
	}
	attachments, _ := meta["attachments"].([]any)
	for _, file := range files {
		if file.Inline {
			continue
		}
		attachments = append(attachments, map[string]any{
			"id": file.ID, "uuid": file.UUID, "filename": file.Filename, "content_type": file.ContentType, "size": file.Size,
			"disposition": "attachment", "url": m.mediaStore.GetURL(file.UUID, file.ContentType, file.Filename),
		})
	}
	if len(attachments) > 0 {
		meta["attachments"] = attachments
	}
	meta["recipients"] = map[string]string{"to": strings.Join(to, ", "), "cc": strings.Join(cc, ", "), "bcc": strings.Join(bcc, ", ")}
	if strings.TrimSpace(stringutil.HTML2Text(existing.Content)) != "" || strings.Contains(existing.Content, "<img") {
		content = existing.Content + "<p></p>" + content
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	var draftID int
	if err := tx.Get(&draftID, `INSERT INTO conversation_drafts (conversation_id, user_id, type, content, meta, updated_at)
		VALUES ($1, $2, 'reply', $3, $4, NOW())
		ON CONFLICT (conversation_id, user_id, type) DO UPDATE SET content = EXCLUDED.content, meta = EXCLUDED.meta, updated_at = NOW()
		RETURNING id`, conversationID, authorID, content, metaJSON); err != nil {
		return err
	}
	for _, file := range files {
		if _, err := tx.Exec(`INSERT INTO conversation_draft_media (draft_id, media_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, draftID, file.ID); err != nil {
			return err
		}
	}
	_, err = tx.Exec(`DELETE FROM outbound_review_media WHERE review_id = $1`, reviewID)
	return err
}

// ResubmitReview sends an edited, returned new email back to the queue.
func (m *Manager) ResubmitReview(uuid string, authorID int, in models.ReviewInput) (models.Review, error) {
	in, err := NormalizeOutgoing(in, true)
	if err != nil {
		return models.Review{}, err
	}
	content, inline := m.normalizeInlineUploads(in.Content, 0, authorID)
	tx, err := m.db.Beginx()
	if err != nil {
		return models.Review{}, m.reviewError(err)
	}
	defer tx.Rollback()
	var id int
	err = tx.Get(&id, `UPDATE outbound_reviews SET status = 'pending', address_id = $3, subject = $4, content = $5, "to" = $6, cc = $7, bcc = $8,
		reviewer_id = NULL, reviewed_at = NULL, decision_note = '', dismissed_at = NULL, updated_at = NOW(), created_at = NOW()
		WHERE uuid = $1::uuid AND author_id = $2 AND kind = 'new' AND status IN ('denied', 'withdrawn') RETURNING id`,
		uuid, authorID, in.AddressID, in.Subject, content, pq.Array(in.To), pq.Array(in.CC), pq.Array(in.BCC))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Review{}, m.reviewError(ErrReviewDecided)
	}
	if err != nil {
		return models.Review{}, m.reviewError(err)
	}
	if err := linkReviewMedia(tx, id, 0, authorID, in.Media, inline); err != nil {
		return models.Review{}, m.reviewError(err)
	}
	if err := tx.Commit(); err != nil {
		return models.Review{}, m.reviewError(err)
	}
	review, err := m.GetReview(uuid)
	if err == nil {
		m.broadcastReview(wsmodels.MessageTypeReviewCreated, review)
	}
	return review, err
}

// DiscardReview deletes a returned new email its author no longer wants.
func (m *Manager) DiscardReview(uuid string, authorID int) error {
	review, err := m.GetReview(uuid)
	if err != nil {
		return err
	}
	res, err := m.db.Exec(`DELETE FROM outbound_reviews WHERE uuid = $1::uuid AND author_id = $2 AND kind = 'new' AND status IN ('denied', 'withdrawn')`, uuid, authorID)
	if err != nil {
		return m.reviewError(err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return m.reviewError(ErrReviewDecided)
	}
	review.Status = "discarded"
	m.broadcastReview(wsmodels.MessageTypeReviewUpdated, review)
	return nil
}

// DismissReview hides a denial notice from its author's composer.
func (m *Manager) DismissReview(uuid string, authorID int) error {
	if _, err := m.db.Exec(`UPDATE outbound_reviews SET dismissed_at = NOW() WHERE uuid = $1::uuid AND author_id = $2 AND status IN ('denied', 'withdrawn')`, uuid, authorID); err != nil {
		return m.reviewError(err)
	}
	return nil
}

// broadcastReview tells the author and every connected reviewer who can read
// the address that the queue changed, so badges, lists and threads refresh.
func (m *Manager) broadcastReview(eventType string, review models.Review) {
	if m.wsHub == nil {
		return
	}
	recipients := []int{review.AuthorID}
	for _, id := range m.wsHub.ConnectedUserIDs() {
		if id == review.AuthorID {
			continue
		}
		agent, err := m.userStore.GetAgentCachedOrLoad(id)
		if err != nil || !agent.Enabled || !slices.Contains(agent.Permissions, "reviews:manage") {
			continue
		}
		var allowed bool
		if err := m.db.Get(&allowed, `SELECT can_access_email_address($1, $2)`, review.AddressID, id); err != nil || !allowed {
			continue
		}
		recipients = append(recipients, id)
	}
	subject := review.Subject
	if subject == "" {
		subject = review.ConversationSubject
	}
	m.broadcastToUsers(recipients, wsmodels.Message{
		Type: eventType,
		Data: map[string]any{
			"uuid":              review.UUID,
			"kind":              review.Kind,
			"status":            review.Status,
			"address_id":        review.AddressID,
			"conversation_uuid": review.ConversationUUID.String,
			"author_id":         review.AuthorID,
			"author_name":       review.AuthorName,
			"reviewer_name":     review.ReviewerName.String,
			"decision_note":     review.DecisionNote,
			"subject":           subject,
			"message_uuid":      review.MessageUUID.String,
		},
	})
}
