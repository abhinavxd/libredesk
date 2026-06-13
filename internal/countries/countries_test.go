package countries

import "testing"

func TestSplitPhoneCountryCode(t *testing.T) {
	tests := []struct {
		input    string
		code     string
		national string
	}{
		{input: "919158696230", code: "IN", national: "9158696230"},
		{input: "14155552671", code: "US", national: "4155552671"},
		{input: "16845551234", code: "AS", national: "5551234"},
		{input: "447911123456", code: "GB", national: "7911123456"},
		{input: "971501234567", code: "AE", national: "501234567"},
		{input: "999123", code: "", national: "999123"},
		// Longest matching prefix wins: +44 1481 (Guernsey) over +44 (UK).
		{input: "441481570000", code: "GG", national: "570000"},
		{input: "12423001234", code: "BS", national: "3001234"},
		// Shared dial codes collapse to the designated primary: +1 -> US (not CA), +7 -> RU (not KZ).
		{input: "14165551234", code: "US", national: "4165551234"},
		{input: "77011234567", code: "RU", national: "7011234567"},
		// Three-digit code shared with Western Sahara collapses to Morocco.
		{input: "212600000000", code: "MA", national: "600000000"},
		// No national digits after the code is not a match.
		{input: "91", code: "", national: "91"},
		// Empty input.
		{input: "", code: "", national: ""},
		// Leading "+" is not handled here (wa_id is digits only).
		{input: "+919158696230", code: "", national: "+919158696230"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			code, national := SplitPhoneCountryCode(tt.input)
			if code != tt.code || national != tt.national {
				t.Errorf("SplitPhoneCountryCode(%q) = (%q, %q), want (%q, %q)", tt.input, code, national, tt.code, tt.national)
			}
		})
	}
}

func TestDialCodeForISO(t *testing.T) {
	tests := []struct {
		iso  string
		want string
	}{
		{iso: "IN", want: "91"},
		{iso: "US", want: "1"},
		{iso: "GB", want: "44"},
		{iso: "AE", want: "971"},
		{iso: "AS", want: "1684"},
		{iso: "RU", want: "7"},
		// Secondaries that share a dial code still resolve to that dial code.
		{iso: "CA", want: "1"},
		{iso: "KZ", want: "7"},
		{iso: "ZZ", want: ""},
		{iso: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.iso, func(t *testing.T) {
			if got := DialCodeForISO(tt.iso); got != tt.want {
				t.Errorf("DialCodeForISO(%q) = %q, want %q", tt.iso, got, tt.want)
			}
		})
	}
}
