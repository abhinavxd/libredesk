package countries

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
)

//go:embed countries.json
var countriesJSON []byte

// sharedDialCodePrimary picks the ISO-2 that owns a dial code shared by several countries, since file order alone would resolve +1 to CA and +7 to KZ.
var sharedDialCodePrimary = map[string]string{
	"1": "US",
	"7": "RU",
}

var dialDigitsReplacer = strings.NewReplacer("+", "", "-", "")

var phoneCountryCodes = buildPhoneCountryCodes()

type phoneCode struct{ prefix, iso string }

// JSON returns the raw country reference list (calling code, name, emoji, ISO-2) embedded at build time.
func JSON() []byte {
	return countriesJSON
}

// SplitPhoneCountryCode splits a full international number without "+" (e.g. a WhatsApp wa_id) into the ISO-2 country code and national number; returns ("", input) when no prefix matches.
func SplitPhoneCountryCode(full string) (string, string) {
	for _, c := range phoneCountryCodes {
		if len(full) > len(c.prefix) && full[:len(c.prefix)] == c.prefix {
			return c.iso, full[len(c.prefix):]
		}
	}
	return "", full
}

// DialCodeForISO returns the dialing-code digits for an ISO-2 country, "" when unknown.
func DialCodeForISO(iso string) string {
	out := ""
	for _, c := range phoneCountryCodes {
		if c.iso == iso && (out == "" || len(c.prefix) < len(out)) {
			out = c.prefix
		}
	}
	return out
}

func buildPhoneCountryCodes() []phoneCode {
	var list []struct {
		CallingCode string `json:"calling_code"`
		ISO2        string `json:"iso_2"`
	}
	if err := json.Unmarshal(countriesJSON, &list); err != nil {
		panic("countries: invalid countries.json: " + err.Error())
	}

	byPrefix := map[string]string{}
	for _, c := range list {
		prefix := dialDigitsReplacer.Replace(c.CallingCode)
		if prefix == "" || c.ISO2 == "" {
			continue
		}
		if primary, ok := sharedDialCodePrimary[prefix]; ok {
			byPrefix[prefix] = primary
			continue
		}
		if _, seen := byPrefix[prefix]; !seen {
			byPrefix[prefix] = c.ISO2
		}
	}

	out := make([]phoneCode, 0, len(byPrefix))
	for prefix, iso := range byPrefix {
		out = append(out, phoneCode{prefix, iso})
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].prefix) != len(out[j].prefix) {
			return len(out[i].prefix) > len(out[j].prefix)
		}
		return out[i].prefix < out[j].prefix
	})
	return out
}
