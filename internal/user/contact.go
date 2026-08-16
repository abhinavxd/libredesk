package user

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
)

// ResolveContact resolves user.ID to a contact by external_user_id or email, creating one if absent; policy decides if a matched contact may be updated.
func (u *Manager) ResolveContact(user *models.User, policy models.ContactPolicy) error {
	if len(user.CustomAttributes) == 0 {
		user.CustomAttributes = []byte("{}")
	}

	user.Email = null.NewString(strings.ToLower(strings.TrimSpace(user.Email.String)), user.Email.Valid)

	if policy == models.ContactSync {
		return u.syncContact(user)
	}
	return u.reuseContact(user)
}

// UpdateContactBasicInfo updates only the name, email and phone of a contact.
func (u *Manager) UpdateContactBasicInfo(id int, firstName, lastName, email, phoneNumber, phoneNumberCountryCode string) error {
	if _, err := u.q.UpdateContactBasicInfo.Exec(id, firstName, lastName, strings.ToLower(strings.TrimSpace(email)), phoneNumber, phoneNumberCountryCode); err != nil {
		u.lo.Error("error updating contact basic info", "error", err)
		return fmt.Errorf("updating contact basic info: %w", err)
	}
	return nil
}

func (u *Manager) UpdateContact(id int, user models.User) error {
	if _, err := u.q.UpdateContact.Exec(id, user.FirstName, user.LastName, user.Email, user.AvatarURL, user.PhoneNumber, user.PhoneNumberCountryCode, user.Country); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.InputError, u.i18n.T("contact.alreadyExistsWithEmail"), nil)
		}
		u.lo.Error("error updating user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// DeleteContact permanently deletes a contact or visitor; conversations, messages, and notes are removed by DB cascades.
func (u *Manager) DeleteContact(id int) error {
	res, err := u.q.DeleteContact.Exec(id)
	if err != nil {
		u.lo.Error("error deleting contact", "contact_id", id, "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return envelope.NewError(envelope.NotFoundError, u.i18n.T("validation.notFoundUser"), nil)
	}
	return nil
}

// ExportContactData returns a contact's profile, non-private conversation messages, and CSAT responses as JSON.
func (u *Manager) ExportContactData(id int) ([]byte, error) {
	var data []byte
	if err := u.q.ExportContactData.Get(&data, id); err != nil {
		u.lo.Error("error exporting contact data", "contact_id", id, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return data, nil
}

func (u *Manager) GetContacts(page, pageSize int, order, orderBy string, filtersJSON, location string) ([]models.UserCompact, error) {
	if pageSize > maxListPageSize {
		pageSize = maxListPageSize
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return u.GetAllUsers(page, pageSize, []string{models.UserTypeContact, models.UserTypeVisitor}, order, orderBy, filtersJSON, location)
}

// reuseContact resolves to an existing contact without ever updating it, inserting one only if absent.
func (u *Manager) reuseContact(user *models.User) error {
	// An unknown ext_id falls back to the email lookup, else a duplicate contact with the same email gets inserted.
	lookup := func() (models.User, error) {
		if user.ExternalUserID.String != "" {
			existing, err := u.GetContactByExternalID(user.ExternalUserID.String)
			if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError || user.Email.String == "" {
				return existing, err
			}
		}
		return u.GetContactByEmail(user.Email.String)
	}

	reuse := func(existing models.User) {
		user.ID = existing.ID
		if existing.Email.String != "" {
			user.Email = existing.Email
		}
	}

	existing, err := lookup()
	if err == nil {
		reuse(existing)
		return nil
	}
	if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
		return err
	}

	password, err := u.newContactPassword()
	if err != nil {
		return err
	}

	err = u.q.InsertContactIfAbsent.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL, user.ExternalUserID, user.CustomAttributes).Scan(&user.ID)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		u.lo.Error("error inserting contact", "error", err)
		return fmt.Errorf("insert contact: %w", err)
	}

	// No returned row means a concurrent request inserted the contact - reuse it.
	existing, err = lookup()
	if err != nil {
		return err
	}
	reuse(existing)
	return nil
}

// syncContact resolves to an existing contact and updates its name/email, enriching external_user_id, inserting one if absent.
func (u *Manager) syncContact(user *models.User) error {
	// Check if email matches an existing contact without ext_id - enrich it.
	if user.ExternalUserID.String != "" {
		if user.Email.Valid && user.Email.String != "" {
			existing, emailErr := u.GetContactByEmailWithoutExtID(user.Email.String)
			if emailErr != nil {
				if envErr, ok := emailErr.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
					return emailErr
				}
			} else {
				enriched, setErr := u.SetExternalUserID(existing.ID, user.ExternalUserID.String)
				if setErr != nil && !dbutil.IsUniqueViolationError(setErr) {
					return setErr
				}
				if enriched {
					user.ID = existing.ID
					return nil
				}
				// ext_id already belongs to another contact, or the contact was deleted mid-flight - fall through to upsert.
				u.lo.Info("skipping contact enrichment, falling back to upsert", "contact_id", existing.ID, "ext_id", user.ExternalUserID.String)
			}
		}

		password, err := u.newContactPassword()
		if err != nil {
			return err
		}

		// Upsert by ext_id - creates new or updates email/name on ext_id conflict.
		if err := u.q.InsertContactWithExtID.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL, user.ExternalUserID, user.CustomAttributes, user.PhoneNumber, user.PhoneNumberCountryCode).Scan(&user.ID); err != nil {
			u.lo.Error("error inserting contact with external ID", "error", err)
			return fmt.Errorf("inserting contact with external ID: %w", err)
		}
		return nil
	}

	if user.Email.Valid && user.Email.String != "" {
		// An ext_id contact owns this email - reuse it; the no-ext-id upsert below can't match it and would insert a duplicate.
		existing, err := u.GetContactByEmail(user.Email.String)
		if err == nil && existing.ExternalUserID.String != "" {
			user.ID = existing.ID
			return nil
		}

		// Other error than not found - fail.
		if err != nil {
			if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
				return err
			}
		}
	}

	password, err := u.newContactPassword()
	if err != nil {
		return err
	}

	// No ext_id contact for this email - insert new, or update the existing no-ext-id contact's name.
	if err := u.q.InsertContactNoExtID.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL).Scan(&user.ID); err != nil {
		u.lo.Error("error inserting contact", "error", err)
		return fmt.Errorf("insert contact: %w", err)
	}
	return nil
}

