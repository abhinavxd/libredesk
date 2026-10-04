// Package activationkey owns Key Distribution: per-app pools of activation
// keys that agents attach to outgoing email from the composer. A key is
// redeemable until an email carrying its placeholder is queued, then it is
// activated and bound to that message for good. Keys are encrypted at rest
// and only one is ever revealed per request.
package activationkey

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jakedolan443/fernmail/internal/crypto"
	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

const (
	StatusRedeemable = "redeemable"
	StatusActivated  = "activated"
	StatusVoided     = "voided"

	settingsKey = "key_distribution"

	// MaxKeysPerEmailLimit bounds the per-email cap an admin can choose.
	MaxKeysPerEmailLimit = 20
	maxImportKeys        = 10000
	maxAppName           = 100
	maxPageSize          = 100
)

// Settings is the workspace's Key Distribution configuration.
type Settings struct {
	Enabled           bool `json:"enabled"`
	MaxKeysPerEmail   int  `json:"max_keys_per_email"`
	LowStockThreshold int  `json:"low_stock_threshold"`
}

// DefaultSettings is what a workspace starts with: off, one key per email.
func DefaultSettings() Settings {
	return Settings{Enabled: false, MaxKeysPerEmail: 1, LowStockThreshold: 10}
}

// App is a game (or any product) with its own pair of key pools.
type App struct {
	ID         int       `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	Archived   bool      `db:"archived" json:"archived"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	Redeemable int       `db:"redeemable" json:"redeemable"`
	Activated  int       `db:"activated" json:"activated"`
}

// Key is a pool row as the admin page lists it. It never carries the key
// itself; see Reveal.
type Key struct {
	ID               int64          `db:"id" json:"id"`
	Status           string         `db:"status" json:"status"`
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
	ActivatedAt      null.Time      `db:"activated_at" json:"activated_at"`
	AddedByName      null.String    `db:"added_by_name" json:"added_by_name"`
	SentByName       null.String    `db:"sent_by_name" json:"sent_by_name"`
	ApprovedByName   null.String    `db:"approved_by_name" json:"approved_by_name"`
	Recipients       pq.StringArray `db:"recipients" json:"recipients"`
	ConversationUUID null.String    `db:"conversation_uuid" json:"conversation_uuid"`
	ConversationRef  null.String    `db:"conversation_ref" json:"conversation_ref"`
	AddressID        null.Int       `db:"address_id" json:"address_id"`
	DeliveryStatus   null.String    `db:"delivery_status" json:"delivery_status"`
	Total            int            `db:"total" json:"-"`
}

// KeyQuery selects one page of one pool.
type KeyQuery struct {
	AppID   int
	Status  string
	Search  string
	Page    int
	PerPage int
}

// ImportResult summarises an import.
type ImportResult struct {
	Added      int `json:"added"`
	Duplicates int `json:"duplicates"`
	Invalid    int `json:"invalid"`
}

// ComposerApp is a game as the composer's key card offers it.
type ComposerApp struct {
	ID        int    `db:"id" json:"id"`
	Name      string `db:"name" json:"name"`
	Available bool   `db:"available" json:"available"`
}

// ComposerInfo is everything the key card needs.
type ComposerInfo struct {
	Enabled         bool          `json:"enabled"`
	MaxKeysPerEmail int           `json:"max_keys_per_email"`
	Apps            []ComposerApp `json:"apps"`
	LastAppID       null.Int      `json:"last_app_id"`
}

// Allocation describes the message that is taking keys.
type Allocation struct {
	Placeholders   []Placeholder
	MessageID      int
	ConversationID int
	SenderID       int
	ApprovedBy     int
	Recipients     []string
}

// Manager stores pools and moves keys between them.
type Manager struct {
	db            *sqlx.DB
	encryptionKey string
	lo            *logf.Logger
}

// Opts configures a Manager.
type Opts struct {
	DB            *sqlx.DB
	Lo            *logf.Logger
	EncryptionKey string
}

