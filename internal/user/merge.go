package user

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type ContactMergePhone struct {
	Number      string `json:"number"`
	CountryCode string `json:"country_code"`
}

type ContactMergeField struct {
	Key      string `json:"key"`
	Keep     any    `json:"keep"`
	Remove   any    `json:"remove"`
	Conflict bool   `json:"conflict"`
}

type ContactMergeProfile struct {
	ID             int                      `json:"id"`
	FirstName      string                   `json:"first_name"`
	LastName       string                   `json:"last_name"`
	Email          string                   `json:"email"`
	ExternalUserID string                   `json:"external_user_id"`
	Type           string                   `json:"type"`
	Enabled        bool                     `json:"enabled"`
	Conversations  int                      `json:"conversations"`
	Notes          int                      `json:"notes"`
	Identities     models.ChannelIdentities `json:"channel_identities"`
}

type ContactMergePreview struct {
	ConversationUUIDs []string            `json:"-"`
	Keep              ContactMergeProfile `json:"keep"`
	Remove            ContactMergeProfile `json:"remove"`
	Fields            []ContactMergeField `json:"fields"`
	Version           string              `json:"version"`
}

type ContactMergeRequest struct {
	RemoveID int               `json:"remove_id"`
	Version  string            `json:"version"`
	Choices  map[string]string `json:"choices"`
}

type mergeContact struct {
	models.User
	Conversations int `db:"conversations"`
	Notes         int `db:"notes"`
}

type contactMergeReference struct {
	ID           int           `db:"id"`
	MergedIntoID sql.NullInt64 `db:"merged_into_id"`
}

func (u *Manager) PreviewContactMerge(keepID, removeID int) (ContactMergePreview, error) {
	tx, err := u.db.Beginx()
	if err != nil {
		return ContactMergePreview{}, u.mergeError(err)
	}
	defer tx.Rollback()
	preview, _, err := u.contactMergePreview(tx, keepID, removeID)
	return preview, err
}

// MergeContacts rejects changed previews and transfers the complete history in one transaction.
func (u *Manager) MergeContacts(keepID int, req ContactMergeRequest, actorID int, actorEmail, ip string) (ContactMergePreview, error) {
	var empty ContactMergePreview
	if keepID <= 0 || req.RemoveID <= 0 || keepID == req.RemoveID {
		return empty, envelope.NewError(envelope.InputError, u.i18n.T("contact.merge.invalidContacts"), nil)
	}
	tx, err := u.db.Beginx()
	if err != nil {
		return empty, u.mergeError(err)
	}
	defer tx.Rollback()
	var locked []int
	if err := tx.Stmtx(u.q.LockMergeContacts).Select(&locked, pq.Array([]int{keepID, req.RemoveID})); err != nil {
		return empty, u.mergeError(err)
	}
	preview, contacts, err := u.contactMergePreview(tx, keepID, req.RemoveID)
	if err != nil {
		return empty, err
	}
	if req.Version == "" || preview.Version != req.Version {
		return empty, envelope.NewError(envelope.ConflictError, u.i18n.T("contact.merge.changed"), nil)
	}
	values := make(map[string]any, len(preview.Fields))
	conflicts := make(map[string]bool)
	for _, field := range preview.Fields {
		value := field.Keep
		if mergeValueEmpty(value) {
			value = field.Remove
		}
		if field.Conflict {
			conflicts[field.Key] = true
			choice := req.Choices[field.Key]
			if choice != "keep" && choice != "remove" {
				return empty, envelope.NewError(envelope.InputError, u.i18n.T("contact.merge.chooseConflicts"), nil)
			}
			if choice == "remove" {
				value = field.Remove
			}
		}
		values[field.Key] = value
	}
	for key := range req.Choices {
		if !conflicts[key] {
			return empty, envelope.NewError(envelope.InputError, u.i18n.T("contact.merge.chooseConflicts"), nil)
		}
	}
	keep, remove := contacts[0], contacts[1]
	attrs := map[string]any{}
	for key, value := range values {
		if name, ok := strings.CutPrefix(key, "custom."); ok {
			attrs[name] = value
		}
	}
	customAttributes, err := json.Marshal(attrs)
	if err != nil {
		return empty, u.mergeError(err)
	}
	contactType := keep.Type
	if remove.Type == models.UserTypeContact {
		contactType = models.UserTypeContact
	}
	externalID := keep.ExternalUserID
	if externalID.String == "" {
		externalID = remove.ExternalUserID
	}
	phone := values["phone"].(ContactMergePhone)
	if _, err := tx.Stmtx(u.q.RegisterMergeEmails).Exec(remove.ID, keep.ID); err != nil {
		return empty, u.mergeError(err)
	}
	if _, err := tx.Stmtx(u.q.MarkContactMerged).Exec(remove.ID, keep.ID); err != nil {
		return empty, u.mergeError(err)
	}
	if _, err := tx.Stmtx(u.q.SaveMergedContact).Exec(keep.ID, values["first_name"], values["last_name"],
		values["email"], phone.Number, phone.CountryCode, values["country"], values["avatar_url"],
		externalID, customAttributes, keep.Enabled && remove.Enabled, contactType); err != nil {
		return empty, u.mergeError(err)
	}
	if err := tx.Stmtx(u.q.TransferMergeConversations).Select(&preview.ConversationUUIDs, remove.ID, keep.ID); err != nil {
		return empty, u.mergeError(err)
	}
	for _, stmt := range []*sqlx.Stmt{u.q.TransferMergeMessages, u.q.TransferMergeNotes,
		u.q.TransferMergeIdentities, u.q.TransferMergeParticipants, u.q.DeleteMergeParticipants, u.q.TransferMergeLastSeen,
		u.q.DeleteMergeLastSeen, u.q.TransferMergeMedia, u.q.TransferMergeCampaigns, u.q.RedirectContactMerges} {
		if _, err := tx.Stmtx(stmt).Exec(remove.ID, keep.ID); err != nil {
			return empty, u.mergeError(err)
		}
	}
	description := u.i18n.Ts("activityLog.contactMerged", "actorEmail", actorEmail, "removeId", fmt.Sprint(remove.ID), "keepId", fmt.Sprint(keep.ID))
	if len(req.Choices) > 0 {
		choices, err := json.Marshal(req.Choices)
		if err != nil {
			return empty, u.mergeError(err)
		}
		description += "\n" + string(choices)
	}
	if _, err := tx.Stmtx(u.q.LogContactMerge).Exec(description, actorID, keep.ID, ip); err != nil {
		return empty, u.mergeError(err)
	}
	if err := tx.Commit(); err != nil {
		return empty, u.mergeError(err)
	}
	return preview, nil
}

