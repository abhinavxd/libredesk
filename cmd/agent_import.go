package main

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleImportAgents handles CSV upload and starts import job
func handleImportAgents(r *fastglue.Request) error {
	var app = r.Context.(*App)

	file, err := r.RequestCtx.FormFile("file")
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.file}"), nil, envelope.InputError)
	}

	fileContent, err := file.Open()
	if err != nil {
		app.lo.Error("error opening uploaded file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}
	defer fileContent.Close()

	reader := csv.NewReader(fileContent)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		app.lo.Error("error parsing CSV", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidCsvFile"), nil, envelope.InputError)
	}

	if len(records) < 2 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("importer.csvMustContainHeadersAndData"), nil, envelope.InputError)
	}

	err = app.importer.Submit("agents", func() error {
		return processAgentImport(app, records)
	})

	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, app.i18n.T("importer.importAlreadyInProgress"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(true)
}

// handleGetAgentImportStatus returns current import status
func handleGetAgentImportStatus(r *fastglue.Request) error {
	var app = r.Context.(*App)
	status, err := app.importer.GetStatus("agents")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(status)
}

func processAgentImport(app *App, records [][]string) error {
	// Parse headers
	headerMap := make(map[string]int)
	for i, h := range records[0] {
		headerMap[strings.TrimSpace(strings.ToLower(h))] = i
	}

	// Validate required columns
	required := []string{"first_name", "last_name", "email", "roles"}
	for _, col := range required {
		if _, ok := headerMap[col]; !ok {
			return fmt.Errorf("missing required column: %s", col)
		}
	}

	// Fetch valid teams and roles once
	allTeams, err := app.team.GetAll()
	if err != nil {
		return fmt.Errorf("failed to fetch teams: %v", err)
	}

	allRoles, err := app.role.GetAll()
	if err != nil {
		return fmt.Errorf("failed to fetch roles: %v", err)
	}

	validTeams := make(map[string]bool)
	for _, t := range allTeams {
		validTeams[t.Name] = true
	}

	validRoles := make(map[string]bool)
	for _, r := range allRoles {
		validRoles[r.Name] = true
	}

	// Initialize import
	total := len(records) - 1
	app.importer.UpdateCounts("agents", total, 0, 0)
	app.importer.AddLog("agents", app.i18n.Ts("importer.startingImport",
		"count", fmt.Sprintf("%d", total),
		"type", "agents"))

	// Process each row
	for i, record := range records[1:] {
		rowNum := i + 1
		rowStr := fmt.Sprintf("%d", rowNum)

		// Parse fields
		firstName := getField(record, headerMap, "first_name")
		lastName := getField(record, headerMap, "last_name")
		email := strings.TrimSpace(strings.ToLower(getField(record, headerMap, "email")))
		rolesStr := getField(record, headerMap, "roles")
		teamsStr := getField(record, headerMap, "teams")

		// Validate required fields
		var missing []string
		if firstName == "" {
			missing = append(missing, "first_name")
		}
		if lastName == "" {
			missing = append(missing, "last_name")
		}
		if email == "" {
			missing = append(missing, "email")
		}
		if rolesStr == "" {
			missing = append(missing, "roles")
		}
		if len(missing) > 0 {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Ts("importer.missingFields",
				"row", rowStr,
				"fields", strings.Join(missing, ", ")))
			continue
		}

		// Validate email format
		if !stringutil.ValidEmail(email) {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Ts("importer.invalidEmail",
				"row", rowStr,
				"email", email))
			continue
		}

		// Parse and validate roles
		roles := parseList(rolesStr)
		if len(roles) == 0 {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Ts("importer.roleRequired",
				"row", rowStr,
				"email", email))
			continue
		}

		invalidRoles := findInvalid(roles, validRoles)
		if len(invalidRoles) > 0 {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Ts("importer.invalidRoles",
				"row", rowStr,
				"email", email,
				"roles", strings.Join(invalidRoles, ", ")))
			continue
		}

		// Parse and validate teams (optional)
		teams := parseList(teamsStr)
		if len(teams) > 0 {
			invalidTeams := findInvalid(teams, validTeams)
			if len(invalidTeams) > 0 {
				app.importer.UpdateCounts("agents", 0, 0, 1)
				app.importer.AddLog("agents", app.i18n.Ts("importer.invalidTeams",
					"row", rowStr,
					"email", email,
					"teams", strings.Join(invalidTeams, ", ")))
				continue
			}
		}

		// Check if agent already exists
		if _, err := app.user.GetAgent(0, email); err == nil {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Ts("importer.emailExists",
				"row", rowStr,
				"email", email))
			continue
		}

		// Create agent
		agent, err := app.user.CreateAgent(firstName, lastName, email, roles)
		if err != nil {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Ts("importer.errorCreating",
				"row", rowStr,
				"email", email,
				"error", err.Error()))
			continue
		}

		// Assign teams (if provided)
		if len(teams) > 0 {
			if err := app.team.UpsertUserTeams(agent.ID, teams); err != nil {
				app.importer.UpdateCounts("agents", 0, 0, 1)
				app.importer.AddLog("agents", app.i18n.Ts("importer.errorAssigningTeams",
					"row", rowStr,
					"email", email,
					"error", err.Error()))
				continue
			}
		}

		app.importer.UpdateCounts("agents", 0, 1, 0)
		app.importer.AddLog("agents", app.i18n.Ts("importer.createdAgent",
			"row", rowStr,
			"name", agent.FullName(),
			"email", agent.Email.String))
	}

	// Final summary
	status, _ := app.importer.GetStatus("agents")
	app.importer.AddLog("agents", app.i18n.Ts("importer.importComplete",
		"success", fmt.Sprintf("%d", status.Success),
		"total", fmt.Sprintf("%d", status.Total),
		"errors", fmt.Sprintf("%d", status.Errors)))

	return nil
}

func getField(record []string, headerMap map[string]int, name string) string {
	if idx, ok := headerMap[name]; ok && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}

func parseList(s string) []string {
	var result []string
	for part := range strings.SplitSeq(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func findInvalid(items []string, validMap map[string]bool) []string {
	var invalid []string
	for _, item := range items {
		if !validMap[item] {
			invalid = append(invalid, item)
		}
	}
	return invalid
}