// New returns a Manager.
func New(opts Opts) (*Manager, error) {
	if opts.DB == nil {
		return nil, errors.New("activation key database is required")
	}
	if len(opts.EncryptionKey) != 32 {
		return nil, crypto.ErrInvalidKey
	}
	return &Manager{db: opts.DB, encryptionKey: opts.EncryptionKey, lo: opts.Lo}, nil
}

func inputError(message string) error {
	return envelope.NewError(envelope.InputError, message, nil)
}

func (m *Manager) internalError(context string, err error) error {
	m.lo.Error("activation keys: "+context, "error", err)
	return envelope.NewError(envelope.GeneralError, "Something went wrong with activation keys. Please try again.", nil)
}

// hash is the lookup and duplicate-check digest. Keys are compared without
// regard to case, as people retype them.
func (m *Manager) hash(key string) string {
	mac := hmac.New(sha256.New, []byte(m.encryptionKey))
	mac.Write([]byte("activation-key:" + strings.ToUpper(key)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (m *Manager) decrypt(ciphertext string) (string, error) {
	// crypto.Decrypt passes unprefixed values through; a key must never be stored that way.
	if !strings.HasPrefix(ciphertext, crypto.EncryptedPrefix) {
		return "", crypto.ErrInvalidCiphertext
	}
	key, err := crypto.Decrypt(ciphertext, m.encryptionKey)
	if err != nil {
		return "", err
	}
	if _, ok := NormalizeKey(key); !ok {
		return "", crypto.ErrInvalidCiphertext
	}
	return key, nil
}

// --- Settings ---------------------------------------------------------------

// GetSettings returns the current settings, falling back to the defaults.
func (m *Manager) GetSettings() (Settings, error) {
	return m.settings(m.db)
}

func (m *Manager) settings(q sqlx.Queryer) (Settings, error) {
	var raw []byte
	err := sqlx.Get(q, &raw, `SELECT value FROM settings WHERE "key" = $1`, settingsKey)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultSettings(), nil
	}
	if err != nil {
		return Settings{}, err
	}
	s := DefaultSettings()
	if err := json.Unmarshal(raw, &s); err != nil {
		return Settings{}, err
	}
	return normalizeSettings(s), nil
}

func normalizeSettings(s Settings) Settings {
	s.MaxKeysPerEmail = min(max(s.MaxKeysPerEmail, 1), MaxKeysPerEmailLimit)
	s.LowStockThreshold = max(s.LowStockThreshold, 0)
	return s
}

// UpdateSettings validates and saves the settings.
func (m *Manager) UpdateSettings(s Settings) (Settings, error) {
	if s.MaxKeysPerEmail < 1 || s.MaxKeysPerEmail > MaxKeysPerEmailLimit {
		return Settings{}, inputError(fmt.Sprintf("Keys per email must be between 1 and %d.", MaxKeysPerEmailLimit))
	}
	if s.LowStockThreshold < 0 || s.LowStockThreshold > 1000000 {
		return Settings{}, inputError("The low stock warning must be zero or more.")
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return Settings{}, m.internalError("encoding settings", err)
	}
	if _, err := m.db.Exec(`INSERT INTO settings ("key", value, updated_at) VALUES ($1, $2, NOW())
		ON CONFLICT ("key") DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, settingsKey, raw); err != nil {
		return Settings{}, m.internalError("saving settings", err)
	}
	return s, nil
}

// RequireEnabled refuses pool changes while Key Distribution is off.
func (m *Manager) RequireEnabled() error {
	s, err := m.GetSettings()
	if err != nil {
		return m.internalError("reading settings", err)
	}
	if !s.Enabled {
		return inputError("Turn on key distribution first.")
	}
	return nil
}

// --- Apps -------------------------------------------------------------------

const appSelect = `SELECT a.id, a.name, a.archived_at IS NOT NULL AS archived, a.created_at,
	count(k.id) FILTER (WHERE k.status = 'redeemable') AS redeemable,
	count(k.id) FILTER (WHERE k.status = 'activated') AS activated
FROM activation_key_apps a LEFT JOIN activation_keys k ON k.app_id = a.id`

// Apps lists every game with its pool sizes, archived ones last.
func (m *Manager) Apps() ([]App, error) {
	apps := []App{}
	if err := m.db.Select(&apps, appSelect+` GROUP BY a.id ORDER BY a.archived_at IS NOT NULL, lower(a.name)`); err != nil {
		return nil, m.internalError("listing apps", err)
	}
	return apps, nil
}

func (m *Manager) app(id int) (App, error) {
	var app App
	err := m.db.Get(&app, appSelect+` WHERE a.id = $1 GROUP BY a.id`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return app, envelope.NewError(envelope.NotFoundError, "That game no longer exists.", nil)
	}
	if err != nil {
		return app, m.internalError("reading app", err)
	}
	return app, nil
}

func validAppName(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" || len([]rune(name)) > maxAppName {
		return "", inputError(fmt.Sprintf("Give the game a name of at most %d characters.", maxAppName))
	}
	return name, nil
}

func (m *Manager) appWriteError(context string, err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return envelope.NewError(envelope.ConflictError, "A game with that name already exists.", nil)
	}
	return m.internalError(context, err)
}

// CreateApp adds a game with two empty pools.
func (m *Manager) CreateApp(name string) (App, error) {
	name, err := validAppName(name)
	if err != nil {
		return App{}, err
	}
	var id int
	if err := m.db.Get(&id, `INSERT INTO activation_key_apps (name) VALUES ($1) RETURNING id`, name); err != nil {
		return App{}, m.appWriteError("creating app", err)
	}
	return m.app(id)
}

// UpdateApp renames a game and archives or restores it. Games are never
// deleted, so their pools are always kept.
func (m *Manager) UpdateApp(id int, name string, archived bool) (App, error) {
	name, err := validAppName(name)
	if err != nil {
		return App{}, err
	}
	res, err := m.db.Exec(`UPDATE activation_key_apps SET name = $2, updated_at = NOW(),
		archived_at = CASE WHEN $3 THEN COALESCE(archived_at, NOW()) ELSE NULL END WHERE id = $1`, id, name, archived)
	if err != nil {
		return App{}, m.appWriteError("updating app", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return App{}, envelope.NewError(envelope.NotFoundError, "That game no longer exists.", nil)
	}
	return m.app(id)
}

// --- Pools ------------------------------------------------------------------

var conversationRefPattern = regexp.MustCompile(`^#?([A-Za-z0-9-]{1,40})$`)

// Keys returns one page of a pool. Search matches a whole key exactly, and in
// the activated pool also a recipient or a conversation reference.
func (m *Manager) Keys(q KeyQuery) ([]Key, int, error) {
	if q.Status != StatusRedeemable && q.Status != StatusActivated {
		return nil, 0, inputError("Unknown pool.")
	}
	if _, err := m.app(q.AppID); err != nil {
		return nil, 0, err
	}
	q.PerPage = min(max(q.PerPage, 1), maxPageSize)
	q.Page = max(q.Page, 1)

	var (
		where = []string{"k.app_id = $1", "k.status = $2"}
		args  = []any{q.AppID, q.Status}
	)
	if search := strings.TrimSpace(q.Search); search != "" {
		var or []string
		if key, ok := NormalizeKey(search); ok {
			args = append(args, m.hash(key))
			or = append(or, fmt.Sprintf("k.key_hash = $%d", len(args)))
		}
		if q.Status == StatusActivated {
			args = append(args, "%"+escapeLike(strings.ToLower(search))+"%")
			or = append(or, fmt.Sprintf("lower(array_to_string(k.recipients, ' ')) LIKE $%d", len(args)))
			if ref := conversationRefPattern.FindStringSubmatch(search); ref != nil {
				args = append(args, ref[1])
				or = append(or, fmt.Sprintf("c.reference_number = $%d", len(args)))
			}
		}
		if len(or) == 0 {
			return []Key{}, 0, nil
		}
		where = append(where, "("+strings.Join(or, " OR ")+")")
	}
	order := "k.id"
	if q.Status == StatusActivated {
		order = "k.activated_at DESC, k.id DESC"
	}
	args = append(args, q.PerPage, (q.Page-1)*q.PerPage)
	keys := []Key{}
	err := m.db.Select(&keys, fmt.Sprintf(`SELECT k.id, k.status, k.created_at, k.activated_at, k.recipients,
		NULLIF(concat_ws(' ', ad.first_name, NULLIF(ad.last_name, '')), '') AS added_by_name,
		NULLIF(concat_ws(' ', sb.first_name, NULLIF(sb.last_name, '')), '') AS sent_by_name,
		NULLIF(concat_ws(' ', ab.first_name, NULLIF(ab.last_name, '')), '') AS approved_by_name,
		c.uuid::text AS conversation_uuid, c.reference_number AS conversation_ref, c.address_id, msg.status::text AS delivery_status,
		count(*) OVER () AS total
		FROM activation_keys k
		LEFT JOIN users ad ON ad.id = k.added_by
		LEFT JOIN users sb ON sb.id = k.sent_by
		LEFT JOIN users ab ON ab.id = k.approved_by
		LEFT JOIN conversations c ON c.id = k.conversation_id
		LEFT JOIN conversation_messages msg ON msg.id = k.message_id
		WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`, strings.Join(where, " AND "), order, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, m.internalError("listing keys", err)
	}
	total := 0
	if len(keys) > 0 {
		total = keys[0].Total
	}
	return keys, total, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// Import adds keys to a game's redeemable pool, one per line (commas,
// semicolons and tabs also separate). A key already in any pool is skipped.
func (m *Manager) Import(appID int, raw string, userID int) (ImportResult, error) {
	var result ImportResult
	app, err := m.app(appID)
	if err != nil {
		return result, err
	}
	if app.Archived {
		return result, inputError("Restore this game before adding keys.")
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(fields) > maxImportKeys {
		return result, inputError(fmt.Sprintf("Add at most %d keys at a time.", maxImportKeys))
	}
	type row struct{ encrypted, hash string }
	var (
		rows []row
		seen = map[string]bool{}
	)
	for _, field := range fields {
		if strings.TrimSpace(field) == "" {
			continue
		}
		key, ok := NormalizeKey(field)
		if !ok {
			result.Invalid++
			continue
		}
		hash := m.hash(key)
		if seen[hash] {
			result.Duplicates++
			continue
		}
		seen[hash] = true
		encrypted, err := crypto.Encrypt(key, m.encryptionKey)
		if err != nil {
			return ImportResult{}, m.internalError("encrypting key", err)
		}
		rows = append(rows, row{encrypted, hash})
	}
	if len(rows) == 0 {
		if result.Invalid > 0 || result.Duplicates > 0 {
			return result, nil
		}
		return result, inputError("Paste at least one key.")
	}

	tx, err := m.db.Beginx()
	if err != nil {
		return ImportResult{}, m.internalError("starting import", err)
	}
	defer tx.Rollback()
	var addedBy null.Int
	if userID > 0 {
		addedBy = null.IntFrom(userID)
	}
	for _, r := range rows {
		res, err := tx.Exec(`INSERT INTO activation_keys (app_id, key_encrypted, key_hash, added_by) VALUES ($1, $2, $3, $4)
			ON CONFLICT (key_hash) WHERE status <> 'voided' DO NOTHING`, appID, r.encrypted, r.hash, addedBy)
		if err != nil {
			return ImportResult{}, m.internalError("importing key", err)
		}
		if n, _ := res.RowsAffected(); n == 1 {
			result.Added++
		} else {
			result.Duplicates++
		}
	}
	if err := tx.Commit(); err != nil {
		return ImportResult{}, m.internalError("committing import", err)
	}
	return result, nil
}

// Reveal decrypts a single key for the admin page.
func (m *Manager) Reveal(id int64) (string, error) {
	var ciphertext string
	err := m.db.Get(&ciphertext, `SELECT key_encrypted FROM activation_keys WHERE id = $1 AND status IN ('redeemable', 'activated')`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", envelope.NewError(envelope.NotFoundError, "That key no longer exists.", nil)
	}
	if err != nil {
		return "", m.internalError("reading key", err)
	}
	key, err := m.decrypt(ciphertext)
	if err != nil {
		return "", m.internalError("decrypting key", err)
	}
	return key, nil
}

// Void removes a redeemable key from its pool. The row is kept as a record;
// activated keys can't be voided.
func (m *Manager) Void(id int64, userID int) error {
	res, err := m.db.Exec(`UPDATE activation_keys SET status = 'voided', voided_at = NOW(), voided_by = NULLIF($2, 0)
		WHERE id = $1 AND status = 'redeemable'`, id, userID)
	if err != nil {
		return m.internalError("voiding key", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return envelope.NewError(envelope.ConflictError, "Only keys that are still redeemable can be voided.", nil)
	}
	return nil
}

// --- Composer ---------------------------------------------------------------

// Composer returns the games the key card offers to userID.
func (m *Manager) Composer(userID int) (ComposerInfo, error) {
	s, err := m.GetSettings()
	if err != nil {
		return ComposerInfo{}, m.internalError("reading settings", err)
	}
	info := ComposerInfo{Enabled: s.Enabled, MaxKeysPerEmail: s.MaxKeysPerEmail, Apps: []ComposerApp{}}
	if !s.Enabled {
		return info, nil
	}
	if err := m.db.Select(&info.Apps, `SELECT a.id, a.name,
		EXISTS (SELECT 1 FROM activation_keys k WHERE k.app_id = a.id AND k.status = 'redeemable') AS available
		FROM activation_key_apps a WHERE a.archived_at IS NULL ORDER BY lower(a.name)`); err != nil {
		return ComposerInfo{}, m.internalError("listing composer apps", err)
	}
	err = m.db.Get(&info.LastAppID, `SELECT p.app_id FROM activation_key_preferences p
		JOIN activation_key_apps a ON a.id = p.app_id AND a.archived_at IS NULL WHERE p.user_id = $1`, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ComposerInfo{}, m.internalError("reading composer preference", err)
	}
	return info, nil
}

// SetLastApp remembers the game userID picked in the key card.
func (m *Manager) SetLastApp(userID, appID int) error {
	res, err := m.db.Exec(`INSERT INTO activation_key_preferences (user_id, app_id, updated_at)
		SELECT $1, id, NOW() FROM activation_key_apps WHERE id = $2 AND archived_at IS NULL
		ON CONFLICT (user_id) DO UPDATE SET app_id = EXCLUDED.app_id, updated_at = NOW()`, userID, appID)
	if err != nil {
		return m.internalError("saving composer preference", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return inputError("That game is no longer available for keys.")
	}
	return nil
}

// --- Sending ----------------------------------------------------------------

// Validate checks an email's placeholders against the settings without
// touching the pools: Key Distribution is on, the email is within the cap,
// and every placeholder is for the same available game.
func (m *Manager) Validate(placeholders []Placeholder) error {
	_, err := m.validate(m.db, placeholders)
	return err
}

func (m *Manager) validate(q sqlx.Queryer, placeholders []Placeholder) (string, error) {
	if len(placeholders) == 0 {
		return "", nil
	}
	s, err := m.settings(q)
	if err != nil {
		return "", m.internalError("reading settings", err)
	}
	if !s.Enabled {
		return "", inputError("Key distribution is turned off, so this email can't include activation keys. Remove the key placeholders to send it.")
	}
	if len(placeholders) > s.MaxKeysPerEmail {
		noun := "keys"
		if s.MaxKeysPerEmail == 1 {
			noun = "key"
		}
		return "", inputError(fmt.Sprintf("An email can include at most %d activation %s.", s.MaxKeysPerEmail, noun))
	}
	appID := placeholders[0].AppID
	for _, p := range placeholders[1:] {
		if p.AppID != appID {
			return "", inputError("All activation keys in an email must be for the same game.")
		}
	}
	var name string
	err = sqlx.Get(q, &name, `SELECT name FROM activation_key_apps WHERE id = $1 AND archived_at IS NULL`, appID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", inputError("The game chosen for this email's activation keys is no longer available.")
	}
	if err != nil {
		return "", m.internalError("reading app", err)
	}
	return name, nil
}

// AllocateTx moves one redeemable key per placeholder to the activated pool
// and binds it to the message, inside the transaction that inserts the
// message. Too few keys fails the whole insert.
func (m *Manager) AllocateTx(tx *sqlx.Tx, in Allocation) error {
	if len(in.Placeholders) == 0 {
		return nil
	}
	name, err := m.validate(tx, in.Placeholders)
	if err != nil {
		return err
	}
	appID := in.Placeholders[0].AppID
	var ids []int64
	if err := tx.Select(&ids, `SELECT id FROM activation_keys WHERE app_id = $1 AND status = 'redeemable'
		ORDER BY id LIMIT $2 FOR UPDATE SKIP LOCKED`, appID, len(in.Placeholders)); err != nil {
		return m.internalError("reserving keys", err)
	}
	if len(ids) < len(in.Placeholders) {
		return inputError(fmt.Sprintf("No keys available for %s.", name))
	}
	recipients := in.Recipients
	if recipients == nil {
		recipients = []string{}
	}
	nullID := func(id int) null.Int { return null.NewInt(id, id > 0) }
	for i, p := range in.Placeholders {
		res, err := tx.Exec(`UPDATE activation_keys SET status = 'activated', activated_at = NOW(), message_id = $2,
			conversation_id = $3, placeholder_id = $4, recipients = $5, sent_by = $6, approved_by = $7
			WHERE id = $1 AND status = 'redeemable'`,
			ids[i], in.MessageID, nullID(in.ConversationID), p.ID, pq.Array(recipients), nullID(in.SenderID), nullID(in.ApprovedBy))
		if err != nil {
			return m.internalError("activating key", err)
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return inputError(fmt.Sprintf("No keys available for %s.", name))
		}
	}
	return nil
}

// MessageKeys returns the decrypted keys bound to a message, by placeholder ID.
func (m *Manager) MessageKeys(messageID int) (map[string]string, error) {
	var rows []struct {
		PlaceholderID string `db:"placeholder_id"`
		Encrypted     string `db:"key_encrypted"`
	}
	if err := m.db.Select(&rows, `SELECT placeholder_id, key_encrypted FROM activation_keys
		WHERE message_id = $1 AND status = 'activated'`, messageID); err != nil {
		return nil, err
	}
	keys := make(map[string]string, len(rows))
	for _, row := range rows {
		key, err := m.decrypt(row.Encrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypting key for placeholder %s: %w", row.PlaceholderID, err)
		}
		if _, dup := keys[row.PlaceholderID]; dup {
			return nil, fmt.Errorf("placeholder %s is bound to more than one key", row.PlaceholderID)
		}
		keys[row.PlaceholderID] = key
	}
	return keys, nil
}

// MaskSent hides, in place, every sent key quoted in texts. Keys are matched
// by hash, so a key is hidden in any conversation it turns up in, and the
// pool is never decrypted to look for it.
func (m *Manager) MaskSent(texts ...*string) error {
	hashes := map[string]string{}
	for _, text := range texts {
		for _, key := range keyRuns(*text) {
			if _, ok := hashes[key]; !ok {
				hashes[key] = m.hash(key)
			}
		}
	}
	if len(hashes) == 0 {
		return nil
	}
	var sent []string
	// status <> 'voided' matches the unique key_hash index, so the lookup uses it.
	if err := m.db.Select(&sent, `SELECT key_hash FROM activation_keys
		WHERE key_hash = ANY($1) AND status <> 'voided' AND status = 'activated'`, pq.Array(slices.Collect(maps.Values(hashes)))); err != nil {
		return err
	}
	if len(sent) == 0 {
		return nil
	}
	for _, text := range texts {
		*text = maskRuns(*text, func(key string) bool { return slices.Contains(sent, hashes[key]) })
	}
	return nil
}

// AppNames maps app IDs to names, for labelling submissions in review.
func (m *Manager) AppNames(ids []int) (map[int]string, error) {
	names := map[int]string{}
	if len(ids) == 0 {
		return names, nil
	}
	var rows []struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}
	if err := m.db.Select(&rows, `SELECT id, name FROM activation_key_apps WHERE id = ANY($1)`, pq.Array(ids)); err != nil {
		return nil, err
	}
	for _, row := range rows {
		names[row.ID] = row.Name
	}
	return names, nil
}
