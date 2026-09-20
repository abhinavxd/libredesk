package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetResourcePolicy(r *fastglue.Request) error {
	app := r.Context.(*App)
	cfg, err := app.setting.GetResourcePolicy()
	if err != nil {
		app.lo.Error("error reading resource policy", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Unable to read resource policy", nil, envelope.GeneralError)
	}
	return r.SendEnvelope(cfg)
}

func handleUpdateResourcePolicy(r *fastglue.Request) error {
	app := r.Context.(*App)
	cfg, err := decodeResourcePolicy(r.RequestCtx.PostBody())
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, envelope.InputError)
	}
	if err := app.setting.SetResourcePolicy(cfg); err != nil {
		app.lo.Error("error saving resource policy", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Unable to save resource policy", nil, envelope.GeneralError)
	}
	return r.SendEnvelope(cfg)
}

func decodeResourcePolicy(body []byte) (resourcepolicy.Config, error) {
	if len(body) > 32768 {
		return resourcepolicy.Blocked(), fmt.Errorf("resource policy exceeds 32 KiB")
	}
	var cfg resourcepolicy.Config
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return resourcepolicy.Blocked(), fmt.Errorf("invalid resource policy JSON")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return resourcepolicy.Blocked(), fmt.Errorf("invalid resource policy JSON")
	}
	return resourcepolicy.Normalize(cfg)
}
