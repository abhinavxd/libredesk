package inbox

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/abhinavxd/libredesk/internal/inbox/models"
)

var (
	ErrInvalidFromAddress  = errors.New("invalid inbox from address")
	ErrInvalidAliasAddress = errors.New("alias must be a bare email address")
	ErrDuplicateAddress    = errors.New("duplicate inbox email address")
)

// NormalizeEmailAddress extracts and lowercases an email address.
func NormalizeEmailAddress(value string) (string, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil || addr.Address == "" {
		return "", fmt.Errorf("invalid email address %q", value)
	}
	return strings.ToLower(strings.TrimSpace(addr.Address)), nil
}

// ValidateEmailAddresses normalizes a primary address and its bare aliases.
func ValidateEmailAddresses(from string, aliases models.EmailAliases) (string, models.EmailAliases, error) {
	primary, err := NormalizeEmailAddress(from)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %q", ErrInvalidFromAddress, from)
	}
	seen := map[string]struct{}{primary: {}}
	normalized := make(models.EmailAliases, 0, len(aliases))
	for _, raw := range aliases {
		addr, err := mail.ParseAddress(strings.TrimSpace(raw.Email))
		if err != nil || addr.Address == "" || addr.Name != "" {
			return "", nil, fmt.Errorf("%w: %q", ErrInvalidAliasAddress, raw.Email)
		}
		alias := strings.ToLower(addr.Address)
		if _, ok := seen[alias]; ok {
			return "", nil, fmt.Errorf("%w: %q", ErrDuplicateAddress, alias)
		}
		seen[alias] = struct{}{}
		status := raw.VerificationStatus
		if status == "" {
			status = models.AliasVerificationNotVerified
		}
		normalized = append(normalized, models.EmailAlias{
			Email:              alias,
			VerificationStatus: status,
			VerifiedAt:         raw.VerifiedAt,
		})
	}
	return primary, normalized, nil
}

// SendableEmailAddresses returns the primary address and verified send aliases.
func SendableEmailAddresses(from string, aliases models.EmailAliases) ([]string, error) {
	primary, normalized, err := ValidateEmailAddresses(from, aliases)
	if err != nil {
		return nil, err
	}
	addresses := []string{primary}
	for _, alias := range normalized {
		if alias.VerificationStatus == models.AliasVerificationVerified {
			addresses = append(addresses, alias.Email)
		}
	}
	return addresses, nil
}
