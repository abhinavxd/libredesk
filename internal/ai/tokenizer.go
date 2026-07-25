package ai

import (
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/pkoukk/tiktoken-go"
	tiktokenloader "github.com/pkoukk/tiktoken-go-loader"
	"github.com/zerodha/logf"
)

// All OpenAI embedding models (text-embedding-3-*, ada-002) tokenize with cl100k_base.
const embeddingEncoding = "cl100k_base"

var (
	encoderOnce sync.Once
	encoder     *tiktoken.Tiktoken
)

// initEncoder loads tiktoken's BPE vocab from the binary (no network fetch); on failure encoder stays nil and callers fall back to a rune estimate.
func initEncoder(lo *logf.Logger) {
	encoderOnce.Do(func() {
		tiktoken.SetBpeLoader(tiktokenloader.NewOfflineLoader())
		enc, err := tiktoken.GetEncoding(embeddingEncoding)
		if err != nil {
			if lo != nil {
				lo.Error("could not load tiktoken encoding, falling back to rune-based token estimates", "error", err)
			}
			return
		}
		encoder = enc
	})
}

// newTokenCounter returns the chunker's token-counting function.
func newTokenCounter(lo *logf.Logger) func(string) int {
	initEncoder(lo)
	return countTokens
}

func countTokens(s string) int {
	if encoder == nil {
		return len([]rune(s)) * 2 / 5
	}
	return len(encoder.Encode(s, nil, nil))
}

// capToTokens truncates s to at most maxTokens, byte-capping on a rune boundary when the encoder is unavailable.
func capToTokens(s string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	if encoder == nil {
		if len(s) <= maxTokens {
			return s
		}
		return trimToRuneBoundary(s, maxTokens)
	}
	toks := encoder.Encode(s, nil, nil)
	if len(toks) <= maxTokens {
		return s
	}
	return strings.ToValidUTF8(encoder.Decode(toks[:maxTokens]), "")
}

func trimToRuneBoundary(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
