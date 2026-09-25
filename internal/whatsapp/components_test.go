package whatsapp

import (
	"reflect"
	"testing"
)

func TestBuildSendComponents_StaticHeaderOmitted(t *testing.T) {
	parts := TemplateSendParts{
		HeaderType:    "TEXT",
		HeaderContent: "Order update",
		BodyContent:   "Hi {{1}}, your order {{2}} is now {{3}}",
		Params:        map[string]string{"body:1": "John", "body:2": "ORD-12345", "body:3": "shipped"},
	}
	out := BuildSendComponents(parts)
	if len(out) != 1 {
		t.Fatalf("expected only body component for static header, got %d components", len(out))
	}
	body := out[0]
	if body["type"] != "body" {
		t.Fatalf("expected body component, got %q", body["type"])
	}
	params, _ := body["parameters"].([]map[string]any)
	if len(params) != 3 {
		t.Fatalf("expected 3 body parameters, got %d", len(params))
	}
	if params[0]["text"] != "John" || params[1]["text"] != "ORD-12345" || params[2]["text"] != "shipped" {
		t.Fatalf("body params out of order: %v", params)
	}
}

func TestBuildSendComponents_ParameterizedTextHeader(t *testing.T) {
	parts := TemplateSendParts{
		HeaderType:    "TEXT",
		HeaderContent: "Order {{1}}",
		BodyContent:   "ETA {{1}}",
		Params:        map[string]string{"header:1": "12345", "body:1": "tomorrow"},
	}
	out := BuildSendComponents(parts)
	if len(out) != 2 {
		t.Fatalf("expected header+body, got %d", len(out))
	}
	hdr := out[0]
	if hdr["type"] != "header" {
		t.Fatalf("expected header first, got %q", hdr["type"])
	}
	hdrParams, _ := hdr["parameters"].([]map[string]any)
	if len(hdrParams) != 1 || hdrParams[0]["text"] != "12345" {
		t.Fatalf("header param wrong: %v", hdrParams)
	}
	body := out[1]
	bodyParams, _ := body["parameters"].([]map[string]any)
	if len(bodyParams) != 1 || bodyParams[0]["text"] != "tomorrow" {
		t.Fatalf("body param wrong: %v", bodyParams)
	}
}

func TestBuildSendComponents_NamedParameters(t *testing.T) {
	parts := TemplateSendParts{
		BodyContent: "Hi {{name}}, order {{order_id}}",
		Params:      map[string]string{"body:name": "John", "body:order_id": "12345"},
	}
	out := BuildSendComponents(parts)
	if len(out) != 1 {
		t.Fatalf("expected one body component, got %d", len(out))
	}
	params, _ := out[0]["parameters"].([]map[string]any)
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	want := []map[string]any{
		{"type": "text", "parameter_name": "name", "text": "John"},
		{"type": "text", "parameter_name": "order_id", "text": "12345"},
	}
	if !reflect.DeepEqual(params, want) {
		t.Fatalf("named params mismatch.\n got: %v\nwant: %v", params, want)
	}
}

func TestBuildSendComponents_URLButton(t *testing.T) {
	parts := TemplateSendParts{
		BodyContent: "Track your shipment",
		Buttons: []TemplateButton{
			{Type: "URL", Text: "Track", URL: "https://example.com/{{1}}"},
		},
		Params: map[string]string{"button_url_0": "12345"},
	}
	out := BuildSendComponents(parts)
	if len(out) != 1 {
		t.Fatalf("expected only button component, got %d", len(out))
	}
	btn := out[0]
	if btn["sub_type"] != "url" || btn["index"] != "0" {
		t.Fatalf("button shape wrong: %v", btn)
	}
}

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	secret := "topsecret"
	// echo -n '{"hello":"world"}' | openssl dgst -sha256 -hmac topsecret
	good := "sha256=afd00617ceb8f63e65ea5c310f06bf78c3901e7a713db532e25da26ad63c7236"
	bad := "sha256=deadbeef"
	if !VerifySignature(body, good, secret) {
		t.Fatalf("expected valid signature to verify")
	}
	if VerifySignature(body, bad, secret) {
		t.Fatalf("expected bad signature to fail")
	}
	if VerifySignature(body, "", secret) {
		t.Fatalf("expected empty header to fail")
	}
	if VerifySignature(body, good, "") {
		t.Fatalf("expected empty secret to fail")
	}
}

