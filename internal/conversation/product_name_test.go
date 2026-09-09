package conversation

import "testing"

func TestProductNameFromInbox(t *testing.T) {
	cases := map[string]string{
		"Drifttt Support":          "Drifttt",
		"Drifttt App":              "Drifttt",
		"Maria App Studio Support": "Maria App Studio",
		"Drifttt":                  "Drifttt",
		"  Studio Helpdesk ":       "Studio",
		"Support":                  "Support",
		"":                         "",
	}
	for in, want := range cases {
		if got := ProductNameFromInbox(in); got != want {
			t.Errorf("ProductNameFromInbox(%q) = %q, want %q", in, got, want)
		}
	}
}
