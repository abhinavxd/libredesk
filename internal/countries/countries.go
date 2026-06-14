package countries

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed countries.json
var countriesJSON []byte

var dialDigitsReplacer = strings.NewReplacer("+", "", "-", "")

var dialCodeByISO = buildDialCodes()

// JSON returns the raw country reference list (calling code, name, emoji, ISO-2) embedded at build time.
func JSON() []byte {
	return countriesJSON
}

// DialCodeForISO returns the dialing-code digits for an ISO-2 country, "" when unknown.
func DialCodeForISO(iso string) string {
	return dialCodeByISO[iso]
}

func buildDialCodes() map[string]string {
	var list []struct {
		CallingCode string `json:"calling_code"`
		ISO2        string `json:"iso_2"`
	}
	if err := json.Unmarshal(countriesJSON, &list); err != nil {
		panic("countries: invalid countries.json: " + err.Error())
	}

	dialByISO := map[string]string{}
	for _, c := range list {
		prefix := dialDigitsReplacer.Replace(c.CallingCode)
		if prefix == "" || c.ISO2 == "" {
			continue
		}
		if existing, ok := dialByISO[c.ISO2]; !ok || len(prefix) < len(existing) {
			dialByISO[c.ISO2] = prefix
		}
	}
	return dialByISO
}
