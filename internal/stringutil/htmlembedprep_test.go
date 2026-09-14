package stringutil

import (
	"strings"
	"testing"
)

func TestPrepareHTMLForEmbedding(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		contains    []string
		notContains []string
	}{
		{
			name:        "link becomes markdown",
			in:          `<p>See <a href="https://example.com/guide">the guide</a> for steps.</p>`,
			contains:    []string{"See [the guide](https://example.com/guide) for steps."},
			notContains: []string{"<a "},
		},
		{
			name:     "bare url link keeps the url once",
			in:       `<p><a href="https://example.com">https://example.com</a></p>`,
			contains: []string{">https://example.com<"},
			notContains: []string{
				"[https://example.com](https://example.com)",
				"<a ",
			},
		},
		{
			name:        "anchor and cid links are left alone",
			in:          `<p><a href="#setup">Setup</a> <a href="cid:img1">inline</a></p>`,
			contains:    []string{`<a href="#setup">Setup</a>`, `<a href="cid:img1">inline</a>`},
			notContains: []string{"[Setup]"},
		},
		{
			name: "table flattens to header labelled lines",
			in: `<table>
				<tr><th>Plan</th><th>Price</th></tr>
				<tr><td>Basic</td><td>$5</td></tr>
				<tr><td>Pro</td><td>$10</td></tr>
			</table>`,
			contains:    []string{"<pre>", "Plan: Basic | Price: $5", "Plan: Pro | Price: $10"},
			notContains: []string{"<table>"},
		},
		{
			name:        "header only table keeps the header line",
			in:          `<table><tr><th>Plan</th><th>Price</th></tr></table>`,
			contains:    []string{"Plan | Price"},
			notContains: []string{"<table>"},
		},
		{
			name: "caption leads the table text",
			in:   `<table><caption>Pricing</caption><tr><td>Basic</td><td>$5</td></tr></table>`,
			contains: []string{
				"Pricing\nBasic | $5",
			},
		},
		{
			name:        "link inside a table cell survives as markdown",
			in:          `<table><tr><th>Doc</th></tr><tr><td><a href="https://example.com/x">Pay now</a></td></tr></table>`,
			contains:    []string{"Doc: [Pay now](https://example.com/x)"},
			notContains: []string{"<a "},
		},
		{
			name:     "invalid html is returned untouched on parse success path",
			in:       `plain text with no markup`,
			contains: []string{"plain text with no markup"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prepareHTMLForEmbedding(tt.in)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\ngot: %s", want, got)
				}
			}
			for _, not := range tt.notContains {
				if strings.Contains(got, not) {
					t.Errorf("output should not contain %q\ngot: %s", not, got)
				}
			}
		})
	}
}
