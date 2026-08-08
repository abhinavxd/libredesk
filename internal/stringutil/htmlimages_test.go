package stringutil

import "testing"

func TestDeferOffscreenImages(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "first image stays eager",
			in:   `<p>a</p><img src="1.png"><p>b</p><img src="2.png">`,
			want: `<p>a</p><img src="1.png"><p>b</p><img src="2.png" loading="lazy" decoding="async">`,
		},
		{
			name: "self closing tag keeps its slash",
			in:   `<img src="1.png" /><img src="2.png" />`,
			want: `<img src="1.png" /><img src="2.png" loading="lazy" decoding="async" />`,
		},
		{
			name: "existing attributes are not duplicated",
			in:   `<img src="1.png"><img src="2.png" loading="eager"><img src="3.png" DECODING="sync">`,
			want: `<img src="1.png"><img src="2.png" loading="eager" decoding="async"><img src="3.png" DECODING="sync" loading="lazy">`,
		},
		{
			name: "single image is untouched",
			in:   `<img src="1.png">`,
			want: `<img src="1.png">`,
		},
		{
			name: "no images",
			in:   `<p>nothing here</p>`,
			want: `<p>nothing here</p>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeferOffscreenImages(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
