// Package whatsapp_template manages WhatsApp templates stored locally and mirrored against Meta.
package whatsapp_template

import (
	"cmp"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/abhinavxd/libredesk/internal/whatsapp_template/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS

	ErrTemplateNotFound = errors.New("whatsapp template not found")
)

// AccountResolver returns the WhatsApp account credentials for an inbox.
type AccountResolver interface {
	WhatsAppAccount(inboxID int) (whatsapp.Account, error)
}

// Manager handles template CRUD and Meta sync.
type Manager struct {
	q        queries
	lo       *logf.Logger
	i18n     *i18n.I18n
	client   *whatsapp.Client
	resolver AccountResolver
}

type queries struct {
	Insert                     *sqlx.Stmt `query:"insert"`
	Update                     *sqlx.Stmt `query:"update"`
	UpdateStatus               *sqlx.Stmt `query:"update-status"`
	UpdateMetaID               *sqlx.Stmt `query:"update-meta-id"`
	Delete                     *sqlx.Stmt `query:"delete"`
	GetByID                    *sqlx.Stmt `query:"get-by-id"`
	GetByInbox                 *sqlx.Stmt `query:"get-by-inbox"`
	GetByNameLanguage          *sqlx.Stmt `query:"get-by-name-language"`
	UpsertFromMeta             *sqlx.Stmt `query:"upsert-from-meta"`
	UpdateStatusByNameLanguage *sqlx.Stmt `query:"update-status-by-meta-name-language"`
}

// Opts holds dependencies.
type Opts struct {
	Lo       *logf.Logger
	DB       *sqlx.DB
	I18n     *i18n.I18n
	Client   *whatsapp.Client
	Resolver AccountResolver
}

// New creates a Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:        q,
		lo:       opts.Lo,
		i18n:     opts.I18n,
		client:   opts.Client,
		resolver: opts.Resolver,
	}, nil
}

