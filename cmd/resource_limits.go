package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/resourceusage"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type resourceLimitsResponse struct {
	MaxIncomingMessageSize int64 `json:"max_incoming_message_size"`
	MaxStorageBytes        int64 `json:"max_storage_bytes"`
}

// Runtime snapshots never mutate Koanf. The writer mutex keeps persisted and
// applied limits in the same order when administrators save concurrently.
type resourceLimitState struct {
	updates sync.Mutex
	current atomic.Pointer[resourceusage.Limits]
}

var liveResourceLimits resourceLimitState

func (s *resourceLimitState) snapshot() resourceusage.Limits {
	if limits := s.current.Load(); limits != nil {
		return *limits
	}
	return resourceusage.Limits{MaxIncomingMessageSize: resourceusage.DefaultMaxIncomingMessageSize}
}

func (s *resourceLimitState) update(limits resourceusage.Limits, persist func(resourceusage.Limits) error, apply func(resourceusage.Limits)) error {
	limits, err := resourceusage.Normalize(limits)
	if err != nil {
		return err
	}
	s.updates.Lock()
	defer s.updates.Unlock()
	if err := persist(limits); err != nil {
		return err
	}
	s.current.Store(&limits)
	apply(limits)
	return nil
}

// Read config defaults only during startup, before request handling begins.
func resourceLimitsFromConfig() resourceusage.Limits {
	maxMessageSize := ko.Int64("message.max_incoming_message_size")
	if !ko.Exists("message.max_incoming_message_size") {
		maxMessageSize = resourceusage.DefaultMaxIncomingMessageSize
	}
	return resourceusage.Limits{
		MaxIncomingMessageSize: maxMessageSize,
		MaxStorageBytes:        ko.Int64("upload.max_storage_bytes"),
	}
}

func loadResourceLimits(settings interface {
	GetResourceLimits(resourceusage.Limits) (resourceusage.Limits, error)
}) {
	limits, err := settings.GetResourceLimits(resourceLimitsFromConfig())
	if err != nil {
		log.Fatalf("error loading resource limits: %v", err)
	}
	if _, err := resourceusage.Normalize(limits); err != nil {
		log.Fatalf("invalid resource limits: %v", err)
	}
	liveResourceLimits.current.Store(&limits)
}

func resourceLimitsResponseFor(limits resourceusage.Limits) resourceLimitsResponse {
	return resourceLimitsResponse{
		MaxIncomingMessageSize: limits.MaxIncomingMessageSize,
		MaxStorageBytes:        limits.MaxStorageBytes,
	}
}

func handleGetResourceLimits(r *fastglue.Request) error {
	return r.SendEnvelope(resourceLimitsResponseFor(liveResourceLimits.snapshot()))
}

func handleUpdateResourceLimits(r *fastglue.Request) error {
	app := r.Context.(*App)
	limits, err := decodeResourceLimits(r.RequestCtx.PostBody())
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, envelope.InputError)
	}
	if err := liveResourceLimits.update(limits, app.setting.SetResourceLimits, func(limits resourceusage.Limits) {
		app.media.SetMaxStorageBytes(limits.MaxStorageBytes)
		app.inbox.SetIncomingMessageSizeLimit(limits.MaxIncomingMessageSize)
	}); err != nil {
		app.lo.Error("error saving resource limits", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Unable to save resource limits", nil, envelope.GeneralError)
	}
	return r.SendEnvelope(resourceLimitsResponseFor(limits))
}

func decodeResourceLimits(body []byte) (resourceusage.Limits, error) {
	if len(body) > 4096 {
		return resourceusage.Limits{}, fmt.Errorf("resource limits exceed 4 KiB")
	}
	var limits resourceusage.Limits
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&limits); err != nil {
		return resourceusage.Limits{}, fmt.Errorf("invalid resource limits JSON")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return resourceusage.Limits{}, fmt.Errorf("invalid resource limits JSON")
	}
	return resourceusage.Normalize(limits)
}
