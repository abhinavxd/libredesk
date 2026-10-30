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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Get("%[1]s Required",
			app.i18n.Get("File")), nil, envelope.InputError)
	}

	fileContent, err := file.Open()
	if err != nil {
		app.lo.Error("error opening uploaded file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Get("Something went wrong"), nil, envelope.GeneralError)
	}
	defer fileContent.Close()

	reader := csv.NewReader(fileContent)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		app.lo.Debug("agent import rejected: invalid CSV", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Get("Invalid CSV file"), nil, envelope.InputError)
	}

	if len(records) < 2 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Get("CSV must contain headers and at least one data row"), nil, envelope.InputError)
	}

	err = app.importer.Submit("agents", func() error {
		return processAgentImport(app, records)
	})

	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, app.i18n.Get("Import already in progress"), nil, envelope.GeneralError)
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
	app.importer.AddLog("agents", app.i18n.Get("Starting import of %[1]s %[2]s", fmt.Sprintf("%d", total), "agents"))

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
			app.importer.AddLog("agents", app.i18n.Get("Row %[1]s: Error - missing required fields: %[2]s",
				rowStr,
				strings.Join(missing, ", ")))
			continue
		}

		// Validate email format
		if !stringutil.ValidEmail(email) {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Get("Row %[1]s: Error - invalid email format: %[2]s", rowStr, email))
			continue
		}

		// Parse and validate roles
		roles := parseList(rolesStr)
		if len(roles) == 0 {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Get("Row %[1]s (%[2]s): Error - at least one role required",
				rowStr,
				email))
			continue
		}

		invalidRoles := findInvalid(roles, validRoles)
		if len(invalidRoles) > 0 {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Get("Row %[1]s (%[2]s): Error - invalid role(s): %[3]s",
				rowStr,
				email,
				strings.Join(invalidRoles, ", ")))
			continue
		}

		// Parse and validate teams (optional)
		teams := parseList(teamsStr)
		if len(teams) > 0 {
			invalidTeams := findInvalid(teams, validTeams)
			if len(invalidTeams) > 0 {
				app.importer.UpdateCounts("agents", 0, 0, 1)
				app.importer.AddLog("agents", app.i18n.Get("Row %[1]s (%[2]s): Error - invalid team(s): %[3]s",
					rowStr,
					email,
					strings.Join(invalidTeams, ", ")))
				continue
			}
		}

		// Check if agent already exists
		if _, err := app.user.GetAgent(0, email); err == nil {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Get("Row %[1]s (%[2]s): Error - email already exists",
				rowStr,
				email))
			continue
		}

		// Create agent
		agent, err := app.user.CreateAgent(firstName, lastName, email, roles)
		if err != nil {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", app.i18n.Get("Row %[1]s (%[2]s): Error - failed to create agent: %[3]s",
				rowStr,
				email,
				err.Error()))
			continue
		}

		// Assign teams (if provided)
		if len(teams) > 0 {
			if err := app.team.UpsertUserTeams(agent.ID, teams); err != nil {
				app.importer.UpdateCounts("agents", 0, 0, 1)
				app.importer.AddLog("agents", app.i18n.Get("Row %[1]s (%[2]s): Error - team assignment failed: %[3]s",
					rowStr,
					email,
					err.Error()))
				continue
			}
		}

		app.importer.UpdateCounts("agents", 0, 1, 0)
		app.importer.AddLog("agents", app.i18n.Get("Row %[1]s: Created agent %[2]s (%[3]s)",
			rowStr,
			agent.FullName(),
			agent.Email.String))
	}

	// Final summary
	status, _ := app.importer.GetStatus("agents")
	app.importer.AddLog("agents", app.i18n.Get("Import completed: %[1]s of %[2]s successful, %[3]s failed",
		fmt.Sprintf("%d", status.Success),
		fmt.Sprintf("%d", status.Total),
		fmt.Sprintf("%d", status.Errors)))

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
