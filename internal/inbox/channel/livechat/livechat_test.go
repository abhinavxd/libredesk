package livechat

import (
	"encoding/json"
	"testing"
)

func TestResolvePreChatFormFallsBackToLegacyValues(t *testing.T) {
	var config Config
	if err := json.Unmarshal([]byte(`{
		"prechat_form":{
			"enabled":true,
			"title":"Before we start",
			"fields":[{"key":"email","label":"Email"}]
		}
	}`), &config); err != nil {
		t.Fatal(err)
	}

	for _, isVisitor := range []bool{true, false} {
		resolved := config.ResolvePreChatForm(isVisitor)
		if !resolved.PreChatForm.Enabled || resolved.PreChatForm.Title != "Before we start" {
			t.Fatalf("resolved pre-chat form = %+v", resolved.PreChatForm)
		}
		if len(resolved.PreChatForm.Fields) != 1 || resolved.PreChatForm.Fields[0].Key != "email" {
			t.Fatalf("resolved fields = %+v", resolved.PreChatForm.Fields)
		}
	}
}

func TestResolvePreChatFormUsesAudienceValues(t *testing.T) {
	var config Config
	if err := json.Unmarshal([]byte(`{
		"prechat_form":{
			"enabled":true,
			"title":"Legacy title",
			"fields":[{"key":"email"}],
			"visitors":{"enabled":true,"title":"Choose a plan","fields":[{"key":"plan"}]},
			"users":{"enabled":false,"title":"What is the issue?","fields":[]}
		}
	}`), &config); err != nil {
		t.Fatal(err)
	}

	visitor := config.ResolvePreChatForm(true)
	if !visitor.PreChatForm.Enabled || visitor.PreChatForm.Title != "Choose a plan" {
		t.Fatalf("visitor pre-chat form = %+v", visitor.PreChatForm)
	}
	if len(visitor.PreChatForm.Fields) != 1 || visitor.PreChatForm.Fields[0].Key != "plan" {
		t.Fatalf("visitor fields = %+v", visitor.PreChatForm.Fields)
	}
	user := config.ResolvePreChatForm(false)
	if user.PreChatForm.Enabled || user.PreChatForm.Title != "What is the issue?" {
		t.Fatalf("user pre-chat form = %+v", user.PreChatForm)
	}
	if user.PreChatForm.Fields == nil || len(user.PreChatForm.Fields) != 0 {
		t.Fatalf("user fields = %+v", user.PreChatForm.Fields)
	}
}
