// Package stringutil provides string utility functions.
package stringutil

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/k3a/html2text"
)

const (
	PasswordDummy = "•"
)

var (
	regexpNonAlNum = regexp.MustCompile(`[^a-zA-Z0-9\-_\.]+`)
	regexpSpaces   = regexp.MustCompile(`[\s]+`)
)

// HTML2Text converts HTML to text.
func HTML2Text(html string) string {
	return strings.TrimSpace(html2text.HTML2Text(html))
}

// SanitizeFilename sanitizes the provided filename.
func SanitizeFilename(fName string) string {
	// Trim whitespace.
	name := strings.TrimSpace(fName)

	// Replace whitespace and "/" with "-"
	name = regexpSpaces.ReplaceAllString(name, "-")

	// Remove or replace any non-alphanumeric characters
	name = regexpNonAlNum.ReplaceAllString(name, "")

	// Convert to lowercase
	name = strings.ToLower(name)
	return filepath.Base(name)
}

// RandomAlphanumeric generates a random alphanumeric string of length n.
func RandomAlphanumeric(n int) (string, error) {
	const dictionary = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	bytes := make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}

	return string(bytes), nil
}

// RandomNumeric generates a random numeric string of length n.
func RandomNumeric(n int) (string, error) {
	const dictionary = "0123456789"

	bytes := make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}

	return string(bytes), nil
}

// GetPathFromURL extracts the path from a URL.
func GetPathFromURL(u string) (string, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return parsedURL.Path, nil
}

// RemoveEmpty removes empty strings from a slice of strings.
func RemoveEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

// GenerateEmailMessageID generates a RFC-compliant Message-ID for an email, does not include the angle brackets.
// The client is expected to wrap the returned string in angle brackets.
func GenerateEmailMessageID(messageID string, fromAddress string) (string, error) {
	if messageID == "" {
		return "", fmt.Errorf("messageID cannot be empty")
	}

	// Parse from address
	addr, err := mail.ParseAddress(fromAddress)
	if err != nil {
		return "", fmt.Errorf("invalid from address: %w", err)
	}

	// Extract domain with validation
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 || parts[1] == "" {
		return "", fmt.Errorf("invalid domain in from address")
	}
	domain := parts[1]

	// Generate cryptographic random component
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Sanitize messageID for email Message-ID
	cleaner := regexp.MustCompile(`[^\w.-]`) // Allow only alphanum, ., -, _
	cleanmessageID := cleaner.ReplaceAllString(messageID, "_")

	// Ensure cleaned messageID isn't empty
	if cleanmessageID == "" {
		return "", fmt.Errorf("messageID became empty after sanitization")
	}

	// Build RFC-compliant Message-ID
	return fmt.Sprintf("%s-%d-%s@%s",
		cleanmessageID,
		time.Now().UnixNano(), // Nanosecond precision
		strings.TrimRight(base64.URLEncoding.EncodeToString(random), "="), // URL-safe base64 without padding
		domain,
	), nil
}

// RemoveItemByValue removes all instances of a value from a slice of strings.
func RemoveItemByValue(slice []string, value string) []string {
	result := []string{}
	for _, v := range slice {
		if v != value {
			result = append(result, v)
		}
	}
	return result
}

// FormatDuration formats a duration as a string.
func FormatDuration(d time.Duration, includeSeconds bool) string {
	d = d.Round(time.Second)
	h := int64(d.Hours())
	d -= time.Duration(h) * time.Hour
	m := int64(d.Minutes())
	d -= time.Duration(m) * time.Minute
	s := int64(d.Seconds())

	var parts []string
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%d hours", h))
	}
	if m >= 0 {
		parts = append(parts, fmt.Sprintf("%d minutes", m))
	}
	if s > 0 && includeSeconds {
		parts = append(parts, fmt.Sprintf("%d seconds", s))
	}
	return strings.Join(parts, " ")
}

// ValidEmail returns true if it's a valid email else return false.
func ValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return addr.Name == "" && addr.Address == email
}

// ExtractEmail extracts the email address from a string.
func ExtractEmail(s string) (string, error) {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", err
	}
	return addr.Address, nil
}

// DedupAndExcludeString returns a deduplicated []string excluding empty and a specific value.
func DedupAndExcludeString(list []string, exclude string) []string {
	seen := make(map[string]struct{}, len(list))
	cleaned := make([]string, 0, len(list))
	for _, s := range list {
		if s == "" || s == exclude {
			continue
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			cleaned = append(cleaned, s)
		}
	}
	return cleaned
}

// ComputeRecipients computes new recipients using last message's recipients and direction.
func ComputeRecipients(
	from, to, cc, bcc []string,
	contactEmail, inboxEmail string,
	lastMessageIncoming bool,
) (finalTo, finalCC, finalBCC []string) {
	if lastMessageIncoming {
		if len(from) > 0 {
			finalTo = from
		} else if contactEmail != "" {
			finalTo = []string{contactEmail}
		}
	} else {
		if len(to) > 0 {
			finalTo = to
		} else if contactEmail != "" {
			finalTo = []string{contactEmail}
		}
	}

	finalCC = append([]string{}, cc...)

	if lastMessageIncoming {
		if len(to) > 0 {
			finalCC = append(finalCC, to...)
		}
		if contactEmail != "" && !slices.Contains(finalTo, contactEmail) && !slices.Contains(finalCC, contactEmail) {
			finalCC = append(finalCC, contactEmail)
		}
	}

	finalTo = DedupAndExcludeString(finalTo, inboxEmail)
	finalCC = DedupAndExcludeString(finalCC, inboxEmail)
	// BCC is one-time only, user is supposed to add it manually.
	finalBCC = []string{}

	return
}
