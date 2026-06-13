package whatsapp

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// placeholderPattern matches {{1}} / {{name}} placeholders inside template text.
var placeholderPattern = regexp.MustCompile(`\{\{([A-Za-z0-9_]+)\}\}`)

// TemplateSendParts is the runtime context for a template send; Params is keyed by placeholder name, with button_url_<i> reserved for URL button parameters.
type TemplateSendParts struct {
	HeaderType    string
	HeaderContent string
	HeaderMediaID string
	BodyContent   string
	Buttons       []TemplateButton
	Params        map[string]string
}

// BuildSendComponents returns the components array for a template send; components without parameters are omitted so Meta uses the approved text.
func BuildSendComponents(p TemplateSendParts) []map[string]any {
	var out []map[string]any

	if h := buildHeaderComponent(p); h != nil {
		out = append(out, h)
	}

	if body := buildBodyComponent(p.BodyContent, p.Params); body != nil {
		out = append(out, body)
	}

	for i, b := range p.Buttons {
		if c := buildButtonComponent(i, b, p.Params); c != nil {
			out = append(out, c)
		}
	}

	return out
}

// OrderedPlaceholders returns the distinct {{...}} names in text; all-numeric sets sort ascending to match Meta's positional mapping.
func OrderedPlaceholders(text string) []string {
	if text == "" {
		return nil
	}
	matches := placeholderPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	keys := make([]string, 0, len(matches))
	allNumeric := true
	for _, m := range matches {
		key := m[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
		if _, err := strconv.Atoi(key); err != nil {
			allNumeric = false
		}
	}
	if allNumeric {
		sort.Slice(keys, func(i, j int) bool {
			a, _ := strconv.Atoi(keys[i])
			b, _ := strconv.Atoi(keys[j])
			return a < b
		})
	}
	return keys
}

func buildHeaderComponent(p TemplateSendParts) map[string]any {
	headerType := strings.ToUpper(p.HeaderType)
	switch headerType {
	case "", "NONE":
		return nil
	case "TEXT":
		// Meta rejects parameters sent for a static header.
		params := positionalParams(p.HeaderContent, p.Params)
		if len(params) == 0 {
			return nil
		}
		return map[string]any{
			"type":       "header",
			"parameters": params,
		}
	case "IMAGE", "VIDEO", "DOCUMENT":
		if p.HeaderMediaID == "" {
			return nil
		}
		mediaKey := strings.ToLower(headerType)
		return map[string]any{
			"type": "header",
			"parameters": []map[string]any{
				{"type": mediaKey, mediaKey: map[string]any{"id": p.HeaderMediaID}},
			},
		}
	}
	return nil
}

func buildBodyComponent(bodyContent string, params map[string]string) map[string]any {
	parameters := positionalParams(bodyContent, params)
	if len(parameters) == 0 {
		return nil
	}
	return map[string]any{
		"type":       "body",
		"parameters": parameters,
	}
}

func buildButtonComponent(index int, b TemplateButton, params map[string]string) map[string]any {
	if strings.ToUpper(b.Type) != "URL" {
		return nil
	}
	key := "button_url_" + strconv.Itoa(index)
	val, ok := params[key]
	if !ok || val == "" {
		return nil
	}
	return map[string]any{
		"type":     "button",
		"sub_type": "url",
		"index":    strconv.Itoa(index),
		"parameters": []map[string]any{
			{"type": "text", "text": val},
		},
	}
}

// positionalParams returns parameter entries for text's placeholders: numeric ones ascending (Meta maps positionally), named ones with parameter_name set.
func positionalParams(text string, params map[string]string) []map[string]any {
	keys := OrderedPlaceholders(text)
	if len(keys) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		entry := map[string]any{"type": "text", "text": params[key]}
		if _, err := strconv.Atoi(key); err != nil {
			entry["parameter_name"] = key
		}
		out = append(out, entry)
	}
	return out
}
