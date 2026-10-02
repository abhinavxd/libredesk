package migrations

import (
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/jmoiron/sqlx"
)

func TestSiteLogoUpgradeFoldsFaviconAndClearsDefaultTitle(t *testing.T) {
	for _, tc := range []struct {
		name, title, logo, favicon string
		wantTitle, wantLogo        string
	}{
		{name: "defaults", title: "Fernmail", favicon: "/favicon.svg", wantTitle: "", wantLogo: ""},
		{name: "root-url default", title: "Fernmail", favicon: "https://mail.example.com/favicon.ico", wantTitle: "", wantLogo: ""},
		{name: "custom favicon", title: "Studio mail", favicon: "https://cdn.example.com/icon.png", wantTitle: "Studio mail", wantLogo: "https://cdn.example.com/icon.png"},
		{name: "logo wins", title: "Fernmail", logo: "https://cdn.example.com/logo.png", favicon: "https://cdn.example.com/icon.png", wantTitle: "", wantLogo: "https://cdn.example.com/logo.png"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t, "site_logo_upgrade")
			db.MustExec(`UPDATE settings SET value = to_jsonb('https://mail.example.com/'::text) WHERE key = 'app.root_url'`)
			db.MustExec(`UPDATE settings SET value = to_jsonb($1::text) WHERE key = 'app.site_name'`, tc.title)
			db.MustExec(`UPDATE settings SET value = to_jsonb($1::text) WHERE key = 'app.logo_url'`, tc.logo)
			restoreFaviconSetting(t, db, tc.favicon)
			for range 2 {
				if err := V3_8_0(db, nil, nil); err != nil {
					t.Fatal(err)
				}
			}
			for key, want := range map[string]string{"app.site_name": tc.wantTitle, "app.logo_url": tc.wantLogo} {
				var got string
				if err := db.Get(&got, `SELECT value #>> '{}' FROM settings WHERE key = $1`, key); err != nil {
					t.Fatal(err)
				}
				if got != want {
					t.Fatalf("%s: got %q, want %q", key, got, want)
				}
			}
			var favicons int
			if err := db.Get(&favicons, `SELECT count(*) FROM settings WHERE key = 'app.favicon_url'`); err != nil || favicons != 0 {
				t.Fatalf("app.favicon_url rows = %d, err = %v", favicons, err)
			}
		})
	}
}

// restoreFaviconSetting puts back the app.favicon_url setting that v3.8.0
// folds into the site logo.
func restoreFaviconSetting(t *testing.T, db *sqlx.DB, value string) {
	t.Helper()
	db.MustExec(`INSERT INTO settings (key, value) VALUES ('app.favicon_url', to_jsonb($1::text))
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, value)
}
