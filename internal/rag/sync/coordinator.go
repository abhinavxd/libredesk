package sync

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/macro"
	"github.com/abhinavxd/libredesk/internal/rag"
	"github.com/abhinavxd/libredesk/internal/rag/models"
	"github.com/zerodha/logf"
)

// Coordinator manages syncing all knowledge sources.
type Coordinator struct {
	rag      *rag.Manager
	macro    *macro.Manager
	lo       *logf.Logger
	interval time.Duration

	macroSyncer   *MacroSyncer
	webpageSyncer *WebpageSyncer
	fileSyncer    *FileSyncer

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// CoordinatorOpts contains options for creating a Coordinator.
type CoordinatorOpts struct {
	RAG          *rag.Manager
	Macro        *macro.Manager
	Lo           *logf.Logger
	SyncInterval time.Duration
}

// NewCoordinator creates a new sync coordinator.
func NewCoordinator(opts CoordinatorOpts) *Coordinator {
	if opts.SyncInterval == 0 {
		opts.SyncInterval = 1 * time.Hour
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Coordinator{
		rag:           opts.RAG,
		macro:         opts.Macro,
		lo:            opts.Lo,
		interval:      opts.SyncInterval,
		macroSyncer:   NewMacroSyncer(opts.Macro, opts.RAG, opts.Lo),
		webpageSyncer: NewWebpageSyncer(opts.RAG, opts.Lo),
		fileSyncer:    NewFileSyncer(opts.RAG, opts.Lo),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins background syncing of all sources.
func (c *Coordinator) Start() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.lo.Info("starting RAG sync coordinator", "interval", c.interval)

		// Initial sync
		c.SyncAll()

		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-c.ctx.Done():
				c.lo.Info("stopping RAG sync coordinator")
				return
			case <-ticker.C:
				c.SyncAll()
			}
		}
	}()
}

// Stop stops the background sync.
func (c *Coordinator) Stop() {
	c.cancel()
	c.wg.Wait()
}

// SyncAll syncs all enabled sources.
func (c *Coordinator) SyncAll() {
	sources, err := c.rag.GetSources()
	if err != nil {
		c.lo.Error("error fetching sources for sync", "error", err)
		return
	}

	for _, source := range sources {
		if !source.Enabled {
			continue
		}

		// Skip file sources in periodic sync - they sync on upload
		if source.SourceType == "file" {
			continue
		}

		if err := c.SyncSource(source); err != nil {
			c.lo.Error("error syncing source", "source_id", source.ID, "name", source.Name, "error", err)
		}
	}
}

// SyncSource syncs a single source.
func (c *Coordinator) SyncSource(source models.Source) error {
	c.lo.Info("syncing source", "source_id", source.ID, "name", source.Name, "type", source.SourceType)

	var err error
	switch source.SourceType {
	case "macro":
		err = c.macroSyncer.Sync(source.ID)
	case "webpage":
		var config models.WebpageConfig
		if err := json.Unmarshal(source.Config, &config); err != nil {
			return err
		}
		err = c.webpageSyncer.Sync(source.ID, config)
	case "file":
		var config models.FileConfig
		if err := json.Unmarshal(source.Config, &config); err != nil {
			return err
		}
		err = c.fileSyncer.Sync(source.ID, config)
	default:
		c.lo.Warn("unknown source type", "type", source.SourceType)
		return nil
	}

	if err != nil {
		return err
	}

	// Update last synced timestamp
	c.rag.UpdateSourceSynced(source.ID)
	return nil
}

// SyncSourceByID syncs a source by ID.
func (c *Coordinator) SyncSourceByID(sourceID int) error {
	source, err := c.rag.GetSource(sourceID)
	if err != nil {
		return err
	}
	return c.SyncSource(source)
}