func (u *Manager) newContactPassword() ([]byte, error) {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("generating password", "error", err)
		return nil, fmt.Errorf("generating password: %w", err)
	}
	return password, nil
}

func (u *Manager) GetContactIDByChannelIdentity(channel, identifier string) (int, error) {
	var id int
	if err := u.q.GetContactIDByChannelIdentity.Get(&id, channel, identifier); err != nil {
		if err == sql.ErrNoRows {
			return 0, envelope.NewError(envelope.NotFoundError, u.i18n.T("validation.notFoundUser"), nil)
		}
		u.lo.Error("error fetching contact by channel identity", "channel", channel, "error", err)
		return 0, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return id, nil
}

func (u *Manager) LinkChannelIdentity(contactID int, channel, identifier string) (int, error) {
	var linkedID int
	if err := u.q.InsertChannelIdentity.QueryRow(contactID, channel, identifier).Scan(&linkedID); err != nil {
		u.lo.Error("error linking channel identity", "contact_id", contactID, "channel", channel, "error", err)
		return 0, fmt.Errorf("linking channel identity: %w", err)
	}
	return linkedID, nil
}

// UpdateChannelIdentity returns the contact id, or 0 when the new identifier already belongs to a contact.
func (u *Manager) UpdateChannelIdentity(channel, oldIdentifier, newIdentifier string) (int, error) {
	var contactID int
	if err := u.q.UpdateChannelIdentity.QueryRow(channel, oldIdentifier, newIdentifier).Scan(&contactID); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		u.lo.Error("error updating channel identity", "channel", channel, "error", err)
		return 0, fmt.Errorf("updating channel identity: %w", err)
	}
	return contactID, nil
}

