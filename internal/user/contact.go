package user

import (
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
)

// CreateContact creates a new contact user.
func (u *Manager) CreateContact(user *models.User) error {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("generating password", "error", err)
		return fmt.Errorf("generating password: %w", err)
	}

	// Normalize email address.
	user.Email = null.NewString(strings.ToLower(user.Email.String), user.Email.Valid)

	// If external_user_id is provided, insert with it.
	if user.ExternalUserID.Valid {
		if err := u.q.InsertContactWithExtID.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL, user.ExternalUserID, user.CustomAttributes).Scan(&user.ID); err != nil {
			u.lo.Error("error inserting contact with external ID", "error", err)
			return fmt.Errorf("insert contact with external ID: %w", err)
		}
		return nil
	}
	// Insert without external_user_id.
	if err := u.q.InsertContactNoExtID.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL).Scan(&user.ID); err != nil {
		u.lo.Error("error inserting contact", "error", err)
		return fmt.Errorf("insert contact: %w", err)
	}
	return nil
}

// UpdateContact updates a contact in the database.
func (u *Manager) UpdateContact(id int, user models.User) error {
	if _, err := u.q.UpdateContact.Exec(id, user.FirstName, user.LastName, user.Email, user.AvatarURL, user.PhoneNumber, user.PhoneNumberCountryCode); err != nil {
		u.lo.Error("error updating user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// GetContact retrieves a contact by ID.
func (u *Manager) GetContact(id int, email string) (models.User, error) {
	return u.Get(id, email, []string{models.UserTypeContact, models.UserTypeVisitor})
}

// GetAllContacts returns a list of all contacts.
func (u *Manager) GetContacts(page, pageSize int, order, orderBy string, filtersJSON string) ([]models.UserCompact, error) {
	if pageSize > maxListPageSize {
		return nil, envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.pageTooLarge", "max", fmt.Sprintf("%d", maxListPageSize)), nil)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return u.GetAllUsers(page, pageSize, []string{models.UserTypeContact, models.UserTypeVisitor}, order, orderBy, filtersJSON)
}
