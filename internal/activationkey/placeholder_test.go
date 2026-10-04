package activationkey

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

const chip = `<span data-type="activation-key" data-id="458219512" data-app-id="3">&lt;KEY_ID_458219512&gt;</span>`

func TestParsePlaceholdersFindsComposerChips(t *testing.T) {
	content := `<p>Hi! Here is your key: ` + chip + `</p><p>And another: <span data-app-id="3" data-id="100000001" data-type="activation-key">&lt;KEY_ID_100000001&gt;</span></p>`
	got, err := ParsePlaceholders(content)
	if err != nil {
		t.Fatal(err)
	}
	want := []Placeholder{{ID: "458219512", AppID: 3}, {ID: "100000001", AppID: 3}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParsePlaceholdersIgnoresLookalikeText(t *testing.T) {
	for _, content := range []string{
		`<p>&lt;KEY_ID_458219512&gt;</p>`,
		`<p>&lt;span data-type="activation-key" data-id="458219512" data-app-id="3"&gt;x&lt;/span&gt;</p>`,
		`<p data-label="activation-key">plain</p>`,
		`<!-- <span data-type="activation-key" data-id="458219512" data-app-id="3">x</span> -->`,
		`<script><span data-type="activation-key" data-id="458219512" data-app-id="3">x</span></script>`,
	} {
		got, err := ParsePlaceholders(content)
		if err != nil || len(got) != 0 {
			t.Errorf("%s: got %+v, %v", content, got, err)
		}
	}
}

func TestParsePlaceholdersRejectsMalformedChips(t *testing.T) {
	for name, content := range map[string]string{
		"duplicate id":   chip + chip,
		"not a span":     `<div data-type="activation-key" data-id="458219512" data-app-id="3">x</div>`,
		"self closing":   `<span data-type="activation-key" data-id="458219512" data-app-id="3"/>`,
		"unclosed":       `<p><span data-type="activation-key" data-id="458219512" data-app-id="3">x</p>`,
		"nested markup":  `<span data-type="activation-key" data-id="458219512" data-app-id="3"><b>x</b></span>`,
		"nested chip":    `<span data-type="activation-key" data-id="458219512" data-app-id="3">` + chip + `</span>`,
		"comment inside": `<span data-type="activation-key" data-id="458219512" data-app-id="3"><!-- x --></span>`,
		"bad id":         `<span data-type="activation-key" data-id="12ab" data-app-id="3">x</span>`,
		"short id":       `<span data-type="activation-key" data-id="123" data-app-id="3">x</span>`,
		"missing app":    `<span data-type="activation-key" data-id="458219512">x</span>`,
		"zero app":       `<span data-type="activation-key" data-id="458219512" data-app-id="0">x</span>`,
		"case variant":   `<SPAN DATA-TYPE="Activation-Key" data-id="458219512">x</SPAN>`,
	} {
		if _, err := ParsePlaceholders(content); !errors.Is(err, ErrInvalidPlaceholder) {
			t.Errorf("%s: expected ErrInvalidPlaceholder, got %v", name, err)
		}
	}
}

func TestParsePlaceholdersUsesFirstOfRepeatedAttributes(t *testing.T) {
	got, err := ParsePlaceholders(`<span data-type="activation-key" data-id="458219512" data-id="999999999" data-app-id="3">x</span>`)
	if err != nil || len(got) != 1 || got[0].ID != "458219512" {
		t.Fatalf("got %+v, %v", got, err)
	}
	// A decoy first data-type means the element is not a placeholder at all.
	got, err = ParsePlaceholders(`<span data-type="mention" data-type="activation-key" data-id="458219512" data-app-id="3">x</span>`)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestReplacePlaceholdersKeepsEverythingElseByteForByte(t *testing.T) {
	before := `<p class="x" style="COLOR: Red">Hello&nbsp;<b>there</b> </p>`
	after := `<p>Thanks &amp; enjoy!<br/><img src="cid:ldsk-abc" ALT="Logo"></p>`
	out, err := ReplacePlaceholders(before+chip+after, func(p Placeholder) string { return "[" + p.ID + "]" })
	if err != nil {
		t.Fatal(err)
	}
	if want := before + "[458219512]" + after; out != want {
		t.Fatalf("got  %q\nwant %q", out, want)
	}
}

func TestReplacePlaceholdersLeavesLookalikeTextAlone(t *testing.T) {
	content := `<p>&lt;KEY_ID_458219512&gt; ` + chip + `</p>`
	out, err := ReplacePlaceholders(content, func(Placeholder) string { return "KEY" })
	if err != nil {
		t.Fatal(err)
	}
	if out != `<p>&lt;KEY_ID_458219512&gt; KEY</p>` {
		t.Fatalf("got %q", out)
	}
}

func TestNormalizeKey(t *testing.T) {
	for raw, ok := range map[string]bool{
		"ABCDE-FGHIJ-KLMNO":       true,
		"  ABCDE-FGHIJ-KLMNO\n":   true,
		"abcd1234":                true,
		"ABCDE-FGHIJ-KLMNO-PQRST": true,
		"short":                   false,
		"-ABCDE-FGHIJ":            false,
		"ABCDE-FGHIJ-":            false,
		"ABCDE FGHIJ":             false,
		"ABCDE<b>FGHIJ":           false,
		"{{ABCDE-FGHIJ}}":         false,
		"enc:ABCDEFGHIJ":          false,
		strings.Repeat("A", 101):  false,
	} {
		if _, got := NormalizeKey(raw); got != ok {
			t.Errorf("NormalizeKey(%q) = %v, want %v", raw, got, ok)
		}
	}
}

func TestMaskRunsHidesWholeKeysOnly(t *testing.T) {
	sent := func(key string) bool { return strings.EqualFold(key, "ABCDE-FGHIJ-KLMNO") }
	content := `<blockquote>Your key: <strong>ABCDE-FGHIJ-KLMNO</strong></blockquote><p>is abcde-fghij-klmno valid? ` +
		`--ABCDE-FGHIJ-KLMNO-- ABCDE-FGHIJ-KLMNP is not mine, nor XABCDE-FGHIJ-KLMNO</p>`
	got := maskRuns(content, sent)
	if strings.Count(got, MaskedKey) != 3 || !strings.Contains(got, "--"+MaskedKey+"--") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "ABCDE-FGHIJ-KLMNP") || !strings.Contains(got, "XABCDE-FGHIJ-KLMNO") {
		t.Fatalf("masked something that isn't the key: %q", got)
	}
	if runs := keyRuns(content); len(runs) != 7 || runs[0] != "blockquote" {
		t.Fatalf("keyRuns = %q", runs)
	}
	if maskRuns("nothing here", sent) != "nothing here" {
		t.Fatal("mask without keys changed content")
	}
}
