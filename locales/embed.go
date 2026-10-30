// Package locales embeds the Gettext catalogs shared by backend builds.
package locales

import "embed"

// LocaleFiles contains Go PO/MO catalogs and compiled Lingui JSON catalogs.
// The English frontend JSON is generated from source; no English PO is needed.
//
//go:embed */LC_MESSAGES
var LocaleFiles embed.FS
