package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/role"
	"github.com/abhinavxd/libredesk/internal/role/models"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetRoles returns all roles
func handleGetRoles(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	roles, err := app.role.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(roles)
}

// handleGetRole returns a single role
func handleGetRole(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	role, err := app.role.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(role)
}

// handleDeleteRole deletes a role
func handleDeleteRole(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err := app.role.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Invalidate all caches when role is deleted.
	app.user.InvalidateAllAgentCache()
	app.authz.InvalidateAllCache()

	return r.SendEnvelope(true)
}

// handleCreateRole creates a new role
func handleCreateRole(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.Role{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}
	createdRole, err := app.role.Create(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(createdRole)
}

// handleUpdateRole updates a role
func handleUpdateRole(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		ip    = realip.FromRequest(r.RequestCtx)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		req   = models.Role{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Get old role before update to compare permissions.
	oldRole, err := app.role.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	updatedRole, err := app.role.Update(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Log permission changes and invalidate caches.
	added, removed := role.ComparePermissions(oldRole.Permissions, updatedRole.Permissions)
	if len(added) > 0 || len(removed) > 0 {
		// Invalidate all caches when role permissions change.
		app.user.InvalidateAllAgentCache()
		app.authz.InvalidateAllCache()

		if err := app.activityLog.RolePermissionsChanged(auser.ID, auser.Email, ip, updatedRole.ID, updatedRole.Name, added, removed); err != nil {
			app.lo.Error("error creating activity log", "error", err)
		}
	}

	return r.SendEnvelope(updatedRole)
}
