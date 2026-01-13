package main

import (
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

// handleGetActivityLogs returns activity logs from the database.
func handleGetActivityLogs(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		order   = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total   = 0
	)
	page, pageSize := getPagination(r)
	logs, err := app.activityLog.GetAll(order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(logs) > 0 {
		total = logs[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    logs,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})

}