func TestOrderedPlaceholders(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"empty text", "", nil},
		{"no placeholders", "plain copy", nil},
		{"positional in order", "Hi {{1}}, order {{2}}", []string{"1", "2"}},
		{"named in order", "Hi {{name}}, order {{order_id}}", []string{"name", "order_id"}},
		{"repeats appear once", "{{name}} and {{name}} again", []string{"name"}},
		{"order follows first use", "{{b}} then {{a}} then {{b}}", []string{"b", "a"}},
		{"malformed braces ignored", "{{}} {{ x }} {{a-b}}", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := OrderedPlaceholders(tc.text)
			if len(got) != len(tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("expected %v, got %v", tc.want, got)
				}
			}
		})
	}
}

// A media header takes its file at send time, so it contributes no component here.
func TestBuildSendComponents_MediaHeaderSkipped(t *testing.T) {
	comps := BuildSendComponents(TemplateSendParts{
		HeaderType:    "IMAGE",
		HeaderContent: "ignored",
		BodyContent:   "Hi there",
	})
	if len(comps) != 0 {
		t.Fatalf("expected no components, got %+v", comps)
	}
}

func TestBuildSendComponents_TextHeaderWithoutParams(t *testing.T) {
	comps := BuildSendComponents(TemplateSendParts{
		HeaderType:    "TEXT",
		HeaderContent: "Static header",
		BodyContent:   "Hi there",
	})
	if len(comps) != 0 {
		t.Fatalf("expected no components for a static header and body, got %+v", comps)
	}
}

func TestBuildSendComponents_ButtonVariants(t *testing.T) {
	tests := []struct {
		name    string
		buttons []TemplateButton
		params  map[string]string
		want    int
	}{
		{
			name:    "quick reply buttons take no parameters",
			buttons: []TemplateButton{{Type: "QUICK_REPLY", Text: "Yes"}},
			params:  map[string]string{"button_url_0": "ignored"},
			want:    0,
		},
		{
			name:    "dynamic url button with no value is skipped",
			buttons: []TemplateButton{{Type: "URL", Text: "Track", URL: "https://x.test/{{1}}"}},
			params:  nil,
			want:    0,
		},
		{
			name:    "dynamic url button with an empty value is skipped",
			buttons: []TemplateButton{{Type: "URL", Text: "Track", URL: "https://x.test/{{1}}"}},
			params:  map[string]string{"button_url_0": ""},
			want:    0,
		},
		{
			name:    "dynamic url button with a value is sent",
			buttons: []TemplateButton{{Type: "URL", Text: "Track", URL: "https://x.test/{{1}}"}},
			params:  map[string]string{"button_url_0": "A1"},
			want:    1,
		},
		{
			name: "only the dynamic button of several is sent",
			buttons: []TemplateButton{
				{Type: "QUICK_REPLY", Text: "No"},
				{Type: "URL", Text: "Track", URL: "https://x.test/{{1}}"},
			},
			params: map[string]string{"button_url_1": "A1"},
			want:   1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			comps := BuildSendComponents(TemplateSendParts{
				BodyContent: "Hi there",
				Buttons:     tc.buttons,
				Params:      tc.params,
			})
			if len(comps) != tc.want {
				t.Fatalf("expected %d components, got %+v", tc.want, comps)
			}
		})
	}
}

// The button index Meta needs is the button's position in the template, not the order it was filled.
func TestBuildSendComponents_ButtonIndexMatchesPosition(t *testing.T) {
	comps := BuildSendComponents(TemplateSendParts{
		BodyContent: "Hi",
		Buttons: []TemplateButton{
			{Type: "QUICK_REPLY", Text: "No"},
			{Type: "URL", Text: "Track", URL: "https://x.test/{{1}}"},
		},
		Params: map[string]string{"button_url_1": "A1"},
	})
	if len(comps) != 1 || comps[0]["index"] != "1" || comps[0]["sub_type"] != "url" {
		t.Fatalf("unexpected button component: %+v", comps)
	}
}
