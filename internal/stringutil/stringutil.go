// Package stringutil provides string utility functions.
package stringutil

import (
	"crypto/rand"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

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
