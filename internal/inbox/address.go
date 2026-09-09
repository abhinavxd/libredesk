package inbox

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/abhinavxd/libredesk/internal/inbox/models"
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
		return "", nil, err
	}
	seen := map[string]struct{}{primary: {}}
	normalized := make(models.EmailAliases, 0, len(aliases))
	for _, raw := range aliases {
		addr, err := mail.ParseAddress(strings.TrimSpace(raw.Email))
		if err != nil || addr.Address == "" || addr.Name != "" {
			return "", nil, fmt.Errorf("alias %q must be a bare email address", raw.Email)
		}
		alias := strings.ToLower(addr.Address)
		if _, ok := seen[alias]; ok {
			return "", nil, fmt.Errorf("duplicate inbox email address %q", alias)
		}
		seen[alias] = struct{}{}
		normalized = append(normalized, models.EmailAlias{
			Email:              alias,
			VerificationStatus: raw.VerificationStatus,
			VerifiedAt:         raw.VerifiedAt,
		})
		if normalized[len(normalized)-1].VerificationStatus == "" {
			normalized[len(normalized)-1].VerificationStatus = models.AliasVerificationNotVerified
		}
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
