package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/httputil"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"golang.org/x/crypto/bcrypt"
)

const setupTokenLifetime = time.Hour

type setupAccount struct {
	Token     string `json:"token"`
	FirstName string `json:"first_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type setupChecklist struct {
	Managed       bool   `json:"managed" db:"managed"`
	Workspace     bool   `json:"workspace" db:"workspace"`
	Notifications bool   `json:"notifications" db:"notifications"`
	Inbox         bool   `json:"inbox" db:"inbox"`
	Conversation  string `json:"conversation" db:"conversation"`
}

func initializeSetup(tx *sqlx.Tx, required bool) error {
	_, err := tx.Exec(`INSERT INTO settings (key, value) VALUES
		('setup.bootstrap', jsonb_build_object('required', $1::boolean)),
		('setup.workspace_complete', 'false'),
		('setup.notifications_skipped', 'false')`, required)
	if err != nil {
		return err
	}
	if rootURL := strings.TrimRight(ko.String("app.root_url"), "/"); rootURL != "" {
		if !httputil.IsValidHTTPURL(rootURL) {
			return fmt.Errorf("app.root_url is invalid. Set your full address, including https://")
		}
		_, err = tx.Exec(`UPDATE settings SET value = to_jsonb($1::text) WHERE key = 'app.root_url'`, rootURL)
	}
	return err
}

func setupRequired(db *sqlx.DB) (bool, error) {
	var required bool
	err := db.Get(&required, `SELECT COALESCE((SELECT (value->>'required')::boolean FROM settings WHERE key = 'setup.bootstrap'), false)`)
	return required, err
}

// issueSetupToken rotates the claim token only for an installation that explicitly enabled browser setup.
func issueSetupToken(ctx context.Context, db *sqlx.DB) error {
	required, err := setupRequired(db)
	if err != nil || !required {
		return err
	}
	token, err := stringutil.RandomAlphanumeric(40)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(token))
	_, err = db.ExecContext(ctx, `UPDATE settings SET value = value || jsonb_build_object('token_hash', $1::text, 'expires', $2::bigint) WHERE key = 'setup.bootstrap' AND (value->>'required')::boolean`, hex.EncodeToString(hash[:]), time.Now().Add(setupTokenLifetime).Unix())
	if err != nil {
		return err
	}
	log.Printf("Create your admin account at /setup. Setup code (expires in 1 hour): %s", token)
	return nil
}

func handleGetSetup(r *fastglue.Request) error {
	app := r.Context.(*App)
	required, err := setupRequired(app.db)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("setup.unavailable"), err))
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	return r.SendEnvelope(map[string]bool{"required": required})
}

func handleCreateSetupAccount(r *fastglue.Request) error {
	app := r.Context.(*App)
	var req setupAccount
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}
	if err := createSetupAccount(r.RequestCtx, app.db, req); err != nil {
		if detail, ok := errors.AsType[envelope.Error](err); ok {
			detail.Message = app.i18n.T(detail.Message)
			err = detail
		}
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// createSetupAccount consumes the token and grants the first admin role in one transaction.
func createSetupAccount(ctx context.Context, db *sqlx.DB, req setupAccount) error {
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	address, emailErr := mail.ParseAddress(req.Email)
	if req.FirstName == "" || len(req.FirstName) > 100 || emailErr != nil || address.Address != req.Email || !user.IsStrongPassword(req.Password) {
		return envelope.NewError(envelope.InputError, "setup.accountInvalid", nil)
	}
	if len(req.Token) != 40 {
		return envelope.NewError(envelope.PermissionError, "setup.codeInvalid", nil)
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, "setup.saveFailed", err)
	}
	defer tx.Rollback()
	var state struct {
		Required bool   `db:"required"`
		Hash     string `db:"hash"`
		Expires  int64  `db:"expires"`
	}
	err = tx.GetContext(ctx, &state, `SELECT (value->>'required')::boolean AS required, COALESCE(value->>'token_hash', '') AS hash, COALESCE((value->>'expires')::bigint, 0) AS expires FROM settings WHERE key = 'setup.bootstrap' FOR UPDATE`)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !state.Required) {
		return envelope.NewError(envelope.ConflictError, "setup.alreadyComplete", nil)
	}
	if err != nil {
		return envelope.NewError(envelope.GeneralError, "setup.saveFailed", err)
	}
	hash := sha256.Sum256([]byte(req.Token))
	if state.Expires <= time.Now().Unix() || subtle.ConstantTimeCompare([]byte(state.Hash), []byte(hex.EncodeToString(hash[:]))) != 1 {
		return envelope.NewError(envelope.PermissionError, "setup.codeInvalid", nil)
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, "setup.saveFailed", err)
	}
	var id int
	if err := tx.GetContext(ctx, &id, `INSERT INTO users (email, type, first_name, last_name, password) VALUES ($1, 'agent', $2, '', $3) RETURNING id`, req.Email, req.FirstName, hashedPassword); err != nil {
		return envelope.NewError(envelope.GeneralError, "setup.accountFailed", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE name = 'Admin'`, id); err != nil {
		return envelope.NewError(envelope.GeneralError, "setup.saveFailed", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE settings SET value = '{"required":false}' WHERE key = 'setup.bootstrap'`); err != nil {
		return envelope.NewError(envelope.GeneralError, "setup.saveFailed", err)
	}
	if err := tx.Commit(); err != nil {
		return envelope.NewError(envelope.GeneralError, "setup.saveFailed", err)
	}
	return nil
}

func handleGetSetupChecklist(r *fastglue.Request) error {
	app := r.Context.(*App)
	out, err := getSetupChecklist(app.db)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("setup.unavailable"), err))
	}
	return r.SendEnvelope(out)
}

func handleSkipSetupNotifications(r *fastglue.Request) error {
	app := r.Context.(*App)
	if _, err := app.db.Exec(`UPDATE settings SET value = 'true' WHERE key = 'setup.notifications_skipped'`); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("setup.unavailable"), err))
	}
	return r.SendEnvelope(true)
}

func getSetupChecklist(db *sqlx.DB) (setupChecklist, error) {
	var out setupChecklist
	err := db.Get(&out, `SELECT
		EXISTS (SELECT 1 FROM settings WHERE key = 'setup.bootstrap') AS managed,
		COALESCE((SELECT value::boolean FROM settings WHERE key = 'setup.workspace_complete'), false) AS workspace,
		(COALESCE((SELECT value::boolean FROM settings WHERE key = 'notification.email.enabled'), false)
			OR COALESCE((SELECT value::boolean FROM settings WHERE key = 'setup.notifications_skipped'), false)) AS notifications,
		EXISTS (SELECT 1 FROM inboxes WHERE enabled) AS inbox,
		COALESCE((SELECT c.uuid::text FROM conversations c WHERE EXISTS (
			SELECT 1 FROM conversation_messages reply WHERE reply.conversation_id = c.id
			AND reply.type = 'outgoing' AND reply.status = 'sent' AND NOT reply.private AND reply.sender_type = 'agent'
			AND EXISTS (SELECT 1 FROM conversation_messages incoming WHERE incoming.conversation_id = c.id
				AND incoming.type = 'incoming' AND incoming.status = 'received' AND NOT incoming.private AND incoming.id < reply.id)
		) LIMIT 1), '') AS conversation`)
	return out, err
}

func markSetupWorkspaceComplete(db *sqlx.DB) error {
	if _, err := db.Exec(`UPDATE settings SET value = 'true' WHERE key = 'setup.workspace_complete'`); err != nil {
		return fmt.Errorf("saving setup progress: %w", err)
	}
	return nil
}
