package conversation

import (
	"strings"
	"testing"
)

const testUUID = "d0355103-455f-4c7d-b9c7-86e9254fe119"
const testUUID2 = "edb7be78-ef7d-4fe9-888b-22494f0ce076"

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
