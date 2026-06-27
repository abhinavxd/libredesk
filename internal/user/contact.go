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

func (u *Manager) CreateContact(user *models.User) error {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("generating password", "error", err)
		return fmt.Errorf("generating password: %w", err)
	}

	if len(user.CustomAttributes) == 0 {
		user.CustomAttributes = []byte("{}")
	}

	// Normalize.
	user.Email = null.NewString(strings.ToLower(strings.TrimSpace(user.Email.String)), user.Email.Valid)

	// Check if email matches an existing contact without ext_id - enrich it.
	if user.ExternalUserID.String != "" {
		if user.Email.Valid && user.Email.String != "" {
			existing, emailErr := u.GetContactByEmailWithoutExtID(user.Email.String)
			if emailErr != nil {
				if envErr, ok := emailErr.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
					return emailErr
				}
			} else {
				if setErr := u.SetExternalUserID(existing.ID, user.ExternalUserID.String); setErr == nil {
					user.ID = existing.ID
					return nil
				}
				// ext_id already belongs to another contact - fall through to upsert.
				u.lo.Info("ext_id already exists on another contact, skipping enrichment", "contact_id", existing.ID, "ext_id", user.ExternalUserID.String)
			}
		}

		// Upsert by ext_id - creates new or updates email/name on ext_id conflict.
		if err := u.q.InsertContactWithExtID.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL, user.ExternalUserID, user.CustomAttributes).Scan(&user.ID); err != nil {
			u.lo.Error("error inserting contact with external ID", "error", err)
			return fmt.Errorf("inserting contact with external ID: %w", err)
		}
		return nil
	}

	if user.Email.Valid && user.Email.String != "" {
		// Reuse any existing contact with this email, preferring one with ext_id if multiple exist.
		existing, err := u.GetContactByEmail(user.Email.String)
		if err == nil {
			user.ID = existing.ID
			return nil
		}

		// Other error than not found - fail.
		if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
			return err
		}
	}

	// No ext_id and no existing contact with email - create new.
	if err := u.q.InsertContactNoExtID.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL).Scan(&user.ID); err != nil {
		u.lo.Error("error inserting contact", "error", err)
		return fmt.Errorf("insert contact: %w", err)
	}
	return nil
}

// UpdateContactBasicInfo updates only the name and email of a contact.
func (u *Manager) UpdateContactBasicInfo(id int, firstName, lastName, email string) error {
	if _, err := u.q.UpdateContactBasicInfo.Exec(id, firstName, lastName, strings.ToLower(strings.TrimSpace(email))); err != nil {
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

// GetAllContacts returns a list of all contacts.
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

// UpsertContactByChannelIdentity resolves the contact for a channel identity, creating and linking one when the identity is new.
func (u *Manager) UpsertContactByChannelIdentity(channel, identifier string, contact *models.User) (int, error) {
	id, err := u.GetContactIDByChannelIdentity(channel, identifier)
	if err == nil {
		return id, nil
	}
	if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
		return 0, err
	}
	// Contacts with no email and no ext_id have no uniqueness key, so CreateContact + LinkChannelIdentity
	// are not atomic - a failed link leaves an orphan user row that grows on retry. Use the atomic CTE instead.
	if contact.Email.String == "" && contact.ExternalUserID.String == "" {
		return u.upsertContactWithChannelIdentity(channel, identifier, contact)
	}
	if err := u.CreateContact(contact); err != nil {
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

// GetChannelIdentities returns all channel identities linked to a contact.
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
