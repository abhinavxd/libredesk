package main

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const importNSTags = "tags"

func handleImportTags(r *fastglue.Request) error {
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
		app.lo.Debug("tag import rejected: invalid CSV", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidCsvFile"), nil, envelope.InputError)
	}

	if len(records) < 2 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("importer.csvMustContainHeadersAndData"), nil, envelope.InputError)
	}

	err = app.importer.Submit(importNSTags, func() error {
		return processTagImport(app, records)
	})

	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, app.i18n.T("importer.importAlreadyInProgress"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(true)
}

func handleGetTagImportStatus(r *fastglue.Request) error {
	var app = r.Context.(*App)
	status, err := app.importer.GetStatus(importNSTags)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(status)
}

func processTagImport(app *App, records [][]string) error {
	headerMap := make(map[string]int)
	for i, h := range records[0] {
		headerMap[strings.TrimSpace(strings.ToLower(h))] = i
	}

	if _, ok := headerMap["name"]; !ok {
		return fmt.Errorf("missing required column: name")
	}

	total := len(records) - 1
	app.importer.UpdateCounts(importNSTags, total, 0, 0)
	app.importer.AddLog(importNSTags, app.i18n.Ts("importer.startingImport",
		"count", strconv.Itoa(total),
		"type", importNSTags))

	for i, record := range records[1:] {
		rowStr := strconv.Itoa(i + 1)

		name := getField(record, headerMap, "name")
		if name == "" {
			app.importer.UpdateCounts(importNSTags, 0, 0, 1)
			app.importer.AddLog(importNSTags, app.i18n.Ts("importer.missingFields",
				"row", rowStr,
				"fields", "name"))
			continue
		}

		_, err := app.tag.Create(name)
		if err != nil {
			app.importer.UpdateCounts(importNSTags, 0, 0, 1)
			e, ok := err.(envelope.Error)
			if ok && e.ErrorType == envelope.ConflictError {
				app.importer.AddLog(importNSTags, app.i18n.Ts("importer.tagExists",
					"row", rowStr,
					"name", name))
			} else {
				app.importer.AddLog(importNSTags, app.i18n.Ts("importer.errorCreatingTag",
					"row", rowStr,
					"name", name,
					"error", err.Error()))
			}
			continue
		}

		app.importer.UpdateCounts(importNSTags, 0, 1, 0)
		app.importer.AddLog(importNSTags, app.i18n.Ts("importer.createdTag",
			"row", rowStr,
			"name", name))
	}

	status, _ := app.importer.GetStatus(importNSTags)
	app.importer.AddLog(importNSTags, app.i18n.Ts("importer.importComplete",
		"success", strconv.Itoa(status.Success),
		"total", strconv.Itoa(status.Total),
		"errors", strconv.Itoa(status.Errors)))

	return nil
}