// GetByInbox returns all templates for a given inbox.
func (m *Manager) GetByInbox(inboxID int) ([]models.Template, error) {
	out := make([]models.Template, 0)
	if err := m.q.GetByInbox.Select(&out, inboxID); err != nil {
		m.lo.Error("error fetching whatsapp templates", "inbox_id", inboxID, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return out, nil
}

// GetByID returns a template by id.
func (m *Manager) GetByID(id int) (models.Template, error) {
	var t models.Template
	if err := m.q.GetByID.Get(&t, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, envelope.NewError(envelope.NotFoundError, m.i18n.T("globals.messages.notFound"), nil)
		}
		m.lo.Error("error fetching whatsapp template", "id", id, "error", err)
		return t, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return t, nil
}

// Exists reports whether a template with this name and language exists for the inbox, regardless of status.
func (m *Manager) Exists(inboxID int, name, language string) (bool, error) {
	var t models.Template
	if err := m.q.GetByNameLanguage.Get(&t, inboxID, name, language); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetApproved returns the approved template matching inbox + name + language.
func (m *Manager) GetApproved(inboxID int, name, language string) (models.Template, error) {
	var t models.Template
	if err := m.q.GetByNameLanguage.Get(&t, inboxID, name, language); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, ErrTemplateNotFound
		}
		return t, err
	}
	if !strings.EqualFold(t.Status, models.StatusApproved) {
		return t, fmt.Errorf("template %q (%s) is not approved (status: %s)", name, language, t.Status)
	}
	return t, nil
}

// Create stores a template locally, submits it to Meta and records the returned template id.
func (m *Manager) Create(ctx context.Context, t models.Template) (models.Template, error) {
	t.Status = cmp.Or(t.Status, models.StatusPending)
	if t.Buttons == nil {
		t.Buttons = json.RawMessage(`[]`)
	}
	if t.SampleValues == nil {
		t.SampleValues = json.RawMessage(`{}`)
	}

	var stored models.Template
	if err := m.q.Insert.Get(&stored,
		t.InboxID, t.MetaTemplateID, t.Name, t.Language, t.Category, t.Status,
		t.HeaderType, t.HeaderContent, t.BodyContent, t.FooterContent,
		t.Buttons, t.SampleValues, t.RejectionReason,
	); err != nil {
		m.lo.Error("error inserting whatsapp template", "error", err)
		if dbutil.IsUniqueViolationError(err) {
			return models.Template{}, envelope.NewError(envelope.ConflictError, m.i18n.T("globals.messages.errorAlreadyExists"), nil)
		}
		return models.Template{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if m.client == nil || m.resolver == nil {
		return stored, nil
	}

	acc, err := m.resolver.WhatsAppAccount(stored.InboxID)
	if err != nil {
		m.lo.Error("error resolving whatsapp account for template submit", "inbox_id", stored.InboxID, "error", err)
		reason := "could not resolve WhatsApp account for submission"
		if _, err := m.q.UpdateStatus.Exec(stored.ID, models.StatusRejected, reason); err != nil {
			m.lo.Error("error persisting template rejected status", "id", stored.ID, "error", err)
		}
		stored.Status = models.StatusRejected
		stored.RejectionReason = null.StringFrom(reason)
		return stored, nil
	}

	submission, err := buildSubmission(stored)
	if err != nil {
		m.lo.Error("error building template submission", "id", stored.ID, "error", err)
		reason := "could not build template submission: " + err.Error()
		if _, err := m.q.UpdateStatus.Exec(stored.ID, models.StatusRejected, reason); err != nil {
			m.lo.Error("error persisting template rejected status", "id", stored.ID, "error", err)
		}
		stored.Status = models.StatusRejected
		stored.RejectionReason = null.StringFrom(reason)
		return stored, nil
	}

	metaID, submitErr := m.client.SubmitTemplate(ctx, acc, submission)
	if submitErr != nil {
		m.lo.Error("error submitting template to meta", "id", stored.ID, "error", submitErr)
		if _, err := m.q.UpdateStatus.Exec(stored.ID, models.StatusRejected, submitErrReason(submitErr)); err != nil {
			m.lo.Error("error persisting template rejected status", "id", stored.ID, "error", err)
		}
		stored.Status = models.StatusRejected
		stored.RejectionReason = null.StringFrom(submitErrReason(submitErr))
		return stored, nil
	}

	if _, err := m.q.UpdateMetaID.Exec(stored.ID, metaID, models.StatusPending); err != nil {
		m.lo.Error("error persisting meta template id", "id", stored.ID, "error", err)
	}
	stored.MetaTemplateID = null.StringFrom(metaID)
	stored.Status = models.StatusPending
	return stored, nil
}

// Delete removes the template locally and on Meta (best-effort).
func (m *Manager) Delete(ctx context.Context, id int) error {
	t, err := m.GetByID(id)
	if err != nil {
		return err
	}
	if strings.HasPrefix(t.Name, models.CSATTemplateNamePrefix) {
		return envelope.NewError(envelope.InputError, "this template is reserved for CSAT surveys and cannot be deleted", nil)
	}
	if m.client != nil && m.resolver != nil {
		if acc, err := m.resolver.WhatsAppAccount(t.InboxID); err == nil {
			if err := m.client.DeleteTemplate(ctx, acc, t.Name); err != nil {
				m.lo.Error("error deleting template on meta", "id", id, "name", t.Name, "error", err)
			}
		}
	}
	if _, err := m.q.Delete.Exec(id); err != nil {
		m.lo.Error("error deleting whatsapp template", "id", id, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// SyncFromMeta pulls templates for an inbox from Meta and upserts them locally.
func (m *Manager) SyncFromMeta(ctx context.Context, inboxID int) (int, error) {
	if m.client == nil || m.resolver == nil {
		return 0, fmt.Errorf("whatsapp client not configured")
	}
	acc, err := m.resolver.WhatsAppAccount(inboxID)
	if err != nil {
		return 0, err
	}
	templates, err := m.client.FetchTemplates(ctx, acc)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, mt := range templates {
		row := metaToRow(inboxID, mt)
		var stored models.Template
		if err := m.q.UpsertFromMeta.Get(&stored,
			row.InboxID, row.MetaTemplateID, row.Name, row.Language, row.Category, row.Status,
			row.HeaderType, row.HeaderContent, row.BodyContent, row.FooterContent,
			row.Buttons, row.SampleValues, row.RejectionReason,
		); err != nil {
			m.lo.Error("error upserting template from meta", "name", mt.Name, "error", err)
			continue
		}
		count++
	}
	return count, nil
}

// HandleStatusUpdate processes a Meta template status webhook.
func (m *Manager) HandleStatusUpdate(inboxID int, name, language, event, reason string) error {
	if name == "" {
		return fmt.Errorf("missing template name in status update")
	}
	if language == "" {
		language = "en_US"
	}
	status := mapTemplateEventToStatus(event)
	if status == "" {
		m.lo.Info("ignoring unhandled whatsapp template status event", "name", name, "language", language, "event", event)
		return nil
	}
	if _, err := m.q.UpdateStatusByNameLanguage.Exec(inboxID, name, status, reason, language); err != nil {
		m.lo.Error("error applying template status update", "name", name, "language", language, "error", err)
		return err
	}
	return nil
}

func metaToRow(inboxID int, mt whatsapp.MetaTemplate) models.Template {
	row := models.Template{
		InboxID:        inboxID,
		MetaTemplateID: null.StringFrom(mt.ID),
		Name:           mt.Name,
		Language:       mt.Language,
		Category:       mt.Category,
		Status:         strings.ToUpper(mt.Status),
	}
	for _, c := range mt.Components {
		switch strings.ToUpper(c.Type) {
		case "HEADER":
			if c.Format != "" {
				row.HeaderType = null.StringFrom(strings.ToUpper(c.Format))
			}
			if c.Text != "" {
				row.HeaderContent = null.StringFrom(c.Text)
			}
		case "BODY":
			row.BodyContent = c.Text
		case "FOOTER":
			if c.Text != "" {
				row.FooterContent = null.StringFrom(c.Text)
			}
		case "BUTTONS":
			if b, err := json.Marshal(c.Buttons); err == nil {
				row.Buttons = b
			}
		}
	}
	if mt.RejectedReason != "" {
		row.RejectionReason = null.StringFrom(mt.RejectedReason)
	}
	if row.Buttons == nil {
		row.Buttons = json.RawMessage(`[]`)
	}
	if row.SampleValues == nil {
		row.SampleValues = json.RawMessage(`{}`)
	}
	return row
}

func buildSubmission(t models.Template) (whatsapp.TemplateSubmission, error) {
	sub := whatsapp.TemplateSubmission{
		Name:     t.Name,
		Language: t.Language,
		Category: strings.ToUpper(t.Category),
	}

	sampleValues := parseSampleValues(t.SampleValues)

	headerText := ""
	if t.HeaderType.Valid && strings.EqualFold(t.HeaderType.String, "TEXT") && t.HeaderContent.Valid {
		headerText = t.HeaderContent.String
	}
	named := isNamed(headerText) || isNamed(t.BodyContent)
	if named {
		sub.ParameterFormat = "NAMED"
	}

	if t.HeaderType.Valid && t.HeaderType.String != "" {
		hdr := whatsapp.TemplateComponent{
			Type:   "HEADER",
			Format: strings.ToUpper(t.HeaderType.String),
		}
		if hdr.Format == "TEXT" && t.HeaderContent.Valid {
			hdr.Text = t.HeaderContent.String
			ex, err := buildExample(hdr.Text, sampleValues, named, "header_text")
			if err != nil {
				return whatsapp.TemplateSubmission{}, err
			}
			hdr.Example = ex
		}
		sub.Components = append(sub.Components, hdr)
	}

	body := whatsapp.TemplateComponent{Type: "BODY", Text: t.BodyContent}
	ex, err := buildExample(body.Text, sampleValues, named, "body_text")
	if err != nil {
		return whatsapp.TemplateSubmission{}, err
	}
	body.Example = ex
	sub.Components = append(sub.Components, body)

	if t.FooterContent.Valid && t.FooterContent.String != "" {
		sub.Components = append(sub.Components, whatsapp.TemplateComponent{
			Type: "FOOTER",
			Text: t.FooterContent.String,
		})
	}

	if len(t.Buttons) > 0 && string(t.Buttons) != "[]" {
		var btns []whatsapp.TemplateButton
		if err := json.Unmarshal(t.Buttons, &btns); err == nil && len(btns) > 0 {
			for i := range btns {
				if !strings.EqualFold(btns[i].Type, "URL") || len(btns[i].Example) > 0 {
					continue
				}
				keys := whatsapp.OrderedPlaceholders(btns[i].URL)
				if len(keys) == 0 {
					continue
				}
				url, err := substitutePlaceholders(btns[i].URL, keys, sampleValues)
				if err != nil {
					return whatsapp.TemplateSubmission{}, err
				}
				btns[i].Example = []string{url}
			}
			sub.Components = append(sub.Components, whatsapp.TemplateComponent{
				Type:    "BUTTONS",
				Buttons: btns,
			})
		}
	}

	return sub, nil
}

// parseSampleValues decodes sample_values JSON, tolerating non-string values from the frontend.
func parseSampleValues(raw json.RawMessage) map[string]string {
	if len(raw) == 0 || string(raw) == "{}" {
		return nil
	}
	var anyMap map[string]any
	if err := json.Unmarshal(raw, &anyMap); err != nil {
		return nil
	}
	out := make(map[string]string, len(anyMap))
	for k, v := range anyMap {
		switch t := v.(type) {
		case string:
			out[k] = t
		case float64:
			out[k] = fmt.Sprintf("%v", t)
		case bool:
			out[k] = fmt.Sprintf("%v", t)
		}
	}
	return out
}

func isNamed(text string) bool {
	for _, key := range whatsapp.OrderedPlaceholders(text) {
		if _, err := strconv.Atoi(key); err != nil {
			return true
		}
	}
	return false
}

func buildExample(text string, samples map[string]string, named bool, positionalKey string) (map[string]any, error) {
	keys := whatsapp.OrderedPlaceholders(text)
	if len(keys) == 0 {
		return nil, nil
	}
	if named {
		params := make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			v, err := sampleValue(samples, key)
			if err != nil {
				return nil, err
			}
			params = append(params, map[string]any{"param_name": key, "example": v})
		}
		return map[string]any{positionalKey + "_named_params": params}, nil
	}
	vals := make([]string, 0, len(keys))
	for _, key := range keys {
		v, err := sampleValue(samples, key)
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	if positionalKey == "body_text" {
		return map[string]any{positionalKey: [][]string{vals}}, nil
	}
	return map[string]any{positionalKey: vals}, nil
}

func substitutePlaceholders(text string, keys []string, samples map[string]string) (string, error) {
	out := text
	for _, key := range keys {
		v, err := sampleValue(samples, key)
		if err != nil {
			return "", err
		}
		out = strings.ReplaceAll(out, "{{"+key+"}}", v)
	}
	return out, nil
}

func sampleValue(samples map[string]string, key string) (string, error) {
	if v, ok := samples[key]; ok && v != "" {
		return v, nil
	}
	return "", fmt.Errorf("missing sample value for placeholder {{%s}}", key)
}

func submitErrReason(err error) string {
	if err == nil {
		return ""
	}
	if me, ok := err.(*whatsapp.MetaAPIError); ok {
		if me.UserMsg != "" {
			return me.UserMsg
		}
		return me.Message
	}
	return err.Error()
}

// mapTemplateEventToStatus maps a Meta event to a local status; REINSTATED is an event, not a status, that means approved again.
func mapTemplateEventToStatus(event string) string {
	switch strings.ToUpper(event) {
	case "APPROVED", "REINSTATED":
		return models.StatusApproved
	case "REJECTED":
		return models.StatusRejected
	case "PAUSED":
		return models.StatusPaused
	case "DISABLED":
		return models.StatusDisabled
	default:
		return ""
	}
}
