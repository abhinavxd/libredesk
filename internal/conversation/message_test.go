package conversation

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

const testUUID = "d0355103-455f-4c7d-b9c7-86e9254fe119"
const testUUID2 = "edb7be78-ef7d-4fe9-888b-22494f0ce076"

type fromTestInbox struct {
	inbox.EmailInbox
	from    string
	primary string
}

func (f fromTestInbox) FromAddress() string      { return f.from }
func (f fromTestInbox) PrimaryAddress() string   { return f.primary }
func (f fromTestInbox) FromNameTemplate() string { return "" }

type citationArticleStore struct {
	references []hcmodels.ArticleReference
	err        error
	lookups    *int
}

type citationSettingsStore struct {
	settingsStore
	rootURL string
	lookups *int
}

func (s citationArticleStore) GetArticleReferences(ids []int) ([]hcmodels.ArticleReference, error) {
	if s.lookups != nil {
		(*s.lookups)++
	}
	return s.references, s.err
}

func (s citationSettingsStore) GetAppRootURL() (string, error) {
	if s.lookups != nil {
		(*s.lookups)++
	}
	return s.rootURL, nil
}

func TestImgSrcUploadsPattern(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCount int
		wantUUIDs []string
	}{
		// Happy paths.
		{
			name:      "relative_url",
			body:      `<img src="/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "absolute_url",
			body:      `<img src="https://libredesk.example.com/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "absolute_url_with_port",
			body:      `<img src="http://localhost:9000/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "with_query_string",
			body:      `<img src="/uploads/` + testUUID + `?sig=abc&exp=123">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "with_html_entity_query",
			body:      `<img src="/uploads/` + testUUID + `?sig=abc&amp;exp=123">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "single_quotes",
			body:      `<img src='/uploads/` + testUUID + `'>`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "attrs_before_src",
			body:      `<img class="inline-image" alt="x" data-foo="y" src="/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "attrs_after_src",
			body:      `<img src="/uploads/` + testUUID + `" class="x" alt="y">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "xhtml_self_closing",
			body:      `<img src="/uploads/` + testUUID + `" />`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "uppercase_img_tag",
			body:      `<IMG SRC="/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "multiline_tag",
			body:      "<img\n  alt=\"x\"\n  src=\"/uploads/" + testUUID + "\"\n>",
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name: "multiple_in_body",
			body: `hello <img src="/uploads/` + testUUID + `"> world ` +
				`<img src="/uploads/` + testUUID2 + `">`,
			wantCount: 2,
			wantUUIDs: []string{testUUID, testUUID2},
		},

		{
			name:      "uppercase_hex_uuid_matches",
			body:      `<img src="/uploads/D0355103-455F-4C7D-B9C7-86E9254FE119">`,
			wantCount: 1,
			wantUUIDs: []string{"D0355103-455F-4C7D-B9C7-86E9254FE119"},
		},
		// `\b` boundary lets data-src match; harmless, no real src to render.
		{
			name:      "quirk_data_src_attribute_matches",
			body:      `<img alt="x" data-src="/uploads/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		// Not context-aware: comments are matched too.
		{
			name:      "quirk_inside_html_comment_matches",
			body:      `<!-- <img src="/uploads/` + testUUID + `"> -->`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},

		// Non-matches.
		{
			name:      "anchor_href_no_match",
			body:      `<a href="/uploads/` + testUUID + `">link</a>`,
			wantCount: 0,
		},
		{
			name:      "picture_source_no_match",
			body:      `<picture><source srcset="/uploads/` + testUUID + `"></picture>`,
			wantCount: 0,
		},
		{
			name:      "malformed_uuid_no_match",
			body:      `<img src="/uploads/not-a-uuid">`,
			wantCount: 0,
		},
		{
			name:      "uploads_filename_no_uuid_no_match",
			body:      `<img src="/uploads/photo.png">`,
			wantCount: 0,
		},
		{
			name:      "uuid_too_short_no_match",
			body:      `<img src="/uploads/abcdef01-2345-6789-abcd-ef0123">`,
			wantCount: 0,
		},
		{
			name:      "empty_src_no_match",
			body:      `<img src="">`,
			wantCount: 0,
		},

		{
			name:      "trailing_path_segment_still_matches",
			body:      `<img src="/uploads/` + testUUID + `/extra">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "s3_path_style_public_url",
			body:      `<img src="https://s3.ap-south-1.amazonaws.com/bucket-name/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "s3_virtual_hosted_url",
			body:      `<img src="https://bucket-name.s3.ap-south-1.amazonaws.com/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "s3_presigned_url_full",
			body:      `<img class="inline-image" src="https://s3.ap-south-1.amazonaws.com/bucket-name/` + testUUID + `?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=ABC%2F20260514%2Fap-south-1%2Fs3%2Faws4_request&X-Amz-Date=20260514T161618Z&X-Amz-Expires=300&X-Amz-SignedHeaders=host&X-Amz-Signature=deadbeef">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "s3_presigned_url_html_entities",
			body:      `<img src="https://s3.ap-south-1.amazonaws.com/bucket/` + testUUID + `?X-Amz-Algorithm=AWS4-HMAC-SHA256&amp;X-Amz-Signature=deadbeef&amp;X-Amz-Expires=300">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "s3_nested_bucket_path",
			body:      `<img src="https://s3.ap-south-1.amazonaws.com/bucket/childpath1/childpath2/` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "s3_nested_bucket_path_presigned",
			body:      `<img src="https://s3.ap-south-1.amazonaws.com/bucket/2026/05/14/` + testUUID + `?X-Amz-Signature=abc">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "s3_compatible_endpoint",
			body:      `<img src="https://minio.example.com:9000/bucket/` + testUUID + `?X-Amz-Signature=abc">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name: "multiple_s3_presigned_urls",
			body: `<img src="https://s3.amazonaws.com/b/` + testUUID + `?X-Amz-Signature=a">` +
				`<img src="https://s3.amazonaws.com/b/` + testUUID2 + `?X-Amz-Signature=b">`,
			wantCount: 2,
			wantUUIDs: []string{testUUID, testUUID2},
		},
		{
			name:      "cdn_proxied_url",
			body:      `<img src="https://cdn.example.com/media/` + testUUID + `/photo.png">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "uuid_in_query_param",
			body:      `<img src="https://example.com/render?id=` + testUUID + `">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "cid_form_skipped",
			body:      `<img src="cid:ldsk-` + testUUID + `">`,
			wantCount: 0,
		},
		{
			name: "s3_presigned_realistic_long_url",
			body: `<img class="inline-image" style="max-width: 100%; height: auto;" src="https://s3.ap-south-1.amazonaws.com/example-bucket/` + testUUID +
				`?X-Amz-Algorithm=AWS4-HMAC-SHA256&amp;X-Amz-Credential=ASIAEXAMPLE%2F20260514%2Fap-south-1%2Fs3%2Faws4_request` +
				`&amp;X-Amz-Date=20260514T161618Z&amp;X-Amz-Expires=300` +
				`&amp;X-Amz-Security-Token=IQoJb3JpZ2luX2VjEJj%2F%2F%2F%2F%2F%2F%2F%2F%2F%2FwEaCmFwLXNvdXRoLTEiSDBGAiEA21MRBCy0mE3AzOx9` +
				`&amp;X-Amz-SignedHeaders=host&amp;response-content-disposition=inline%3B%20filename%3D%22image.png%22` +
				`&amp;X-Amz-Signature=1ba1c7feb9ba2dd3c7df72e3054dd9bb32aaced52adc48d672f926c4b95e3115">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "fs_store_signed_url",
			body:      `<img src="https://libredesk.example.com/uploads/` + testUUID + `?sig=deadbeefcafe&exp=1768435200">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "fs_store_signed_url_html_entity",
			body:      `<img src="https://libredesk.example.com/uploads/` + testUUID + `?sig=deadbeefcafe&amp;exp=1768435200">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
		{
			name:      "fs_store_custom_port",
			body:      `<img src="http://localhost:9000/uploads/` + testUUID + `?sig=abc&exp=1">`,
			wantCount: 1,
			wantUUIDs: []string{testUUID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInlineImageUUIDs(tt.body)
			if len(got) != tt.wantCount {
				t.Fatalf("got %d uuids, want %d (got=%v)", len(got), tt.wantCount, got)
			}
			for i, want := range tt.wantUUIDs {
				if !strings.EqualFold(got[i], want) {
					t.Errorf("uuid %d = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestImgSrcUploadsPattern_Adversarial(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCount int
	}{
		{
			name:      "image_element_should_not_match",
			body:      `<image src="/uploads/` + testUUID + `">`,
			wantCount: 0,
		},
		{
			name:      "imgblah_tag_should_not_match",
			body:      `<imgblah src="/uploads/` + testUUID + `">`,
			wantCount: 0,
		},
		{
			name:      "src_keyword_inside_alt_value_should_not_match",
			body:      `<img alt="see src=/uploads/foo" data-foo="bar">`,
			wantCount: 0,
		},
		{
			name:      "input_element_should_not_match",
			body:      `<input src="/uploads/` + testUUID + `">`,
			wantCount: 0,
		},
		{
			name:      "multiline_img_src_should_match",
			body:      "<img\n\tsrc=\"/uploads/" + testUUID + "\"\n>",
			wantCount: 1,
		},
		{
			name:      "extra_trailing_attributes_should_match",
			body:      `<img src="/uploads/` + testUUID + `" width="100" height="50" loading="lazy">`,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInlineImageUUIDs(tt.body)
			if len(got) != tt.wantCount {
				t.Errorf("got %d uuids, want %d\nbody: %s\nuuids: %v",
					len(got), tt.wantCount, tt.body, got)
			}
		})
	}
}

func TestExtractInlineImageUUIDs(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "empty_body",
			body: "",
			want: []string{},
		},
		{
			name: "no_images",
			body: "Just some text, no images here.",
			want: []string{},
		},
		{
			name: "single_image",
			body: `<img src="/uploads/` + testUUID + `">`,
			want: []string{testUUID},
		},
		{
			name: "two_distinct_images",
			body: `<img src="/uploads/` + testUUID + `"><img src="/uploads/` + testUUID2 + `">`,
			want: []string{testUUID, testUUID2},
		},
		{
			name: "duplicate_uuid_deduped",
			body: `<img src="/uploads/` + testUUID + `"><img src="/uploads/` + testUUID + `?v=2">`,
			want: []string{testUUID},
		},
		{
			name: "ignores_cid_form",
			body: `<img src="cid:ldsk-` + testUUID + `">`,
			want: []string{},
		},
		{
			name: "mixed_cid_and_s3_extracts_only_s3",
			body: `<img src="cid:ldsk-` + testUUID + `">` +
				`<img src="https://s3.amazonaws.com/b/` + testUUID2 + `?X-Amz-Signature=x">`,
			want: []string{testUUID2},
		},
		{
			name: "s3_presigned_url_extracts_uuid",
			body: `<img class="inline-image" src="https://s3.ap-south-1.amazonaws.com/example-bucket/` + testUUID + `?X-Amz-Algorithm=AWS4-HMAC-SHA256&amp;X-Amz-Signature=abc">`,
			want: []string{testUUID},
		},
		{
			name: "nested_bucket_path_extracts_uuid",
			body: `<img src="https://s3.amazonaws.com/bucket/2026/05/14/` + testUUID + `">`,
			want: []string{testUUID},
		},
		{
			name: "dedupes_across_s3_and_relative",
			body: `<img src="/uploads/` + testUUID + `">` +
				`<img src="https://s3.amazonaws.com/b/` + testUUID + `?X-Amz-Signature=x">`,
			want: []string{testUUID},
		},
		{
			name: "footer_image_ignored_inline_extracted",
			body: `<header><img src="https://static.example.com/brand/logo.png" alt="brand" /></header>` +
				`<p>Hello</p>` +
				`<img class="inline-image" src="https://s3.amazonaws.com/example-bucket/` + testUUID + `?X-Amz-Signature=abc">`,
			want: []string{testUUID},
		},
		{
			name: "fs_store_signed_url_extracts",
			body: `<img src="https://libredesk.example.com/uploads/` + testUUID + `?sig=deadbeef&amp;exp=1768435200">`,
			want: []string{testUUID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInlineImageUUIDs(tt.body)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractInlineContentIDs(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "empty_body",
			body: "",
			want: []string{},
		},
		{
			name: "ignores_non_cid_src",
			body: `<img src="/uploads/` + testUUID + `">`,
			want: []string{},
		},
		{
			name: "single_cid_extracted",
			body: `<img src="cid:ldsk-` + testUUID + `">`,
			want: []string{"ldsk-" + testUUID},
		},
		{
			name: "mixed_cid_and_url_returns_only_cids",
			body: `<img src="/uploads/` + testUUID + `"><img src="cid:ldsk-` + testUUID2 + `">`,
			want: []string{"ldsk-" + testUUID2},
		},
		{
			name: "single_quotes_around_src",
			body: `<img src='cid:ldsk-` + testUUID + `'>`,
			want: []string{"ldsk-" + testUUID},
		},
		{
			name: "empty_cid_skipped",
			body: `<img src="cid:">`,
			want: []string{},
		},
		{
			name: "multi_with_dedup_and_order",
			body: `<img src="cid:ldsk-` + testUUID2 + `"><img src="cid:ldsk-` + testUUID + `"><img src="cid:ldsk-` + testUUID2 + `">`,
			want: []string{"ldsk-" + testUUID2, "ldsk-" + testUUID},
		},
		{
			name: "src_after_other_attributes",
			body: `<img class="inline" alt="x" src="cid:ldsk-` + testUUID + `">`,
			want: []string{"ldsk-" + testUUID},
		},
		{
			name: "uppercase_cid_prefix_not_matched",
			body: `<img src="CID:ldsk-` + testUUID + `">`,
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractInlineContentIDs(tt.body)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRewriteInlineImagesToCID(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "empty_body",
			body: "",
			want: "",
		},
		{
			name: "no_change_when_no_uploads",
			body: `<p>hello world</p>`,
			want: `<p>hello world</p>`,
		},
		{
			name: "single_relative",
			body: `<img src="/uploads/` + testUUID + `">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "absolute_url_with_query",
			body: `<img src="https://host.example.com/uploads/` + testUUID + `?sig=abc&exp=1">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "s3_presigned_url_rewritten_to_cid",
			body: `<img class="inline-image" src="https://s3.ap-south-1.amazonaws.com/bucket/` + testUUID + `?X-Amz-Signature=abc">`,
			want: `<img class="inline-image" src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "s3_presigned_url_with_html_entities_rewritten",
			body: `<img src="https://s3.ap-south-1.amazonaws.com/bucket/` + testUUID + `?X-Amz-Algorithm=AWS4-HMAC-SHA256&amp;X-Amz-Signature=abc&amp;X-Amz-Expires=300">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "s3_nested_path_rewritten",
			body: `<img src="https://s3.amazonaws.com/bucket/childpath1/childpath2/` + testUUID + `?X-Amz-Signature=abc">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "s3_virtual_hosted_rewritten",
			body: `<img src="https://bucket-name.s3.ap-south-1.amazonaws.com/` + testUUID + `?X-Amz-Signature=abc">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "multiple_s3_urls_rewritten",
			body: `<img src="https://s3.amazonaws.com/b/` + testUUID + `?X-Amz-Signature=a">` +
				`<img src="https://s3.amazonaws.com/b/` + testUUID2 + `?X-Amz-Signature=b">`,
			want: `<img src="cid:ldsk-` + testUUID + `">` +
				`<img src="cid:ldsk-` + testUUID2 + `">`,
		},
		{
			name: "mixed_cid_and_s3_leaves_cid_alone",
			body: `<img src="cid:ldsk-` + testUUID + `">` +
				`<img src="https://s3.amazonaws.com/b/` + testUUID2 + `?X-Amz-Signature=x">`,
			want: `<img src="cid:ldsk-` + testUUID + `">` +
				`<img src="cid:ldsk-` + testUUID2 + `">`,
		},
		{
			name: "preserves_other_attributes",
			body: `<img class="inline-image" alt="hi" src="/uploads/` + testUUID + `">`,
			want: `<img class="inline-image" alt="hi" src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "preserves_single_quotes",
			body: `<img src='/uploads/` + testUUID + `'>`,
			want: `<img src='cid:ldsk-` + testUUID + `'>`,
		},
		{
			name: "rewrites_multiple",
			body: `<img src="/uploads/` + testUUID + `"><img src="/uploads/` + testUUID2 + `">`,
			want: `<img src="cid:ldsk-` + testUUID + `"><img src="cid:ldsk-` + testUUID2 + `">`,
		},
		{
			name: "leaves_cid_form_alone",
			body: `<img src="cid:ldsk-` + testUUID + `">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
		{
			name: "leaves_non_uploads_alone",
			body: `<a href="/uploads/` + testUUID + `">link</a>`,
			want: `<a href="/uploads/` + testUUID + `">link</a>`,
		},
		{
			name: "is_idempotent",
			body: `<img src="cid:ldsk-` + testUUID + `">`,
			want: `<img src="cid:ldsk-` + testUUID + `">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteInlineImagesToCID(tt.body)
			if got != tt.want {
				t.Errorf("\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}

	// Round-trip: extract from URL form, rewrite, then extract again should
	// produce zero URL-form matches (only cid-form references remain).
	t.Run("round_trip_url_to_cid", func(t *testing.T) {
		body := `<img src="/uploads/` + testUUID + `">`
		rewritten := rewriteInlineImagesToCID(body)
		if strings.Contains(rewritten, "/uploads/") {
			t.Errorf("rewritten body still contains /uploads/: %s", rewritten)
		}
		leftover := extractInlineImageUUIDs(rewritten)
		if len(leftover) != 0 {
			t.Errorf("expected 0 URL-form UUIDs after rewrite, got %v", leftover)
		}
	})

	t.Run("round_trip_s3_presigned_to_cid", func(t *testing.T) {
		body := `<img class="inline-image" src="https://s3.ap-south-1.amazonaws.com/bucket/` + testUUID + `?X-Amz-Algorithm=AWS4-HMAC-SHA256&amp;X-Amz-Signature=abc&amp;X-Amz-Expires=300">`
		rewritten := rewriteInlineImagesToCID(body)
		if strings.Contains(rewritten, "amazonaws.com") {
			t.Errorf("rewritten body still contains presigned URL: %s", rewritten)
		}
		if strings.Contains(rewritten, "X-Amz-Signature") {
			t.Errorf("rewritten body still contains X-Amz-Signature: %s", rewritten)
		}
		leftover := extractInlineImageUUIDs(rewritten)
		if len(leftover) != 0 {
			t.Errorf("expected 0 URL-form UUIDs after rewrite, got %v", leftover)
		}
	})
}

func TestEmailFromAddress(t *testing.T) {
	tests := []struct {
		name   string
		from   string
		sender string
		want   string
	}{
		{"primary keeps inbox from", "Acme Support <support@acme.com>", "support@acme.com", "Acme Support <support@acme.com>"},
		{"alias takes inbox name", "Acme Support <support@acme.com>", "billing@acme.com", `"Acme Support" <billing@acme.com>`},
		{"alias without inbox name", "support@acme.com", "billing@acme.com", "billing@acme.com"},
	}
	m := &Manager{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inb := fromTestInbox{from: tt.from, primary: "support@acme.com"}
			if got := m.emailFromAddress(inb, models.Message{SenderType: models.SenderTypeContact}, tt.sender); got != tt.want {
				t.Errorf("emailFromAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMessageArticleIDs(t *testing.T) {
	tests := []struct {
		meta string
		want []int
	}{
		{`{"ai_assistant_id":1,"ai_article_ids":[12,12,0,-1,34]}`, []int{12, 34}},
		{`{"ai_assistant_id":1,"ai_article_ids":[]}`, []int{}},
		{`{"ai_article_ids":[12]}`, nil},
		{`{"ai_assistant_id":0,"ai_article_ids":[12]}`, nil},
		{`{"ai_assistant_id":1,"ai_article_ids":["12"]}`, nil},
		{`not-json`, nil},
	}
	for _, tt := range tests {
		if got := messageArticleIDs(json.RawMessage(tt.meta)); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("messageArticleIDs(%s) = %v, want %v", tt.meta, got, tt.want)
		}
	}
}

func TestRenderArticleReferences(t *testing.T) {
	lo := logf.New(logf.Opts{})
	m := Manager{
		articleReferenceStore: citationArticleStore{references: []hcmodels.ArticleReference{
			{ID: 12, Title: `Refunds <script> & "policy"`, HelpCenterSlug: "support", Locale: "en", Slug: "refunds"},
			{ID: 34, Title: "Delivery", HelpCenterSlug: "support", Locale: "fr", Slug: "delivery", CustomDomain: "https://help.example.com"},
		}},
		settingsStore: citationSettingsStore{rootURL: "https://desk.example.com/"},
		i18n:          testutil.NewI18n(t),
		lo:            &lo,
	}
	meta := json.RawMessage(`{"ai_assistant_id":1,"ai_article_ids":[12,34]}`)
	message := models.Message{Content: "<p>Answer.</p>", Meta: meta}
	m.RenderArticleReferences(&message)
	for _, part := range []string{"(1)", "(2)", "https://desk.example.com/hc/support/en/articles/refunds", "https://help.example.com/fr/articles/delivery", "Refunds &lt;script&gt; &amp; &#34;policy&#34;"} {
		if !strings.Contains(message.Content, part) {
			t.Errorf("rendered references missing %q: %s", part, message.Content)
		}
	}
	if strings.Contains(message.Content, "<script>") {
		t.Fatal("article title inserted unescaped HTML")
	}
	plain := stringutil.HTML2TextMarkdownLinks(message.Content)
	for _, url := range []string{"https://desk.example.com/hc/support/en/articles/refunds", "https://help.example.com/fr/articles/delivery"} {
		if !strings.Contains(plain, url) {
			t.Errorf("plain text email is missing %s", url)
		}
	}
	if string(meta) != string(message.Meta) || strings.Contains(string(message.Meta), "https://") {
		t.Fatal("message metadata changed or persisted an absolute URL")
	}
	m.settingsStore = citationSettingsStore{rootURL: "https://new.example.com"}
	message.Content = "<p>Answer.</p>"
	m.RenderArticleReferences(&message)
	if strings.Contains(message.Content, "https://desk.example.com") || !strings.Contains(message.Content, "https://new.example.com/hc/support/en/articles/refunds") {
		t.Fatal("references did not follow the current root URL")
	}
	for _, store := range []citationArticleStore{{}, {err: errors.New("lookup failed")}} {
		m.articleReferenceStore = store
		message.Content = "<p>Answer.</p>"
		m.RenderArticleReferences(&message)
		if message.Content != "<p>Answer.</p>" {
			t.Fatal("unavailable references changed the answer")
		}
	}
}

func TestInlineArticleReferences(t *testing.T) {
	references := []hcmodels.ArticleReference{
		{ID: 12, Title: "JWT {{name}}", URL: "https://help.example.com/jwt"},
		{ID: 34, Title: "Logout", URL: "https://help.example.com/logout"},
	}
	content := "<p>Use JWT.<!--ld-cite:12--> Log out.<!--ld-cite:34--> JWT again.<!--ld-cite:12--></p>"
	rendered := chatArticleReferencesHTML(content, references)
	for _, part := range []string{
		"Use JWT.<sup>",
		"Log out.<sup>",
		"JWT again.<sup>",
		`title="JWT {{name}}"`,
		">(2)</a></sup>",
	} {
		if !strings.Contains(rendered, part) {
			t.Errorf("inline references missing %q: %s", part, rendered)
		}
	}
	if strings.Count(rendered, ">(1)</a></sup>") != 2 || strings.Contains(rendered, "ld-cite:") || strings.Contains(rendered, "<ul>") {
		t.Fatalf("unexpected inline references: %s", rendered)
	}
	if got := chatArticleReferencesHTML(content, nil /* references */); got != "<p>Use JWT. Log out. JWT again.</p>" {
		t.Fatalf("unavailable references changed the answer: %s", got)
	}
	if got := chatArticleReferencesHTML("<p>Answer.<!--ld-cite:999--></p>", references[:1]); strings.Contains(got, "ld-cite:") || !strings.Contains(got, "</p><p><sup>") {
		t.Fatalf("missing placement did not produce a numbered footer: %s", got)
	}
	lo := logf.New(logf.Opts{})
	m := Manager{
		articleReferenceStore: citationArticleStore{references: []hcmodels.ArticleReference{
			{ID: 12, Title: "JWT", HelpCenterSlug: "support", Locale: "en", Slug: "jwt"},
		}},
		settingsStore: citationSettingsStore{rootURL: "https://desk.example.com"},
		lo:            &lo,
	}
	message := models.Message{Content: content, Meta: json.RawMessage(`{"ai_assistant_id":1,"ai_article_ids":[12,34]}`)}
	data := map[string]any{}
	mailContent := m.emailTemplateContent(&message, data)
	if strings.Contains(mailContent, "ld-cite:") || strings.Contains(mailContent, "<sup>") {
		t.Fatalf("email retained widget markers: %s", mailContent)
	}
	if !strings.Contains(data["ArticleReferences"].(string), ">(1)</a>") {
		t.Fatal("email lost its numbered references")
	}
}

func TestArticleReferencePublicURL(t *testing.T) {
	reference := hcmodels.ArticleReference{HelpCenterSlug: "support", Locale: "en", Slug: "refund policy"}
	for _, tt := range []struct{ root, custom, want string }{
		{"https://desk.example.com/", "", "https://desk.example.com/hc/support/en/articles/refund%20policy"},
		{"https://desk.example.com", "https://help.example.com/path", "https://help.example.com/en/articles/refund%20policy"},
		{"", "https://help.example.com", "https://help.example.com/en/articles/refund%20policy"},
		{"", "", ""},
		{"javascript:alert(1)", "", ""},
	} {
		reference.CustomDomain = tt.custom
		if got := reference.PublicURL(tt.root); got != tt.want {
			t.Errorf("PublicURL(%q, %q) = %q, want %q", tt.root, tt.custom, got, tt.want)
		}
	}
}

func TestContinuityEmailArticleReferences(t *testing.T) {
	lo := logf.New(logf.Opts{})
	m := Manager{
		articleReferenceStore: citationArticleStore{references: []hcmodels.ArticleReference{
			{ID: 12, Title: "Refund {{name}}", HelpCenterSlug: "support", Locale: "en", Slug: "refunds"},
		}},
		settingsStore: citationSettingsStore{rootURL: "https://desk.example.com"},
		i18n:          testutil.NewI18n(t),
		lo:            &lo,
	}
	messages := []models.ContinuityUnreadMessage{
		{Message: models.Message{Content: "<p>AI answer.</p>", Meta: json.RawMessage(`{"ai_assistant_id":1,"ai_article_ids":[12,12,0,-1]}`)}},
		{Message: models.Message{Content: "<p>Human answer. {{name}} {{ .Author.FirstName }} {{</p>"}},
	}
	stored := m.buildContinuityEmailContent(messages, "" /* websiteURL */)
	rendered, _ := m.renderContinuityEmailContent(messages, "" /* websiteURL */)
	wantURL := "https://desk.example.com/hc/support/en/articles/refunds"
	if strings.Count(rendered, wantURL) != 1 || !strings.Contains(rendered, `target="_blank"`) {
		t.Fatalf("offline email is missing article links: %s", rendered)
	}
	if !strings.Contains(rendered, ">(1)</a>") || strings.Contains(rendered, "Refund {{name}}") || strings.Contains(rendered, "Article references") {
		t.Fatalf("offline email should contain numbered links: %s", rendered)
	}
	message := models.Message{Content: rendered, Meta: json.RawMessage(`{"continuity_email":true}`)}
	data := map[string]any{"Author": map[string]any{"FirstName": "Agent"}}
	content := m.emailTemplateContent(&message, data)
	tmpl, err := template.New("content").Parse(content)
	if err != nil {
		t.Fatalf("parsing offline email: %v", err)
	}
	var output strings.Builder
	if err := tmpl.Execute(&output, data); err != nil {
		t.Fatalf("rendering offline email: %v", err)
	}
	if output.String() != rendered || data["IsContinuityEmail"] != true {
		t.Fatal("offline email changed literal message content or lost its template flag")
	}
	if strings.Index(rendered, wantURL) > strings.Index(rendered, "Human answer.") {
		t.Fatal("article reference is attached to the wrong reply")
	}
	if !strings.Contains(stringutil.HTML2TextMarkdownLinks(rendered), wantURL) {
		t.Fatal("plain text offline email is missing the article URL")
	}
	if stored != m.buildContinuityEmailContent(messages, "" /* websiteURL */) || strings.Contains(stored, wantURL) {
		t.Fatal("rendering persisted an article URL in the saved email")
	}
	if again, _ := m.renderContinuityEmailContent(messages, "" /* websiteURL */); again != rendered {
		t.Fatal("rendering the offline email twice duplicated article references")
	}
	for _, store := range []citationArticleStore{{}, {err: errors.New("lookup failed")}} {
		m.articleReferenceStore = store
		if got, _ := m.renderContinuityEmailContent(messages, "" /* websiteURL */); got != stored {
			t.Fatal("unavailable references changed the offline email")
		}
	}
}

func TestEmailTemplateArticleReferences(t *testing.T) {
	lo := logf.New(logf.Opts{})
	for _, title := range []string{"Refund {{name}}", "Refund {{ .Author.FirstName }}", "Refund {{", `<script> & "refunds"`} {
		t.Run(title, func(t *testing.T) {
			m := Manager{
				articleReferenceStore: citationArticleStore{references: []hcmodels.ArticleReference{
					{ID: 12, Title: title, HelpCenterSlug: "support", Locale: "en", Slug: "refunds"},
					{ID: 34, Title: "Delivery", HelpCenterSlug: "support", Locale: "fr", Slug: "delivery", CustomDomain: "https://help.example.com"},
				}},
				settingsStore: citationSettingsStore{rootURL: "https://desk.example.com"},
				lo:            &lo,
			}
			message := models.Message{
				Content: "<p>Hello {{ .Author.FirstName }}.</p>",
				Meta:    json.RawMessage(`{"ai_assistant_id":1,"ai_article_ids":[12,34]}`),
			}
			data := map[string]any{"Author": map[string]any{"FirstName": "Agent"}}
			content := m.emailTemplateContent(&message, data)
			tmpl, err := template.New("content").Parse(content)
			if err != nil {
				t.Fatalf("parsing email: %v", err)
			}
			var output strings.Builder
			if err := tmpl.Execute(&output, data); err != nil {
				t.Fatalf("rendering email: %v", err)
			}
			want := `<p>Hello Agent.</p><p><a href="https://desk.example.com/hc/support/en/articles/refunds" target="_blank" rel="noopener noreferrer">(1)</a> <a href="https://help.example.com/fr/articles/delivery" target="_blank" rel="noopener noreferrer">(2)</a></p>`
			if output.String() != want || data["IsContinuityEmail"] != false {
				t.Fatalf("unexpected email content: %s", output.String())
			}
			plain := stringutil.HTML2TextMarkdownLinks(output.String())
			for _, part := range []string{"(1)", "(2)", "https://desk.example.com/hc/support/en/articles/refunds", "https://help.example.com/fr/articles/delivery"} {
				if !strings.Contains(plain, part) {
					t.Errorf("plain text email is missing %q: %s", part, plain)
				}
			}
			if again := m.emailTemplateContent(&message, data); again != content {
				t.Fatal("preparing the email twice duplicated article references")
			}
			for _, store := range []citationArticleStore{{}, {err: errors.New("lookup failed")}} {
				m.articleReferenceStore = store
				if got := m.emailTemplateContent(&message, data); got != message.Content {
					t.Fatal("unavailable references changed the email content")
				}
			}
		})
	}
}

func TestArticleReferenceBatchLookups(t *testing.T) {
	renderers := []struct {
		name   string
		render func(*Manager, []models.Message) string
	}{
		{"messages", func(m *Manager, messages []models.Message) string {
			m.RenderMessagesArticleReferences(messages)
			var output strings.Builder
			for _, message := range messages {
				output.WriteString(message.Content)
			}
			return output.String()
		}},
		{"transcript", func(m *Manager, messages []models.Message) string {
			return string(m.BuildTranscript(models.Conversation{}, messages, time.Time{} /* downloadedAt */))
		}},
		{"offline email", func(m *Manager, messages []models.Message) string {
			unread := make([]models.ContinuityUnreadMessage, len(messages))
			for i, message := range messages {
				unread[i].Message = message
			}
			content, _ := m.renderContinuityEmailContent(unread, "" /* websiteURL */)
			return content
		}},
	}
	for _, renderer := range renderers {
		t.Run(renderer.name, func(t *testing.T) {
			lo := logf.New(logf.Opts{})
			articleLookups, settingsLookups := 0, 0
			m := Manager{
				articleReferenceStore: citationArticleStore{
					references: []hcmodels.ArticleReference{
						{ID: 12, Title: "Refund policy", HelpCenterSlug: "support", Locale: "en", Slug: "refunds"},
						{ID: 34, Title: "Delivery", HelpCenterSlug: "support", Locale: "en", Slug: "delivery"},
					},
					lookups: &articleLookups,
				},
				settingsStore: citationSettingsStore{rootURL: "https://desk.example.com", lookups: &settingsLookups},
				i18n:          testutil.NewI18n(t),
				lo:            &lo,
			}
			messages := make([]models.Message, 151)
			for i := range 150 {
				messages[i] = models.Message{
					Content: "<p>AI answer.</p>",
					Meta:    json.RawMessage(`{"ai_assistant_id":1,"ai_article_ids":[34,12,34,0,-1,999]}`),
				}
			}
			messages[150].Content = "<p>Human answer.</p>"
			output := renderer.render(&m, messages)
			if articleLookups != 1 || settingsLookups != 1 {
				t.Fatalf("150 replies made %d article lookups and %d settings lookups", articleLookups, settingsLookups)
			}
			for _, slug := range []string{"delivery", "refunds"} {
				if count := strings.Count(output, "https://desk.example.com/hc/support/en/articles/"+slug); count != 150 {
					t.Errorf("article %s appeared %d times, want 150", slug, count)
				}
			}
			if strings.Index(output, "/articles/delivery") > strings.Index(output, "/articles/refunds") {
				t.Fatal("batch lookup changed citation order")
			}
			if strings.Contains(messages[150].Content, "href=") {
				t.Fatal("uncited reply gained article links")
			}
			articleLookups, settingsLookups = 0, 0
			renderer.render(&m, []models.Message{{Content: "<p>Human answer.</p>"}})
			renderer.render(&m, nil /* messages */)
			if articleLookups != 0 || settingsLookups != 0 {
				t.Fatal("uncited or empty batches performed reference lookups")
			}
		})
	}
}
