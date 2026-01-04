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

	// Get file from form
	file, err := r.RequestCtx.FormFile("file")
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No file provided", nil, envelope.InputError)
	}

	// Open file
	fileContent, err := file.Open()
	if err != nil {
		app.lo.Error("error opening uploaded file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, envelope.GeneralError)
	}
	defer fileContent.Close()

	// Parse CSV
	reader := csv.NewReader(fileContent)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		app.lo.Error("error parsing CSV", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid CSV format", nil, envelope.InputError)
	}

	if len(records) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty CSV file", nil, envelope.InputError)
	}

	// Submit import job
	err = app.importer.Submit("agents", func() error {
		return processAgentImport(app, records)
	})

	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, err.Error(), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]string{
		"message": "Import started",
	})
}

// handleGetAgentImportStatus returns current import status
func handleGetAgentImportStatus(r *fastglue.Request) error {
	var app = r.Context.(*App)

	status, err := app.importer.GetStatus("agents")
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, err.Error(), nil, envelope.NotFoundError)
	}

	return r.SendEnvelope(status)
}

// processAgentImport processes CSV records and creates agents
func processAgentImport(app *App, records [][]string) error {
	if len(records) < 2 {
		return fmt.Errorf("CSV must have headers and at least one data row")
	}

	// Parse headers
	headers := records[0]
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(strings.ToLower(h))] = i
	}

	// Validate required columns
	required := []string{"first_name", "last_name", "email", "roles", "teams"}
	for _, r := range required {
		if _, ok := headerMap[r]; !ok {
			return fmt.Errorf("missing required column: %s", r)
		}
	}

	// Set total count
	total := len(records) - 1
	app.importer.UpdateCounts("agents", total, 0, 0)
	app.importer.AddLog("agents", fmt.Sprintf("Starting import of %d agents", total))

	// Process each row
	for i, record := range records[1:] {
		rowNum := i + 2 // +2 for header and 1-based indexing

		// Skip empty rows
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		// Parse fields
		firstName := getField(record, headerMap, "first_name")
		lastName := getField(record, headerMap, "last_name")
		email := strings.TrimSpace(strings.ToLower(getField(record, headerMap, "email")))
		rolesStr := getField(record, headerMap, "roles")
		teamsStr := getField(record, headerMap, "teams")

		// Validate required fields
		if firstName == "" || lastName == "" || email == "" || rolesStr == "" || teamsStr == "" {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", fmt.Sprintf("Row %d: Missing required fields", rowNum))
			continue
		}

		// Validate email
		if !stringutil.ValidEmail(email) {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", fmt.Sprintf("Row %d: Invalid email format - %s", rowNum, email))
			continue
		}

		// Parse roles (comma or semicolon separated)
		rolesStr = strings.ReplaceAll(rolesStr, ";", ",")
		rolesParts := strings.Split(rolesStr, ",")
		var roles []string
		for _, role := range rolesParts {
			if r := strings.TrimSpace(role); r != "" {
				roles = append(roles, r)
			}
		}

		if len(roles) == 0 {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", fmt.Sprintf("Row %d: At least one role is required", rowNum))
			continue
		}

		// Parse teams (comma or semicolon separated)
		teamsStr = strings.ReplaceAll(teamsStr, ";", ",")
		teamsParts := strings.Split(teamsStr, ",")
		var teams []string
		for _, team := range teamsParts {
			if t := strings.TrimSpace(team); t != "" {
				teams = append(teams, t)
			}
		}

		if len(teams) == 0 {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", fmt.Sprintf("Row %d: At least one team is required", rowNum))
			continue
		}

		// Create agent
		agent, err := app.user.CreateAgent(firstName, lastName, email, roles)
		if err != nil {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", fmt.Sprintf("Row %d: Failed to create - %v", rowNum, err))
			continue
		}

		// Assign teams
		if err := app.team.UpsertUserTeams(agent.ID, teams); err != nil {
			app.importer.UpdateCounts("agents", 0, 0, 1)
			app.importer.AddLog("agents", fmt.Sprintf("Row %d: Created agent but failed to assign teams - %v", rowNum, err))
			continue
		}

		app.importer.UpdateCounts("agents", 0, 1, 0)
		app.importer.AddLog("agents", fmt.Sprintf("Row %d: Created agent %s (%s) with teams", rowNum, agent.FullName(), agent.Email.String))

		// Log progress every 10 records
		if (i+1)%10 == 0 {
			app.importer.AddLog("agents", fmt.Sprintf("Progress: %d/%d processed", i+1, total))
		}
	}

	// Final summary
	status, _ := app.importer.GetStatus("agents")
	app.importer.AddLog("agents", fmt.Sprintf("Import completed: %d successful, %d failed out of %d total",
		status.Success, status.Errors, status.Total))

	return nil
}

// getField safely retrieves a field from CSV record
func getField(record []string, headerMap map[string]int, name string) string {
	if idx, ok := headerMap[name]; ok && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}
