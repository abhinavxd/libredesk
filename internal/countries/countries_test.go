package countries

import "testing"

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
