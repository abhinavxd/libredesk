package activationkey

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/zerodha/logf"
)

func newTestManager(t *testing.T, name string) *Manager {
	t.Helper()
	db := testutil.NewDB(t, name)
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, EncryptionKey: strings.Repeat("k", 32)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.UpdateSettings(Settings{Enabled: true, MaxKeysPerEmail: 1, LowStockThreshold: 10}); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestImportEncryptsAndRejectsDuplicatesEverywhere(t *testing.T) {
	m := newTestManager(t, "keys_import")
	first, err := m.CreateApp("First Game")
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.CreateApp("Second Game")
	if err != nil {
		t.Fatal(err)
	}
	result, err := m.Import(first.ID, "ABCDE-FGHIJ-KLMNO\nabcde-fghij-klmno, short ; <b>bad</b>\r\n  QQQQQ-WWWWW-EEEEE  \n\nnot a key\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	if result != (ImportResult{Added: 2, Duplicates: 1, Invalid: 3}) {
		t.Fatalf("first import = %+v", result)
	}
	// The same key can't sit in another game's pool either.
	if result, err = m.Import(second.ID, "QQQQQ-WWWWW-EEEEE", 0); err != nil || result.Added != 0 || result.Duplicates != 1 {
		t.Fatalf("cross-game duplicate = %+v %v", result, err)
	}
	var stored []string
	if err := m.db.Select(&stored, `SELECT key_encrypted FROM activation_keys`); err != nil {
		t.Fatal(err)
	}
	for _, value := range stored {
		if !strings.HasPrefix(value, "enc:") || strings.Contains(strings.ToUpper(value), "ABCDE") {
			t.Fatalf("key stored without encryption: %q", value)
		}
	}
	if _, err := m.CreateApp("first game"); err == nil {
		t.Fatal("game names must be unique regardless of case")
	}
}

func TestPoolListingNeverCarriesKeysAndRevealsOneAtATime(t *testing.T) {
	m := newTestManager(t, "keys_listing")
	app, err := m.CreateApp("Game")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Import(app.ID, "ABCDE-FGHIJ-KLMNO\nQQQQQ-WWWWW-EEEEE", 0); err != nil {
		t.Fatal(err)
	}
	keys, total, err := m.Keys(KeyQuery{AppID: app.ID, Status: StatusRedeemable, Page: 1, PerPage: 10})
	if err != nil || total != 2 || len(keys) != 2 {
		t.Fatalf("listing = %d/%d %v", len(keys), total, err)
	}
	raw, _ := json.Marshal(keys)
	if strings.Contains(string(raw), "ABCDE") || strings.Contains(string(raw), "enc:") {
		t.Fatalf("listing leaks keys: %s", raw)
	}
	found, total, err := m.Keys(KeyQuery{AppID: app.ID, Status: StatusRedeemable, Search: " qqqqq-wwwww-eeeee ", Page: 1, PerPage: 10})
	if err != nil || total != 1 {
		t.Fatalf("exact search = %d %v", total, err)
	}
	if key, err := m.Reveal(found[0].ID); err != nil || key != "QQQQQ-WWWWW-EEEEE" {
		t.Fatalf("reveal = %q %v", key, err)
	}
	if partial, total, _ := m.Keys(KeyQuery{AppID: app.ID, Status: StatusRedeemable, Search: "QQQQQ", Page: 1, PerPage: 10}); total != 0 || len(partial) != 0 {
		t.Fatal("a partial key must not match")
	}

	// Voiding takes a key out of circulation for good; the row stays as a record.
	if err := m.Void(found[0].ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := m.Void(found[0].ID, 0); err == nil {
		t.Fatal("voided twice")
	}
	if _, err := m.db.Exec(`UPDATE activation_keys SET status='redeemable' WHERE id=$1`, found[0].ID); err == nil {
		t.Fatal("a voided key came back")
	}
	if _, err := m.Reveal(found[0].ID); err == nil {
		t.Fatal("a voided key was revealed")
	}
	apps, _ := m.Apps()
	if apps[0].Redeemable != 1 || apps[0].Activated != 0 {
		t.Fatalf("pool sizes = %+v", apps[0])
	}
}

func TestComposerOffersOnlyLiveGames(t *testing.T) {
	m := newTestManager(t, "keys_composer")
	stocked, _ := m.CreateApp("Stocked")
	empty, _ := m.CreateApp("Empty")
	archived, _ := m.CreateApp("Archived")
	if _, err := m.Import(stocked.ID, "ABCDE-FGHIJ-KLMNO", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := m.UpdateApp(archived.ID, "Archived", true); err != nil {
		t.Fatal(err)
	}
	var userID int
	if err := m.db.Get(&userID, `INSERT INTO users(type,email,first_name) VALUES('agent','a@example.test','A') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := m.SetLastApp(userID, empty.ID); err != nil {
		t.Fatal(err)
	}
	if err := m.SetLastApp(userID, archived.ID); err == nil {
		t.Fatal("remembered an archived game")
	}
	info, err := m.Composer(userID)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Enabled || len(info.Apps) != 2 || info.LastAppID.Int != empty.ID {
		t.Fatalf("composer = %+v", info)
	}
	for _, app := range info.Apps {
		if app.Available != (app.ID == stocked.ID) {
			t.Fatalf("availability = %+v", info.Apps)
		}
	}
	if _, err := m.UpdateSettings(Settings{Enabled: false, MaxKeysPerEmail: 1}); err != nil {
		t.Fatal(err)
	}
	if info, _ := m.Composer(userID); info.Enabled || len(info.Apps) != 0 {
		t.Fatalf("disabled composer = %+v", info)
	}
	if _, err := m.UpdateSettings(Settings{Enabled: true, MaxKeysPerEmail: MaxKeysPerEmailLimit + 1}); err == nil {
		t.Fatal("accepted a cap above the limit")
	}
}
