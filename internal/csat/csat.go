// Package csat contains the logic for managing CSAT.
package csat

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/csat/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs                  embed.FS
	ErrCSATAlreadyExists = errors.New("CSAT already exists")
)

const (
	csatURL = "%s/csat/%s"
)

// Manager manages CSAT.
type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	Insert *sqlx.Stmt `query:"insert"`
	Get    *sqlx.Stmt `query:"get"`
	Update *sqlx.Stmt `query:"update"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// Create creates a new CSAT for the given conversation ID.
func (m *Manager) Create(conversationID int) (models.CSATResponse, error) {
	var (
		uuid string
		rsp  models.CSATResponse
	)
	if err := m.q.Insert.QueryRow(conversationID).Scan(&uuid); err != nil {
		m.lo.Error("error creating CSAT", "error", err)
		return rsp, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.csatSurvey}"), nil)
	}
	return m.Get(uuid)
}

// Get retrieves the CSAT for the given UUID.
func (m *Manager) Get(uuid string) (models.CSATResponse, error) {
	var csat models.CSATResponse
	err := m.q.Get.Get(&csat, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return csat, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.csatSurvey}"), nil)
		}
		m.lo.Error("error getting CSAT", "error", err)
		return csat, err
	}
	return csat, nil
}

// UpdateResponse updates the CSAT response for the given csat.
func (m *Manager) UpdateResponse(uuid string, score int, feedback string) error {
	csat, err := m.Get(uuid)
	if err != nil {
		return err
	}

	if csat.Rating > 0 || !csat.ResponseTimestamp.IsZero() {
		return envelope.NewError(envelope.InputError, m.i18n.T("csat.alreadySubmitted"), nil)
	}

	_, err = m.q.Update.Exec(uuid, score, feedback)
	if err != nil {
		m.lo.Error("error updating CSAT", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorSaving", "name", "{globals.terms.csatResponse}"), nil)
	}
	return nil
}

// MakePublicURL returns the public URL for the given CSAT UUID.
func (m *Manager) MakePublicURL(appBaseURL, uuid string) string {
	return fmt.Sprintf(csatURL, appBaseURL, uuid)
}
