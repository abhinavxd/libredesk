package helpcenter

import (
	"slices"
	"strings"

	"github.com/abhinavxd/libredesk/internal/helpcenter/models"
)

// supportedLocales are the locale codes a help center may be authored in, keyed to their English name.
var supportedLocales = map[string]string{
	"ar":      "Arabic",
	"bg":      "Bulgarian",
	"bn":      "Bengali",
	"bs":      "Bosnian",
	"ca":      "Catalan",
	"cs":      "Czech",
	"da":      "Danish",
	"de":      "German",
	"de-AT":   "German (Austria)",
	"de-CH":   "German (Switzerland)",
	"el":      "Greek",
	"en":      "English",
	"en-AU":   "English (Australia)",
	"en-CA":   "English (Canada)",
	"en-GB":   "English (UK)",
	"en-IN":   "English (India)",
	"en-US":   "English (USA)",
	"es":      "Spanish",
	"es-MX":   "Spanish (Mexico)",
	"et":      "Estonian",
	"eu":      "Basque",
	"fa":      "Persian",
	"fi":      "Finnish",
	"fil":     "Filipino",
	"fr":      "French",
	"fr-BE":   "French (Belgium)",
	"fr-CA":   "French (Canada)",
	"fr-CH":   "French (Switzerland)",
	"ga":      "Irish",
	"he":      "Hebrew",
	"hi":      "Hindi",
	"hr":      "Croatian",
	"hu":      "Hungarian",
	"hy":      "Armenian",
	"id":      "Indonesian",
	"is":      "Icelandic",
	"it":      "Italian",
	"ja":      "Japanese",
	"ko":      "Korean",
	"lt":      "Lithuanian",
	"lv":      "Latvian",
	"mn":      "Mongolian",
	"mr":      "Marathi",
	"ms":      "Malay",
	"ne":      "Nepali",
	"nl":      "Dutch",
	"nl-BE":   "Dutch (Belgium)",
	"no":      "Norwegian",
	"pl":      "Polish",
	"pt":      "Portuguese",
	"pt-BR":   "Portuguese (Brazil)",
	"ro":      "Romanian",
	"ru":      "Russian",
	"sk":      "Slovak",
	"sl":      "Slovenian",
	"sr":      "Serbian",
	"sv":      "Swedish",
	"sw":      "Swahili",
	"ta":      "Tamil",
	"th":      "Thai",
	"tr":      "Turkish",
	"uk":      "Ukrainian",
	"uz":      "Uzbek",
	"vi":      "Vietnamese",
	"yi":      "Yiddish",
	"zh-Hans": "Chinese (Simplified)",
	"zh-Hant": "Chinese (Traditional)",
}

// SupportedLocales returns the selectable help center locales, sorted by English name.
func SupportedLocales() []models.Locale {
	locales := make([]models.Locale, 0, len(supportedLocales))
	for code, name := range supportedLocales {
		locales = append(locales, models.Locale{Code: code, Name: name})
	}
	slices.SortFunc(locales, func(a, b models.Locale) int {
		return strings.Compare(a.Name, b.Name)
	})
	return locales
}