func (u *Manager) UpsertContactByChannelIdentity(channel, identifier string, contact *models.User) (int, error) {
	id, err := u.GetContactIDByChannelIdentity(channel, identifier)
	if err == nil {
		return id, nil
	}
	if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
		return 0, err
	}
	// Contacts with no email and no ext_id have no uniqueness key, so ResolveContact + LinkChannelIdentity
	// are not atomic - a failed link leaves an orphan user row that grows on retry. Use the atomic CTE instead.
	if contact.Email.String == "" && contact.ExternalUserID.String == "" {
		return u.upsertContactWithChannelIdentity(channel, identifier, contact)
	}
	if err := u.ResolveContact(contact, models.ContactSync); err != nil {
		return 0, err
	}
	return u.LinkChannelIdentity(contact.ID, channel, identifier)
}

func (u *Manager) upsertContactWithChannelIdentity(channel, identifier string, contact *models.User) (int, error) {
	password, err := u.generatePassword()
	if err != nil {
		return 0, fmt.Errorf("generating password: %w", err)
	}
	var id int
	if err := u.q.UpsertContactWithChannelIdentity.QueryRow(
		contact.Email, contact.FirstName, contact.LastName, password, contact.AvatarURL,
		channel, identifier,
	).Scan(&id); err != nil {
		u.lo.Error("error upserting contact with channel identity", "channel", channel, "identifier", identifier, "error", err)
		return 0, fmt.Errorf("upserting contact with channel identity: %w", err)
	}
	contact.ID = id
	return id, nil
}

// SetContactPhoneIfMissing sets phone_number only when it is empty, never clobbering an agent-curated value.
func (u *Manager) SetContactPhoneIfMissing(id int, phone, countryCode string) error {
	if id == 0 || phone == "" {
		return nil
	}
	if _, err := u.q.SetContactPhoneIfMissing.Exec(id, phone, countryCode); err != nil {
		u.lo.Error("error setting contact phone number", "id", id, "error", err)
		return fmt.Errorf("setting contact phone number: %w", err)
	}
	return nil
}

func (u *Manager) GetChannelIdentities(contactID int) ([]models.ChannelIdentity, error) {
	out := make([]models.ChannelIdentity, 0)
	if err := u.q.GetChannelIdentitiesByContact.Select(&out, contactID); err != nil {
		u.lo.Error("error fetching channel identities", "contact_id", contactID, "error", err)
		return nil, fmt.Errorf("fetching channel identities: %w", err)
	}
	return out, nil
}

// GetChannelIdentity returns the contact's identifier on a channel, "" with nil error when none.
func (u *Manager) GetChannelIdentity(contactID int, channel string) (string, error) {
	var identifier string
	if err := u.q.GetChannelIdentity.Get(&identifier, contactID, channel); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		u.lo.Error("error fetching channel identity", "contact_id", contactID, "channel", channel, "error", err)
		return "", fmt.Errorf("fetching channel identity: %w", err)
	}
	return identifier, nil
}

// UpdateContactNameIfDefault replaces the name only while it still equals defaultName, never over agent edits.
func (u *Manager) UpdateContactNameIfDefault(id int, firstName, lastName, defaultName string) error {
	if id == 0 || firstName == "" {
		return nil
	}
	if _, err := u.q.UpdateContactNameIfDefault.Exec(id, firstName, lastName, defaultName); err != nil {
		u.lo.Error("error updating contact name", "id", id, "error", err)
		return fmt.Errorf("updating contact name: %w", err)
	}
	return nil
}
