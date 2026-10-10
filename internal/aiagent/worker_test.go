package aiagent

import (
	"reflect"
	"strings"
	"testing"

	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/aiagent/models"
)

func TestSplitSuggestions(t *testing.T) {
	body, suggestions := splitSuggestions("Which plan?\n[[suggestions]] [\"Basic\",\"Pro\",\"Enterprise\",\"Extra\"]")
	if body != "Which plan?" {
		t.Fatalf("unexpected body: %q", body)
	}
	want := []string{"Basic", "Pro", "Enterprise"}
	if !reflect.DeepEqual(suggestions, want) {
		t.Fatalf("unexpected suggestions: %#v", suggestions)
	}
}

func TestSplitSuggestionsRejectsInvalidPayload(t *testing.T) {
	body, suggestions := splitSuggestions("Tell me more\n[[suggestions]] not-json")
	if body != "Tell me more" || suggestions != nil {
		t.Fatalf("unexpected result: %q %#v", body, suggestions)
	}
}

func TestSplitArticleSources(t *testing.T) {
	hits := []aimodels.SearchResult{
		{SourceID: 12, SourceType: aimodels.SourceHelpArticle},
		{SourceID: 34, SourceType: aimodels.SourceHelpArticle},
		{SourceID: 34, SourceType: aimodels.SourceHelpArticle},
		{SourceID: 56, SourceType: aimodels.SourceSnippet},
	}
	tests := []struct {
		name    string
		answer  string
		enabled bool
		body    string
		ids     []int
	}{
		{"used articles", "Answer.\n[[sources]] [34,12]", true, "Answer.", []int{34, 12}},
		{"duplicates and unknown IDs", "Answer.\n[[sources]] [12,12,0,-1,999,56,34]", true, "Answer.", []int{12, 34}},
		{"multiple searches and lines", "Answer.\n[[sources]] [12]\n[[sources]] [12,34]", true, "Answer.", []int{12, 34}},
		{"disabled", "Answer.\n[[sources]] [12]", false, "Answer.", nil},
		{"invalid payload", "Answer.\n[[sources]] [12,", true, "Answer.", nil},
		{"noninteger IDs", "Answer.\n[[sources]] [12,1.5]", true, "Answer.", nil},
		{"empty list", "Answer.\n[[sources]] []", true, "Answer.", nil},
		{"null", "Answer.\n[[sources]] null", true, "Answer.", nil},
		{"no marker", "Hello!", true, "Hello!", nil},
		{"inline marker", "Answer. [[sources]] [12]", true, "Answer.", []int{12}},
		{"next line array", "Answer.\n[[sources]]\n[12,34]", true, "Answer.", []int{12, 34}},
		{"multiline array", "Answer.\n[[sources]] [\n12,\n34\n]\nFollowup.", true, "Answer.\nFollowup.", []int{12, 34}},
		{"same line confirmation", "Answer.\n[[sources]] [12] [[confirm]] Did that help?", true, "Answer.\n[[confirm]] Did that help?", []int{12}},
		{"same line suggestions", "Answer.\n[[sources]] [12] [[suggestions]] [\"Yes\",\"No\"]", true, "Answer.\n[[suggestions]] [\"Yes\",\"No\"]", []int{12}},
		{"invalid array with confirmation", "Answer.\n[[sources]] [12, [[confirm]] Did that help?", true, "Answer.\n[[confirm]] Did that help?", nil},
		{"invalid payload with suggestions", "Answer.\n[[sources]] garbage [[suggestions]] [\"Yes\",\"No\"]", true, "Answer.\n[[suggestions]] [\"Yes\",\"No\"]", nil},
		{"noninteger multiline array", "Answer.\n[[sources]]\n[12,1.5]\n[[confirm]] Did that help?", true, "Answer.\n[[confirm]] Did that help?", nil},
		{"multiple markers on one line", "Answer.\n[[sources]] [12] [[sources]] [12,34] [[confirm]] Did that help?", true, "Answer.\n[[confirm]] Did that help?", []int{12, 34}},
		{"disabled multiline", "Answer.\n[[sources]]\n[12,34] [[confirm]] Did that help?", false, "Answer.\n[[confirm]] Did that help?", nil},
		{"with confirmation", "Answer.\n[[sources]] [12]\n[[confirm]]\nDid that help?\n[[suggestions]] [\"Yes\",\"No\"]", true, "Answer.\n[[confirm]]\nDid that help?\n[[suggestions]] [\"Yes\",\"No\"]", []int{12}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, ids := splitArticleSources(tt.answer, hits, tt.enabled)
			if body != tt.body || !reflect.DeepEqual(ids, tt.ids) {
				t.Fatalf("got %q, %v, want %q, %v", body, ids, tt.body, tt.ids)
			}
		})
	}
}

func TestArticleCitationPrompt(t *testing.T) {
	if strings.Contains(buildSystemPrompt(models.Assistant{}), "[[sources]]") {
		t.Fatal("disabled assistant requests article references")
	}
	if !strings.Contains(buildSystemPrompt(models.Assistant{CitationsEnabled: true}), "[[sources]]") {
		t.Fatal("enabled assistant does not request article references")
	}
}
