package aiagent

import (
	"reflect"
	"strings"
	"testing"

	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/aiagent/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
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
		{"inline claims", "Use JWT.[[cite:12]] Log out.[[cite:34]]", true, "Use JWT.[[cite:12]] Log out.[[cite:34]]", []int{12, 34}},
		{"repeated inline article", "Use JWT.[[cite:34]] More JWT.[[cite:34]]", true, "Use JWT.[[cite:34]] More JWT.[[cite:34]]", []int{34}},
		{"invalid inline articles", "Answer.[[cite:999]][[cite:56]][[cite:0]][[cite:-1]][[cite:no]]", true, "Answer.", nil},
		{"inline disabled", "Answer.[[cite:12]]", false, "Answer.", nil},
		{"mixed citation styles", "Answer.[[cite:34]]\n[[sources]] [12,34]", true, "Answer.[[cite:34]]", []int{34, 12}},
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
	if strings.Contains(buildSystemPrompt(models.Assistant{}), "[[cite:") {
		t.Fatal("disabled assistant requests article references")
	}
	if !strings.Contains(buildSystemPrompt(models.Assistant{CitationsEnabled: true}), "[[cite:") {
		t.Fatal("enabled assistant does not request article references")
	}
}

func TestArticleCitationHTML(t *testing.T) {
	text := "Use **JWT**.[[cite:12]]\nDéconnectez-vous.[[cite:34]][[cite:999]]"
	content := articleCitationHTML(text, []int{12, 34})
	want := "<p>Use <strong>JWT</strong>.<!--ld-cite:12--><br>\nDéconnectez-vous.<!--ld-cite:34--></p>\n"
	if content != want {
		t.Fatalf("got %q, want %q", content, want)
	}
	plain := stringutil.HTML2Text(content)
	if strings.Contains(plain, "cite") || !strings.Contains(plain, "Déconnectez-vous.") {
		t.Fatalf("citation markers changed stored plain text: %q", plain)
	}
	if got := articleCitationHTML("Answer.[[cite:12]]", nil /* articleIDs */); got != "<p>Answer.</p>\n" {
		t.Fatalf("uncited reply retained a marker: %q", got)
	}
	link := articleCitationHTML(`[Article](https://example.com "[[cite:12]]")`, []int{12})
	if strings.Contains(link, "ld-cite") || strings.Contains(link, "[[cite:") {
		t.Fatalf("citation inserted inside an HTML attribute: %q", link)
	}
	if got := numberedCitationText("JWT.[[cite:12]] Logout.[[cite:34]][[cite:999]]", []int{12, 34}); got != "JWT.(1) Logout.(2)" {
		t.Fatalf("unexpected preview: %q", got)
	}
}
