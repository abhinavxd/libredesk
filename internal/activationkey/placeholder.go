package activationkey

import (
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// PlaceholderType is the data-type of the composer's key chip:
//
//	<span data-type="activation-key" data-id="458219512" data-app-id="3">&lt;KEY_ID_458219512&gt;</span>
//
// Only this element stands for a key. Text that merely looks like
// <KEY_ID_…> is ordinary text and is never replaced.
const PlaceholderType = "activation-key"

// MaskedKey replaces a sent key wherever it is shown back to someone who may
// not see it, e.g. quoted in a customer's reply.
const MaskedKey = "[activation-key-hidden]"

// ErrInvalidPlaceholder means an element claims to be a key placeholder but
// is not exactly the shape the composer produces.
var ErrInvalidPlaceholder = errors.New("invalid activation key placeholder")

var (
	placeholderIDPattern = regexp.MustCompile(`^[0-9]{6,20}$`)
	// Keys are restricted to letters, digits and inner hyphens so that a key
	// can never carry markup or template syntax into an email.
	keyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{6,98}[A-Za-z0-9]$`)
)

// Placeholder is one key slot placed in an email with the composer's key card.
type Placeholder struct {
	ID    string
	AppID int
}

// ParsePlaceholders returns the placeholders in content, in order. Malformed
// or repeated placeholders are an error rather than being skipped, so what is
// validated before queueing is exactly what is replaced at send time.
func ParsePlaceholders(content string) ([]Placeholder, error) {
	_, found, err := walk(content, nil)
	return found, err
}

// ReplacePlaceholders swaps each placeholder element, start tag to end tag,
// for replace's output and copies every other byte of content unchanged.
func ReplacePlaceholders(content string, replace func(Placeholder) string) (string, error) {
	out, _, err := walk(content, replace)
	return out, err
}

func walk(content string, replace func(Placeholder) string) (string, []Placeholder, error) {
	var (
		z     = html.NewTokenizer(strings.NewReader(content))
		out   strings.Builder
		found []Placeholder
		seen  = map[string]bool{}
	)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if errors.Is(z.Err(), io.EOF) {
				return out.String(), found, nil
			}
			return "", nil, z.Err()
		}
		raw := string(z.Raw())
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			out.WriteString(raw)
			continue
		}
		name, attrs := tagAttrs(z)
		if !strings.EqualFold(strings.TrimSpace(attrs["data-type"]), PlaceholderType) {
			out.WriteString(raw)
			continue
		}

		// From here on this element is a placeholder and must be well formed.
		if name != "span" || tt != html.StartTagToken {
			return "", nil, ErrInvalidPlaceholder
		}
		p := Placeholder{ID: attrs["data-id"]}
		appID, err := strconv.Atoi(attrs["data-app-id"])
		if err != nil || appID <= 0 || !placeholderIDPattern.MatchString(p.ID) || seen[p.ID] {
			return "", nil, ErrInvalidPlaceholder
		}
		p.AppID = appID
		seen[p.ID] = true

		// The chip holds only its label text; any nested markup, comment or a
		// missing end tag would make its extent ambiguous.
		element := raw
		for closed := false; !closed; {
			switch z.Next() {
			case html.TextToken:
				element += string(z.Raw())
			case html.EndTagToken:
				if endName, _ := z.TagName(); string(endName) != "span" {
					return "", nil, ErrInvalidPlaceholder
				}
				element += string(z.Raw())
				closed = true
			default:
				return "", nil, ErrInvalidPlaceholder
			}
		}
		found = append(found, p)
		if replace != nil {
			out.WriteString(replace(p))
		} else {
			out.WriteString(element)
		}
	}
}

// tagAttrs reads the current tag's name and attributes. A repeated attribute
// keeps its first value, as browsers do.
func tagAttrs(z *html.Tokenizer) (string, map[string]string) {
	name, more := z.TagName()
	attrs := map[string]string{}
	for more {
		var k, v []byte
		k, v, more = z.TagAttr()
		if _, dup := attrs[string(k)]; !dup {
			attrs[string(k)] = string(v)
		}
	}
	return string(name), attrs
}

// NormalizeKey trims a key and reports whether it is acceptable.
func NormalizeKey(raw string) (string, bool) {
	key := strings.TrimSpace(raw)
	return key, keyPattern.MatchString(key)
}

// keyRunPattern finds the runs of key characters a quoted key can be. A key
// is recognised only as a whole run, standing apart from the letters and
// digits around it, as it does in the email that carried it.
var keyRunPattern = regexp.MustCompile(`[A-Za-z0-9-]{8,}`)

// keyIn returns the key-shaped core of a run, without the hyphens that may
// join it to surrounding punctuation.
func keyIn(run string) (string, bool) {
	key := strings.Trim(run, "-")
	return key, keyPattern.MatchString(key)
}

// keyRuns lists the key-shaped runs in content.
func keyRuns(content string) []string {
	var keys []string
	for _, run := range keyRunPattern.FindAllString(content, -1) {
		if key, ok := keyIn(run); ok {
			keys = append(keys, key)
		}
	}
	return keys
}

// maskRuns replaces each key-shaped run in content that hide reports on.
func maskRuns(content string, hide func(key string) bool) string {
	return keyRunPattern.ReplaceAllStringFunc(content, func(run string) string {
		key, ok := keyIn(run)
		if !ok || !hide(key) {
			return run
		}
		return strings.Replace(run, key, MaskedKey, 1)
	})
}
