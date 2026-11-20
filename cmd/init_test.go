package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	hcmodels "github.com/abhinavxd/libredesk/internal/helpcenter/models"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/knadh/koanf/v2"
)

func TestInboxConfigCannotChangeAppPolicy(t *testing.T) {
	original := ko
	t.Cleanup(func() { ko = original })
	for _, channel := range []string{"livechat", "email"} {
		for _, allow := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/allow=%v", channel, allow), func(t *testing.T) {
				ko = koanf.New(".")
				for key, value := range map[string]any{
					"app.log_level": "error", "app.env": "dev", hcAppDomainCustomCodeKey: allow,
				} {
					if err := ko.Set(key, value); err != nil {
						t.Fatal(err)
					}
				}
				before := ko.Raw()
				for _, payload := range []string{
					`{"help_center":{"dangerously_allow_custom_code_on_app_domain":true}}`,
					`{"help_center":{"dangerously_allow_custom_code_on_app_domain":false}}`,
					`{"help_center.dangerously_allow_custom_code_on_app_domain":true}`,
					`{}`,
					`{`,
				} {
					record := imodels.Inbox{ID: 1, Channel: channel, Name: "Test", From: "support@example.com", Config: json.RawMessage(payload)}
					var err error
					if channel == "livechat" {
						_, err = initLiveChatInbox(record, nil /** msgStore **/, nil /** usrStore **/, nil /** signAvatarURL **/)
					} else {
						_, err = initEmailInbox(record, nil /** msgStore **/, nil /** usrStore **/, nil /** mgr **/, nil /** authStatusHook **/)
					}
					if (err != nil) != (payload == "{") {
						t.Fatalf("config %q: unexpected error %v", payload, err)
					}
					if !reflect.DeepEqual(ko.Raw(), before) {
						t.Fatalf("config %q changed application settings", payload)
					}
					app := newValidatorTestApp(t)
					if got := helpCenterCustomCodeAllowed(app, request(testAppBaseURL+"/hc/docs/en", "" /** ifNoneMatch **/), hcmodels.HelpCenter{Slug: "docs"}); got != allow {
						t.Fatalf("custom code allowed = %v, want startup policy %v", got, allow)
					}
				}
			})
		}
	}
}
