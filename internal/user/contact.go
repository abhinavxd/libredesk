package user

import (
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
