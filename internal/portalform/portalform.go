// Package portalform manages the ticket forms rendered on the customer portal.
package portalform

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/portalform/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

const (
	maxFields      = 25
	maxOptions     = 50
	maxLabelLength = 140
)

var (
	//go:embed queries.sql
	efs embed.FS

	fieldKeyRe = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

	fieldTypes = map[string]struct{}{
		models.FieldTypeText:     {},
		models.FieldTypeTextarea: {},
		models.FieldTypeSelect:   {},
		models.FieldTypeCheckbox: {},
		models.FieldTypeNumber:   {},
		models.FieldTypeDate:     {},
		models.FieldTypeEmail:    {},
		models.FieldTypeLink:     {},
	}
)

// Manager manages portal ticket forms.
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
	GetAll *sqlx.Stmt `query:"get-all"`
	Get    *sqlx.Stmt `query:"get"`
	Insert *sqlx.Stmt `query:"insert"`
	Update *sqlx.Stmt `query:"update"`
	Delete *sqlx.Stmt `query:"delete"`
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

func (m *Manager) GetAll() ([]models.Form, error) {
	var forms = make([]models.Form, 0)
	if err := m.q.GetAll.Select(&forms); err != nil {
		m.lo.Error("error fetching portal forms", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for i := range forms {
		m.decodeFields(&forms[i])
	}
	return forms, nil
}

func (m *Manager) Get(id int) (models.Form, error) {
	var form models.Form
	if err := m.q.Get.Get(&form, id); err != nil {
		if err == sql.ErrNoRows {
			return form, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching portal form", "id", id, "error", err)
		return form, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.decodeFields(&form)
	return form, nil
}

func (m *Manager) Create(form models.Form) (models.Form, error) {
	fields, err := m.validate(form)
	if err != nil {
		return models.Form{}, err
	}
	var out models.Form
	if err := m.q.Insert.Get(&out, form.Name, form.AskSubject, fields); err != nil {
		m.lo.Error("error inserting portal form", "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.decodeFields(&out)
	return out, nil
}

func (m *Manager) Update(id int, form models.Form) (models.Form, error) {
	fields, err := m.validate(form)
	if err != nil {
		return models.Form{}, err
	}
	var out models.Form
	if err := m.q.Update.Get(&out, id, form.Name, form.AskSubject, fields); err != nil {
		if err == sql.ErrNoRows {
			return out, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error updating portal form", "id", id, "error", err)
		return out, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.decodeFields(&out)
	return out, nil
}

// Articles referencing the form fall back to the portal default via ON DELETE SET NULL.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.Delete.Exec(id); err != nil {
		m.lo.Error("error deleting portal form", "id", id, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// validate returns the fields marshalled ready to store.
func (m *Manager) validate(form models.Form) ([]byte, error) {
	if strings.TrimSpace(form.Name) == "" {
		return nil, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.required", "name", "{globals.terms.name}"), nil)
	}
	if len(form.Fields) > maxFields {
		return nil, envelope.NewError(envelope.InputError, m.i18n.T("admin.portalForm.tooManyFields"), nil)
	}

	seen := make(map[string]struct{}, len(form.Fields))
	for i, f := range form.Fields {
		if !fieldKeyRe.MatchString(f.Key) {
			return nil, envelope.NewError(envelope.InputError, m.i18n.T("admin.portalForm.invalidFieldKey"), nil)
		}
		if _, dup := seen[f.Key]; dup {
			return nil, envelope.NewError(envelope.InputError, m.i18n.T("admin.portalForm.duplicateFieldKey"), nil)
		}
		seen[f.Key] = struct{}{}

		if strings.TrimSpace(f.Label) == "" || len(f.Label) > maxLabelLength {
			return nil, envelope.NewError(envelope.InputError, m.i18n.T("admin.portalForm.invalidFieldLabel"), nil)
		}
		if _, ok := fieldTypes[f.Type]; !ok {
			return nil, envelope.NewError(envelope.InputError, m.i18n.T("admin.portalForm.invalidFieldType"), nil)
		}
		if f.Type == models.FieldTypeSelect && (len(f.Options) == 0 || len(f.Options) > maxOptions) {
			return nil, envelope.NewError(envelope.InputError, m.i18n.T("admin.portalForm.invalidFieldOptions"), nil)
		}
		switch f.Target {
		case models.TargetMessage:
			form.Fields[i].AttributeKey = ""
		case models.TargetAttribute:
			if f.AttributeKey == "" {
				return nil, envelope.NewError(envelope.InputError, m.i18n.T("admin.portalForm.attributeRequired"), nil)
			}
		default:
			return nil, envelope.NewError(envelope.InputError, m.i18n.T("admin.portalForm.invalidFieldTarget"), nil)
		}
	}

	b, err := json.Marshal(form.Fields)
	if err != nil {
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return b, nil
}

func (m *Manager) decodeFields(form *models.Form) {
	form.Fields = make([]models.Field, 0)
	if len(form.FieldsJSON) == 0 {
		return
	}
	if err := json.Unmarshal(form.FieldsJSON, &form.Fields); err != nil {
		m.lo.Error("error decoding portal form fields", "id", form.ID, "error", err)
	}
}

// RenderHeaderBlock formats message-target answers as the block above the contact's text.
func RenderHeaderBlock(lines [][2]string) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for _, l := range lines {
		fmt.Fprintf(&b, "%s: %s\n", l[0], l[1])
	}
	return b.String()
}
