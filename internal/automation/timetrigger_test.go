package automation

import (
	"errors"
	"fmt"
	"testing"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/stretchr/testify/mock"
)

func makeRefs(fromID, n int) []cmodels.ConversationRef {
	refs := make([]cmodels.ConversationRef, 0, n)
	for i := 0; i < n; i++ {
		id := fromID + i
		refs = append(refs, cmodels.ConversationRef{ID: id, UUID: fmt.Sprintf("uuid-%d", id)})
	}
	return refs
}

func timeTriggerEngine(store *mockConversationStore) *Engine {
	e := createTestEngine(store)
	e.rules = []models.Rule{{Type: models.RuleTypeTimeTrigger}}
	return e
}

func TestTimeTrigger_NoRules_SkipsFetch(t *testing.T) {
	store := &mockConversationStore{}
	engine := createTestEngine(store)

	engine.handleTimeTrigger()

	store.AssertNotCalled(t, "GetConversationsCreatedAfter", mock.Anything, mock.Anything, mock.Anything)
}

func TestTimeTrigger_PagesUntilShortBatch(t *testing.T) {
	store := &mockConversationStore{}
	engine := timeTriggerEngine(store)

	batch1 := makeRefs(1, timeTriggerBatchSize)
	batch2 := makeRefs(timeTriggerBatchSize+1, 3)
	store.On("GetConversationsCreatedAfter", mock.Anything, 0, timeTriggerBatchSize).Return(batch1, nil).Once()
	store.On("GetConversationsCreatedAfter", mock.Anything, timeTriggerBatchSize, timeTriggerBatchSize).Return(batch2, nil).Once()
	store.On("GetConversation", 0, mock.Anything, "").Return(cmodels.Conversation{}, nil)

	engine.handleTimeTrigger()

	store.AssertExpectations(t)
	store.AssertNumberOfCalls(t, "GetConversation", timeTriggerBatchSize+3)
}

func TestTimeTrigger_EmptyBatch_StopsWithoutFetchingConversations(t *testing.T) {
	store := &mockConversationStore{}
	engine := timeTriggerEngine(store)

	store.On("GetConversationsCreatedAfter", mock.Anything, 0, timeTriggerBatchSize).Return([]cmodels.ConversationRef{}, nil).Once()

	engine.handleTimeTrigger()

	store.AssertExpectations(t)
	store.AssertNotCalled(t, "GetConversation", mock.Anything, mock.Anything, mock.Anything)
}

func TestTimeTrigger_FetchError_Stops(t *testing.T) {
	store := &mockConversationStore{}
	engine := timeTriggerEngine(store)

	store.On("GetConversationsCreatedAfter", mock.Anything, 0, timeTriggerBatchSize).Return([]cmodels.ConversationRef{}, errors.New("db down")).Once()

	engine.handleTimeTrigger()

	store.AssertExpectations(t)
	store.AssertNotCalled(t, "GetConversation", mock.Anything, mock.Anything, mock.Anything)
}

func TestTimeTrigger_ConversationFetchError_ContinuesBatch(t *testing.T) {
	store := &mockConversationStore{}
	engine := timeTriggerEngine(store)

	store.On("GetConversationsCreatedAfter", mock.Anything, 0, timeTriggerBatchSize).Return(makeRefs(1, 2), nil).Once()
	store.On("GetConversation", 0, "uuid-1", "").Return(cmodels.Conversation{}, errors.New("gone")).Once()
	store.On("GetConversation", 0, "uuid-2", "").Return(cmodels.Conversation{}, nil).Once()

	engine.handleTimeTrigger()

	store.AssertExpectations(t)
}
