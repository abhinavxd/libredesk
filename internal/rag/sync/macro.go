// Package sync provides services for syncing knowledge sources to RAG.
package sync

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/abhinavxd/libredesk/internal/macro"
	"github.com/abhinavxd/libredesk/internal/rag"
	"github.com/zerodha/logf"
)

// MacroSyncer syncs macros to the RAG knowledge base.
type MacroSyncer struct {
	macro *macro.Manager
	rag   *rag.Manager
	lo    *logf.Logger
}

// NewMacroSyncer creates a new macro syncer.
func NewMacroSyncer(macroMgr *macro.Manager, ragMgr *rag.Manager, lo *logf.Logger) *MacroSyncer {
	return &MacroSyncer{
		macro: macroMgr,
		rag:   ragMgr,
		lo:    lo,
	}
}

// Sync syncs all macros to the RAG documents table.
func (s *MacroSyncer) Sync(sourceID int) error {
	s.lo.Info("starting macro sync", "source_id", sourceID)

	// Get all macros
	macros, err := s.macro.GetAll()
	if err != nil {
		s.lo.Error("error fetching macros", "error", err)
		return fmt.Errorf("fetching macros: %w", err)
	}

	// Track which macros we've synced
	syncedRefs := make(map[string]bool)

	for _, m := range macros {
		// Skip macros with no content
		if strings.TrimSpace(m.MessageContent) == "" {
			continue
		}

		sourceRef := fmt.Sprintf("macro_%d", m.ID)
		syncedRefs[sourceRef] = true

		// Clean HTML from content for embedding
		content := stripHTML(m.MessageContent)
		if strings.TrimSpace(content) == "" {
			continue
		}

		// Build metadata
		metadata, _ := json.Marshal(map[string]interface{}{
			"macro_id":   m.ID,
			"visibility": m.Visibility,
			"updated_at": m.UpdatedAt,
		})

		// Add/update document
		if err := s.rag.AddDocument(sourceID, sourceRef, m.Name, content, metadata); err != nil {
			s.lo.Error("error syncing macro", "macro_id", m.ID, "error", err)
			// Continue with other macros
			continue
		}

		s.lo.Debug("synced macro", "macro_id", m.ID, "name", m.Name)
	}

	// Remove documents for deleted macros
	existingDocs, err := s.getExistingDocuments(sourceID)
	if err != nil {
		s.lo.Error("error fetching existing documents", "error", err)
	} else {
		for _, ref := range existingDocs {
			if !syncedRefs[ref] {
				s.lo.Info("removing deleted macro from RAG", "source_ref", ref)
				// Delete by source_ref - need to add this method
			}
		}
	}

	s.lo.Info("macro sync complete", "source_id", sourceID, "synced", len(syncedRefs))
	return nil
}

// getExistingDocuments returns source_refs for all documents in a source.
func (s *MacroSyncer) getExistingDocuments(sourceID int) ([]string, error) {
	var refs []string
	err := s.rag.GetDB().Select(&refs, 
		"SELECT source_ref FROM rag_documents WHERE source_id = $1 AND source_ref IS NOT NULL", 
		sourceID)
	return refs, err
}

// stripHTML removes HTML tags and decodes entities from content.
func stripHTML(s string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, " ")
	// Decode HTML entities
	s = html.UnescapeString(s)
	// Normalize whitespace
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