func (u *Manager) ResolveMergedContactID(id int) (int, error) {
	var resolved int
	if err := u.q.ResolveMergedContactID.Get(&resolved, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return id, nil
		}
		return 0, u.mergeError(err)
	}
	return resolved, nil
}

// ResolveMergedContactIDTx holds the current contact until the caller's write commits.
func (u *Manager) ResolveMergedContactIDTx(tx *sqlx.Tx, id int) (int, error) {
	for {
		var reference contactMergeReference
		if err := tx.Stmtx(u.q.LockContactMergeReference).Get(&reference, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return id, nil
			}
			return 0, u.mergeError(err)
		}
		if !reference.MergedIntoID.Valid {
			return reference.ID, nil
		}
		id = int(reference.MergedIntoID.Int64)
	}
}

func (u *Manager) SearchMergeContacts(query string) ([]models.UserCompact, error) {
	contacts := make([]models.UserCompact, 0)
	if err := u.q.SearchMergeContacts.Select(&contacts, dbutil.ContainsPattern(query)); err != nil {
		return nil, u.mergeError(err)
	}
	return contacts, nil
}

func (u *Manager) contactMergePreview(tx *sqlx.Tx, keepID, removeID int) (ContactMergePreview, []mergeContact, error) {
	var preview ContactMergePreview
	if keepID <= 0 || removeID <= 0 || keepID == removeID {
		return preview, nil, envelope.NewError(envelope.InputError, u.i18n.T("contact.merge.invalidContacts"), nil)
	}
	var rows []mergeContact
	if err := tx.Stmtx(u.q.GetMergeContacts).Select(&rows, pq.Array([]int{keepID, removeID})); err != nil {
		return preview, nil, u.mergeError(err)
	}
	if len(rows) != 2 {
		return preview, nil, envelope.NewError(envelope.NotFoundError, u.i18n.T("validation.notFoundUser"), nil)
	}
	if rows[0].ID != keepID {
		slices.Reverse(rows)
	}
	keep, remove := rows[0], rows[1]
	if keep.ExternalUserID.String != "" && remove.ExternalUserID.String != "" && keep.ExternalUserID.String != remove.ExternalUserID.String {
		return preview, nil, envelope.NewError(envelope.InputError, u.i18n.T("contact.merge.differentCustomerIDs"), nil)
	}
	preview.Keep, preview.Remove = mergeProfile(keep), mergeProfile(remove)
	keepValues, err := mergeValues(keep)
	if err != nil {
		return preview, nil, u.mergeError(err)
	}
	removeValues, err := mergeValues(remove)
	if err != nil {
		return preview, nil, u.mergeError(err)
	}
	keys := map[string]any{}
	maps.Copy(keys, keepValues)
	maps.Copy(keys, removeValues)
	for _, key := range slices.Sorted(maps.Keys(keys)) {
		a, b := keepValues[key], removeValues[key]
		preview.Fields = append(preview.Fields, ContactMergeField{Key: key, Keep: a, Remove: b,
			Conflict: !mergeValueEmpty(a) && !mergeValueEmpty(b) && !reflect.DeepEqual(a, b)})
	}
	data, err := json.Marshal(preview, json.Deterministic(true))
	if err != nil {
		return preview, nil, u.mergeError(err)
	}
	version := sha256.Sum256(data)
	preview.Version = hex.EncodeToString(version[:])
	return preview, rows, nil
}

func (u *Manager) mergeError(err error) error {
	u.lo.Error("error merging contacts", "error", err)
	return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
}

func mergeProfile(c mergeContact) ContactMergeProfile {
	return ContactMergeProfile{ID: c.ID, FirstName: c.FirstName, LastName: c.LastName, Email: c.Email.String,
		ExternalUserID: c.ExternalUserID.String, Type: c.Type, Enabled: c.Enabled, Conversations: c.Conversations,
		Notes: c.Notes, Identities: c.ChannelIdentities}
}

func mergeValues(c mergeContact) (map[string]any, error) {
	values := map[string]any{"first_name": c.FirstName, "last_name": c.LastName, "email": c.Email.String,
		"phone":   ContactMergePhone{Number: c.PhoneNumber.String, CountryCode: c.PhoneNumberCountryCode.String},
		"country": c.Country.String, "avatar_url": c.AvatarURL.String}
	attrs := map[string]any{}
	if err := json.Unmarshal(c.CustomAttributes, &attrs); err != nil {
		return nil, err
	}
	for key, value := range attrs {
		values["custom."+key] = value
	}
	return values, nil
}

func mergeValueEmpty(value any) bool {
	if phone, ok := value.(ContactMergePhone); ok {
		return phone.Number == ""
	}
	return value == nil || reflect.DeepEqual(value, "")
}
