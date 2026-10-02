package email

import (
	"net/mail"
	"strings"

	imodels "github.com/jakedolan443/fernmail/internal/inbox/models"
)

var recipientHeaders = []string{"Delivered-To", "X-Original-To", "To", "Cc"}

// normalizeRecipient turns a mailbox header value into a comparable address.
// It also removes plus-addressing so replies to alias+conv-<uuid> still map to
// the configured alias.
func normalizeRecipient(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}

	if parsed, err := mail.ParseAddress(value); err == nil {
		value = strings.ToLower(strings.TrimSpace(parsed.Address))
	}

	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 {
		return value
	}
	if plus := strings.IndexByte(parts[0], '+'); plus >= 0 {
		parts[0] = parts[0][:plus]
	}
	return parts[0] + "@" + parts[1]
}

func configuredAliases(aliases []imodels.EmailAlias) map[string]imodels.EmailAlias {
	result := make(map[string]imodels.EmailAlias, len(aliases))
	for _, alias := range aliases {
		// Retirement disables sending, not delivery ownership: mail still
		// addressed here must retain this address's access boundary.
		address := normalizeRecipient(alias.Address)
		if address != "" {
			alias.Address = address
			result[address] = alias
		}
	}
	return result
}

func headerAddresses(value string) []string {
	if value == "" {
		return nil
	}
	if parsed, err := mail.ParseAddressList(value); err == nil {
		addresses := make([]string, 0, len(parsed))
		for _, address := range parsed {
			addresses = append(addresses, address.Address)
		}
		return addresses
	}
	return strings.Split(value, ",")
}

// resolveRecipientAlias returns the first configured alias found in the
// delivery headers. Delivered-To and X-Original-To are preferred because they
// preserve the envelope recipient when a message was addressed to multiple
// aliases.
func resolveRecipientAlias(aliases []imodels.EmailAlias, headers map[string]string) string {
	configured := configuredAliases(aliases)
	for _, header := range recipientHeaders {
		for _, value := range headerAddresses(headers[header]) {
			candidate := normalizeRecipient(value)
			if _, ok := configured[candidate]; ok {
				return candidate
			}
		}
	}

	for _, alias := range aliases {
		if alias.Default {
			return normalizeRecipient(alias.Address)
		}
	}
	return ""
}
