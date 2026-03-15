package user

import (
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
)

// CreateVisitor creates a new visitor user.
func (u *Manager) CreateVisitor(user *models.User) error {
	// Normalize email address.
	user.Email = null.NewString(strings.ToLower(user.Email.String), user.Email.Valid)

	if user.FirstName == "" && user.LastName == "" {
		if user.Email.Valid && user.Email.String != "" {
			user.FirstName = strings.Split(user.Email.String, "@")[0]
		} else {
			user.FirstName = "Visitor"
		}
	}

	if err := u.q.InsertVisitor.Get(user, user.Email, user.FirstName, user.LastName, user.CustomAttributes); err != nil {
		u.lo.Error("error inserting contact", "error", err)
		return fmt.Errorf("insert contact: %w", err)
	}
	return nil
}

// GetVisitor retrieves a visitor user by ID
func (u *Manager) GetVisitor(id int) (models.User, error) {
	return u.Get(id, "", []string{models.UserTypeVisitor})
}
