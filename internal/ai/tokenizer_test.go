package ai

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// The old chunker estimated tokens as runes*0.4, which badly under-counts CJK text and let
// oversized chunks reach the 8192-token embedding limit. Verify the tiktoken counter reflects
// reality: CJK counts far higher per rune than the old heuristic assumed.
func TestTokenCounterCatchesCJKUndercount(t *testing.T) {
	count := newTokenCounter(nil)

	cjk := strings.Repeat("这是一个测试知识库的中文句子。", 200)
	oldEstimate := len([]rune(cjk)) * 2 / 5
	real := count(cjk)

	if real <= oldEstimate {
		t.Fatalf("expected tiktoken count (%d) to exceed the old rune estimate (%d) for CJK", real, oldEstimate)
	}

	english := "The quick brown fox jumps over the lazy dog."
	if got := count(english); got == 0 || got > len(english) {
		t.Fatalf("english token count %d looks wrong for %q", got, english)
	}
}

func TestCapToTokens(t *testing.T) {
	initEncoder(nil)

	small := "short text that is well under the limit"
	if got := capToTokens(small, 8000); got != small {
		t.Fatalf("small input should pass through unchanged, got %q", got)
	}

	big := strings.Repeat("这是一个很长的中文段落。", 3000)
	capped := capToTokens(big, 8000)
	if countTokens(capped) > 8000 {
		t.Fatalf("capped input still exceeds limit: %d tokens", countTokens(capped))
	}
	if !utf8.ValidString(capped) {
		t.Fatal("capped input is not valid UTF-8")
	}
	if len(capped) >= len(big) {
		t.Fatal("oversized input should have been truncated")
	}
}
