// Package organization handles the management of organizations.
package organization

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/organization/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

const (
	maxListPageSize = 100
)

var (
	//go:embed queries.sql
	efs embed.FS

	// Simple domain validation regex
	domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
)

type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
	db   *sqlx.DB
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetOrganization              *sqlx.Stmt `query:"get-organization"`
	InsertOrganization           *sqlx.Stmt `query:"insert-organization"`
	UpdateOrganization           *sqlx.Stmt `query:"update-organization"`
	DeleteOrganization           *sqlx.Stmt `query:"delete-organization"`
	GetOrganizationContactsCount *sqlx.Stmt `query:"get-organization-contacts-count"`
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
		db:   opts.DB,
	}, nil
}

// GetAll retrieves all organizations with pagination and filtering.
func (o *Manager) GetAll(page, pageSize int, order, orderBy, filtersJSON string) ([]models.Organization, error) {
	if pageSize > maxListPageSize {
		return nil, envelope.NewError(envelope.InputError, o.i18n.Ts("globals.messages.pageTooLarge", "max", fmt.Sprintf("%d", maxListPageSize)), nil)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	query, qArgs, err := o.makeOrganizationListQuery(page, pageSize, order, orderBy, filtersJSON)
	if err != nil {
		o.lo.Error("error creating organization list query", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorFetching", "name", "organization"), nil)
	}

	// Start a read-only txn.
	tx, err := o.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		o.lo.Error("error starting read-only transaction", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorFetching", "name", "organization"), nil)
	}
	defer tx.Rollback()

	// Execute query
	var organizations = make([]models.Organization, 0)
	if err := tx.Select(&organizations, query, qArgs...); err != nil {
		o.lo.Error("error fetching organizations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorFetching", "name", "organization"), nil)
	}

	return organizations, nil
}

// Get retrieves an organization by ID.
func (o *Manager) Get(id string) (models.Organization, error) {
	var org models.Organization
	if err := o.q.GetOrganization.Get(&org, id); err != nil {
		if err == sql.ErrNoRows {
			return org, envelope.NewError(envelope.NotFoundError, o.i18n.Ts("globals.messages.notFound", "name", "Organization"), nil)
		}
		o.lo.Error("error fetching organization", "error", err, "id", id)
		return org, envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorFetching", "name", "organization"), nil)
	}
	return org, nil
}

// Create creates a new organization.
func (o *Manager) Create(org models.Organization) (models.Organization, error) {
	// Validate required fields
	if strings.TrimSpace(org.Name) == "" {
		return org, envelope.NewError(envelope.InputError, o.i18n.Ts("globals.messages.empty", "name", "Name"), nil)
	}

	// Validate email domain if provided
	if org.EmailDomain.Valid && org.EmailDomain.String != "" {
		if !domainRegex.MatchString(org.EmailDomain.String) {
			return org, envelope.NewError(envelope.InputError, "Invalid email domain format", nil)
		}
	}

	// Validate website URL if provided
	if org.Website.Valid && org.Website.String != "" {
		if _, err := url.ParseRequestURI(org.Website.String); err != nil {
			return org, envelope.NewError(envelope.InputError, "Invalid website URL format", nil)
		}
	}

	var result models.Organization
	if err := o.q.InsertOrganization.Get(&result, org.Name, org.Website, org.EmailDomain, org.Phone); err != nil {
		o.lo.Error("error inserting organization", "error", err)
		return result, envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorCreating", "name", "organization"), nil)
	}

	return result, nil
}

// Update updates an organization by ID.
func (o *Manager) Update(id string, org models.Organization) (models.Organization, error) {
	// Validate required fields
	if strings.TrimSpace(org.Name) == "" {
		return org, envelope.NewError(envelope.InputError, o.i18n.Ts("globals.messages.empty", "name", "Name"), nil)
	}

	// Validate email domain if provided
	if org.EmailDomain.Valid && org.EmailDomain.String != "" {
		if !domainRegex.MatchString(org.EmailDomain.String) {
			return org, envelope.NewError(envelope.InputError, "Invalid email domain format", nil)
		}
	}

	// Validate website URL if provided
	if org.Website.Valid && org.Website.String != "" {
		if _, err := url.ParseRequestURI(org.Website.String); err != nil {
			return org, envelope.NewError(envelope.InputError, "Invalid website URL format", nil)
		}
	}

	var result models.Organization
	if err := o.q.UpdateOrganization.Get(&result, id, org.Name, org.Website, org.EmailDomain, org.Phone); err != nil {
		if err == sql.ErrNoRows {
			return result, envelope.NewError(envelope.NotFoundError, o.i18n.Ts("globals.messages.notFound", "name", "Organization"), nil)
		}
		o.lo.Error("error updating organization", "error", err, "id", id)
		return result, envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorUpdating", "name", "organization"), nil)
	}

	return result, nil
}

// Delete deletes an organization by ID.
// Returns an error if there are contacts assigned to this organization.
func (o *Manager) Delete(id string) error {
	// Check if organization has any contacts
	var count int
	if err := o.q.GetOrganizationContactsCount.Get(&count, id); err != nil {
		o.lo.Error("error checking organization contacts count", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorDeleting", "name", "organization"), nil)
	}

	if count > 0 {
		return envelope.NewError(envelope.ConflictError, fmt.Sprintf("Cannot delete organization with %d assigned contact(s). Please reassign or remove contacts first.", count), nil)
	}

	if _, err := o.q.DeleteOrganization.Exec(id); err != nil {
		if err == sql.ErrNoRows {
			return envelope.NewError(envelope.NotFoundError, o.i18n.Ts("globals.messages.notFound", "name", "Organization"), nil)
		}
		o.lo.Error("error deleting organization", "error", err, "id", id)
		return envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorDeleting", "name", "organization"), nil)
	}

	return nil
}

// makeOrganizationListQuery builds a paginated query for fetching organizations with filters.
func (o *Manager) makeOrganizationListQuery(page, pageSize int, order, orderBy, filtersJSON string) (string, []interface{}, error) {
	// Base query with pagination support
	baseQuery := `
		SELECT COUNT(*) OVER() as total, id, created_at, updated_at, name, website, email_domain, phone
		FROM organizations
	`

	var qArgs []any
	return dbutil.BuildPaginatedQuery(baseQuery, qArgs, dbutil.PaginationOptions{
		Order:    order,
		OrderBy:  orderBy,
		Page:     page,
		PageSize: pageSize,
	}, filtersJSON, dbutil.AllowedFields{
		"organizations": {"name", "email_domain", "created_at", "updated_at"},
	})
}

// GetCompact retrieves all organizations in a compact form (for dropdowns).
func (o *Manager) GetCompact() ([]models.OrganizationCompact, error) {
	query := `SELECT id, name FROM organizations ORDER BY name ASC`
	var organizations = make([]models.OrganizationCompact, 0)
	if err := o.db.Select(&organizations, query); err != nil {
		o.lo.Error("error fetching compact organizations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, o.i18n.Ts("globals.messages.errorFetching", "name", "organization"), nil)
	}
	return organizations, nil
}
