package stringutil

import (
	"regexp"
	"strings"
)

// imgAttrPrefix skips over complete attributes so the name can't match inside another
// attribute's name (data-loading) or a quoted value (alt="loading=x").
const imgAttrPrefix = `(?is)^<img(?:\s+[^\s=>]+(?:\s*=\s*(?:"[^"]*"|'[^']*'|[^\s"'>]*))?)*\s+`

var (
	imgTagRe = regexp.MustCompile(`(?is)<img\b(?:"[^"]*"|'[^']*'|[^>"'])*>`)

	imgLoadingAttrRe = regexp.MustCompile(imgAttrPrefix + `loading\s*=`)

	imgDecodingAttrRe = regexp.MustCompile(imgAttrPrefix + `decoding\s*=`)
)

// DeferOffscreenImages adds loading="lazy" and decoding="async" to every <img> tag
// except the first, which is left eager since it is the likely LCP element.
func DeferOffscreenImages(html string) string {
	n := 0
	return imgTagRe.ReplaceAllStringFunc(html, func(tag string) string {
		n++
		if n == 1 {
			return tag
		}
		var attrs strings.Builder
		if !imgLoadingAttrRe.MatchString(tag) {
			attrs.WriteString(` loading="lazy"`)
		}
		if !imgDecodingAttrRe.MatchString(tag) {
			attrs.WriteString(` decoding="async"`)
		}
		if attrs.Len() == 0 {
			return tag
		}
		closing := ">"
		body := strings.TrimSuffix(tag, ">")
		if trimmed, ok := strings.CutSuffix(body, "/"); ok {
			body = trimmed
			closing = " />"
		}
		return strings.TrimRight(body, " \t\r\n") + attrs.String() + closing
	})
}
