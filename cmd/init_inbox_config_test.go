package main

import (
	"encoding/json"
	"testing"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalInboxConfigIsolatesInboxes(t *testing.T) {
	var delegated imodels.Config
	require.NoError(t, unmarshalInboxConfig(json.RawMessage(`{"auth_type":"oauth2","oauth":{"provider":"microsoft","access_token":"delegated-token","expires_at":"2099-01-01T00:00:00Z"},"smtp":[{"host":"smtp.office365.com","port":587}]}`), &delegated))
	require.Equal(t, "delegated-token", delegated.OAuth.AccessToken)

	var appOnly imodels.Config
	require.NoError(t, unmarshalInboxConfig(json.RawMessage(`{"auth_type":"oauth2_app","oauth":{"provider":"microsoft","client_id":"app"}}`), &appOnly))
	require.Equal(t, "app", appOnly.OAuth.ClientID)
	require.Empty(t, appOnly.OAuth.AccessToken, "keys missing from one inbox must not inherit another inbox's values")
	require.True(t, appOnly.OAuth.ExpiresAt.IsZero())
	require.Empty(t, appOnly.SMTP)

	require.False(t, ko.Exists("oauth"), "inbox config must not leak into app config")
}
