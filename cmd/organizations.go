package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	omodels "github.com/abhinavxd/libredesk/internal/organization/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

// handleGetOrganizations returns a list of organizations with pagination and filtering.
func handleGetOrganizations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)

	organizations, err := app.organization.GetAll(page, pageSize, order, orderBy, filters)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if len(organizations) > 0 {
		total = organizations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    organizations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetOrganization returns a single organization by ID.
func handleGetOrganization(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		id  = r.RequestCtx.UserValue("id").(string)
	)

	if id == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	org, err := app.organization.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(org)
}

// handleCreateOrganization creates a new organization.
func handleCreateOrganization(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		org = omodels.Organization{}
	)

	if err := r.Decode(&org, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	createdOrg, err := app.organization.Create(org)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(createdOrg)
}

// handleUpdateOrganization updates an existing organization.
func handleUpdateOrganization(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		id  = r.RequestCtx.UserValue("id").(string)
		org = omodels.Organization{}
	)

	if id == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&org, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	updatedOrg, err := app.organization.Update(id, org)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(updatedOrg)
}

// handleDeleteOrganization deletes an organization.
func handleDeleteOrganization(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		id  = r.RequestCtx.UserValue("id").(string)
	)

	if id == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := app.organization.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleGetOrganizationsCompact returns a compact list of all organizations (for dropdowns).
func handleGetOrganizationsCompact(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)

	organizations, err := app.organization.GetCompact()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(organizations)
}
