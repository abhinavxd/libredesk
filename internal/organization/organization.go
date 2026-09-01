package organization

import (
	"database/sql"
	"embed"
	"errors"
	"strings"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/organization/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	lo   *logf.Logger
	i18n *i18n.I18n
	q    queries
}

type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

type queries struct {
	GetAll              *sqlx.Stmt `query:"get-all"`
	Get                 *sqlx.Stmt `query:"get"`
	Insert              *sqlx.Stmt `query:"insert"`
	Update              *sqlx.Stmt `query:"update"`
	Delete              *sqlx.Stmt `query:"delete"`
	FindByDomain        *sqlx.Stmt `query:"find-by-domain"`
	AssignIfEmpty       *sqlx.Stmt `query:"assign-if-empty"`
	SetUserOrganization *sqlx.Stmt `query:"set-user-organization"`
}

func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{q: q, lo: opts.Lo, i18n: opts.I18n}, nil
}

func (m *Manager) GetAll() ([]models.Organization, error) {
	out := make([]models.Organization, 0)
	if err := m.q.GetAll.Select(&out); err != nil && !errors.Is(err, sql.ErrNoRows) {
		m.lo.Error("error fetching organizations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return out, nil
}

func (m *Manager) Get(id int) (models.Organization, error) {
	var org models.Organization
	if err := m.q.Get.Get(&org, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return org, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching organization", "error", err)
		return org, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return org, nil
}

func (m *Manager) Create(org models.Organization) (models.Organization, error) {
	if strings.TrimSpace(org.Name) == "" {
		return models.Organization{}, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.required"), nil)
	}
	var out models.Organization
	if err := m.q.Insert.Get(&out, org.Name, pq.Array(normalizeDomains(org.Domains)), org.Notes, org.ExternalID.String); err != nil {
		m.lo.Error("error creating organization", "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return out, nil
}

func (m *Manager) Update(id int, org models.Organization) (models.Organization, error) {
	if strings.TrimSpace(org.Name) == "" {
		return models.Organization{}, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.required"), nil)
	}
	var out models.Organization
	if err := m.q.Update.Get(&out, id, org.Name, pq.Array(normalizeDomains(org.Domains)), org.Notes, org.ExternalID.String); err != nil {
		m.lo.Error("error updating organization", "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return out, nil
}

func (m *Manager) Delete(id int) error {
	if _, err := m.q.Delete.Exec(id); err != nil {
		m.lo.Error("error deleting organization", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func (m *Manager) AssignByEmail(userID int, email string) {
	domain := emailDomain(email)
	if userID <= 0 || domain == "" {
		return
	}
	var org models.Organization
	if err := m.q.FindByDomain.Get(&org, domain); err != nil {
		return
	}
	if _, err := m.q.AssignIfEmpty.Exec(userID, org.ID); err != nil {
		m.lo.Error("error assigning organization by domain", "error", err, "user_id", userID)
	}
}

func (m *Manager) SetUserOrganization(userID, orgID int) error {
	var id any
	if orgID > 0 {
		id = orgID
	}
	if _, err := m.q.SetUserOrganization.Exec(userID, id); err != nil {
		m.lo.Error("error setting user organization", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func normalizeDomains(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, d := range in {
		d = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(d, "@")))
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

func emailDomain(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	i := strings.LastIndex(email, "@")
	if i < 0 || i == len(email)-1 {
		return ""
	}
	return email[i+1:]
}
