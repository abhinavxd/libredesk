package automation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// mockConversationStore tracks all ApplyAction calls for verification
type mockConversationStore struct {
	mock.Mock
	appliedActions []models.RuleAction
	callCount      int
}

func (m *mockConversationStore) ApplyAction(action models.RuleAction, conversation cmodels.Conversation, user umodels.User) error {
	m.callCount++
	m.appliedActions = append(m.appliedActions, action)
	args := m.Called(action, conversation, user)
	return args.Error(0)
}

func (m *mockConversationStore) GetConversation(teamID int, uuid string) (cmodels.Conversation, error) {
	args := m.Called(teamID, uuid)
	return args.Get(0).(cmodels.Conversation), args.Error(1)
}

func (m *mockConversationStore) GetConversationsCreatedAfter(t time.Time) ([]cmodels.Conversation, error) {
	args := m.Called(t)
	return args.Get(0).([]cmodels.Conversation), args.Error(1)
}

// Test Helpers
func createTestEngine(store *mockConversationStore) *Engine {
	logger := logf.New(logf.Opts{Level: logf.DebugLevel})
	return &Engine{
		conversationStore: store,
		lo:                &logger,
	}
}

func createTestConversation(opts ...func(*cmodels.Conversation)) cmodels.Conversation {
	conv := cmodels.Conversation{
		ID:        1,
		UUID:      "test-uuid-123",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		InboxID:   1,
		StatusID:  null.IntFrom(1),
		Contact: cmodels.ConversationContact{
			ID:        1,
			Email:     null.StringFrom("test@example.com"),
			FirstName: "Test",
			LastName:  "User",
		},
	}
	
	for _, opt := range opts {
		opt(&conv)
	}
	
	return conv
}

func createTestRule(groups []models.RuleGroup, actions []models.RuleAction, groupOp string) models.Rule {
	return models.Rule{
		Groups:        groups,
		Actions:       actions,
		GroupOperator: groupOp,
		ExecutionMode: models.ExecutionModeAll,
	}
}

// Test: Basic AND operator within a group - all conditions must pass
func TestEvaluateGroup_AND_AllTrue(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1)
		c.PriorityID = null.IntFrom(2)
	})

	rules := []models.Rule{
		createTestRule(
			[]models.RuleGroup{
				{
					LogicalOp: models.OperatorAnd,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
						{Field: models.ConversationPriority, Operator: models.RuleOperatorEquals, Value: "2", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			[]models.RuleAction{
				{Type: models.ActionSetStatus, Value: []string{"2"}},
			},
			models.OperatorOR,
		),
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "ApplyAction should be called once")
	assert.Equal(t, models.ActionSetStatus, mockStore.appliedActions[0].Type)
}

// Test: AND operator short-circuit - stops on first false
func TestEvaluateGroup_AND_ShortCircuit(t *testing.T) {
	mockStore := new(mockConversationStore)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1)
		c.PriorityID = null.IntFrom(3) // Different from expected
	})

	rules := []models.Rule{
		createTestRule(
			[]models.RuleGroup{
				{
					LogicalOp: models.OperatorAnd,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
						{Field: models.ConversationPriority, Operator: models.RuleOperatorEquals, Value: "2", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			[]models.RuleAction{
				{Type: models.ActionSetStatus, Value: []string{"2"}},
			},
			models.OperatorOR,
		),
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 0, mockStore.callCount, "ApplyAction should not be called when AND conditions fail")
}

// Test: OR operator - at least one condition must pass
func TestEvaluateGroup_OR_OnlyOneTrue(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1)
		c.PriorityID = null.IntFrom(3) // Different, but OR should still pass
	})

	rules := []models.Rule{
		createTestRule(
			[]models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
						{Field: models.ConversationPriority, Operator: models.RuleOperatorEquals, Value: "2", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			[]models.RuleAction{
				{Type: models.ActionAssignUser, Value: []string{"100"}},
			},
			models.OperatorOR,
		),
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "ApplyAction should be called once for OR condition")
}

// Test: Two groups with AND group operator
func TestTwoGroups_AND_BothMustPass(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1)
		c.PriorityID = null.IntFrom(2)
		c.InboxID = 11
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationInbox, Operator: models.RuleOperatorEquals, Value: "11", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSendPrivateNote, Value: []string{"Both groups matched!"}},
			},
			GroupOperator: models.OperatorAnd,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "ApplyAction should be called when both groups pass")
}

// Test: Empty group handling
func TestEmptyGroup_Skipped(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1)
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
				{
					LogicalOp: models.OperatorOR,
					Rules:     []models.RuleDetail{}, // Empty group
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetPriority, Value: []string{"3"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "ApplyAction should be called, empty group is skipped")
}

// Test: First match execution mode
func TestExecutionMode_FirstMatch(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1)
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetStatus, Value: []string{"2"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeFirstMatch,
		},
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetPriority, Value: []string{"3"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeFirstMatch,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "Only first matching rule should execute in first_match mode")
	assert.Equal(t, models.ActionSetStatus, mockStore.appliedActions[0].Type)
}

// Test: All execution mode
func TestExecutionMode_All(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1)
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetStatus, Value: []string{"2"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetPriority, Value: []string{"3"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 2, mockStore.callCount, "All matching rules should execute in 'all' mode")
}

// Test: Null field handling with set/not set operators
func TestNullFields_SetNotSet(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.AssignedUserID = null.Int{} // Not set (null)
		c.StatusID = null.IntFrom(1)  // Set
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorAnd,
					Rules: []models.RuleDetail{
						{Field: models.ConversationAssignedUser, Operator: models.RuleOperatorNotSet, Value: "", FieldType: models.FieldTypeConversationField},
						{Field: models.ConversationStatus, Operator: models.RuleOperatorSet, Value: "", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionAssignUser, Value: []string{"100"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "Should handle null fields with set/not set operators")
}

// Test: Custom attributes - basic string comparison
func TestCustomAttributes_StringComparison(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	customAttrs := map[string]interface{}{
		"client_id": "YYYYYY",
		"region":    "US",
	}
	customJSON, _ := json.Marshal(customAttrs)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.Contact.CustomAttributes = customJSON
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorAnd,
					Rules: []models.RuleDetail{
						{Field: "client_id", Operator: models.RuleOperatorEquals, Value: "YYYYYY", FieldType: models.FieldTypeContactCustomAttribute},
						{Field: "region", Operator: models.RuleOperatorEquals, Value: "US", FieldType: models.FieldTypeContactCustomAttribute},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSendCSAT, Value: []string{"0"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "Custom attributes should be compared correctly")
	assert.Equal(t, models.ActionSendCSAT, mockStore.appliedActions[0].Type)
}

// Test: Custom attributes - missing field
func TestCustomAttributes_MissingField(t *testing.T) {
	mockStore := new(mockConversationStore)
	engine := createTestEngine(mockStore)

	customAttrs := map[string]interface{}{
		"client_id": "YYYYYY",
	}
	customJSON, _ := json.Marshal(customAttrs)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.Contact.CustomAttributes = customJSON
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: "non_existent_field", Operator: models.RuleOperatorEquals, Value: "test", FieldType: models.FieldTypeContactCustomAttribute},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetStatus, Value: []string{"2"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 0, mockStore.callCount, "Missing custom attribute should fail the rule")
}

// Test: Contains operator with multiple values
func TestContainsOperator_MultipleValues(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.Subject = null.StringFrom("Urgent: Need help with billing issue")
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationSubject, Operator: models.RuleOperatorContains, Value: "urgent, critical, emergency", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetPriority, Value: []string{"4"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "Contains operator should match with comma-separated values")
}

// Test: Not contains operator
func TestNotContainsOperator(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.Subject = null.StringFrom("Regular support request")
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationSubject, Operator: models.RuleOperatorNotContains, Value: "spam, test, ignore", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionAssignTeam, Value: []string{"1"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "Not contains operator should pass when values are not present")
}

// Test: Greater than and less than operators
func TestNumericComparisons(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.CreatedAt = time.Now().Add(-25 * time.Hour) // 25 hours ago
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationHoursSinceCreated, Operator: models.RuleOperatorGreaterThan, Value: "24", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetPriority, Value: []string{"4"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "Greater than operator should work with numeric comparisons")
}

// Test: Real-world CSAT automation
func TestRealWorld_CSATAutomation(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	customAttrs := map[string]interface{}{
		"client_id": "YYYYYY",
	}
	customJSON, _ := json.Marshal(customAttrs)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(5) // Resolved
		c.Contact.CustomAttributes = customJSON
	})

	// Simulating the SEND CSAT ON RESOLVE rule from the database
	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorAnd,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "5", FieldType: models.FieldTypeConversationField},
						{Field: "client_id", Operator: models.RuleOperatorEquals, Value: "YYYYYY", FieldType: models.FieldTypeContactCustomAttribute},
					},
				},
				{
					LogicalOp: models.OperatorAnd,
					Rules: []models.RuleDetail{
						{Field: "client_id", Operator: models.RuleOperatorEquals, Value: "XXXXXX", FieldType: models.FieldTypeContactCustomAttribute},
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "5", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSendCSAT, Value: []string{"0"}},
			},
			GroupOperator: models.OperatorOR, // Either group can match
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "CSAT should be sent when status is resolved and client_id matches")
	assert.Equal(t, models.ActionSendCSAT, mockStore.appliedActions[0].Type)
}

// Test: Real-world NEW TICKET automation
func TestRealWorld_NewTicketAutomation(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(2)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1) // New
	})

	// Simulating the NEW TICKET rule from the database
	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSendPrivateNote, Value: []string{"<p>New ticket!</p>"}},
				{Type: models.ActionSetSLA, Value: []string{"11"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 2, mockStore.callCount, "Both actions should be executed for new ticket")
	assert.Equal(t, models.ActionSendPrivateNote, mockStore.appliedActions[0].Type)
	assert.Equal(t, models.ActionSetSLA, mockStore.appliedActions[1].Type)
}

// Test: Case sensitivity for Contains/NotContains operators
func TestCaseSensitivity_ContainsOperators(t *testing.T) {
	testCases := []struct {
		name          string
		subject       string
		searchValue   string
		caseSensitive bool
		operator      string
		shouldMatch   bool
	}{
		{
			name:          "Contains case-sensitive: TEST != test",
			subject:       "This is a TEST message",
			searchValue:   "test",
			caseSensitive: true,
			operator:      models.RuleOperatorContains,
			shouldMatch:   false,
		},
		{
			name:          "Contains case-insensitive: TEST == test",
			subject:       "This is a TEST message",
			searchValue:   "test",
			caseSensitive: false,
			operator:      models.RuleOperatorContains,
			shouldMatch:   true,
		},
		{
			name:          "NotContains case-sensitive: TEST != test",
			subject:       "This is a TEST message",
			searchValue:   "test",
			caseSensitive: true,
			operator:      models.RuleOperatorNotContains,
			shouldMatch:   true,
		},
		{
			name:          "NotContains case-insensitive: TEST == test",
			subject:       "This is a TEST message",
			searchValue:   "test",
			caseSensitive: false,
			operator:      models.RuleOperatorNotContains,
			shouldMatch:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := new(mockConversationStore)
			if tc.shouldMatch {
				mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			}
			engine := createTestEngine(mockStore)

			conversation := createTestConversation(func(c *cmodels.Conversation) {
				c.Subject = null.StringFrom(tc.subject)
			})

			rules := []models.Rule{
				{
					Groups: []models.RuleGroup{
						{
							LogicalOp: models.OperatorOR,
							Rules: []models.RuleDetail{
								{
									Field:              models.ConversationSubject,
									Operator:           tc.operator,
									Value:              tc.searchValue,
									FieldType:          models.FieldTypeConversationField,
									CaseSensitiveMatch: tc.caseSensitive,
								},
							},
						},
					},
					Actions: []models.RuleAction{
						{Type: models.ActionSetStatus, Value: []string{"2"}},
					},
					GroupOperator: models.OperatorOR,
					ExecutionMode: models.ExecutionModeAll,
				},
			}

			engine.evalConversationRules(rules, conversation)
			
			if tc.shouldMatch {
				assert.Equal(t, 1, mockStore.callCount, "Expected action to be triggered")
			} else {
				assert.Equal(t, 0, mockStore.callCount, "Expected action NOT to be triggered")
			}
		})
	}
}

// Test: Case sensitivity for equals operator
func TestCaseSensitivity_Equals(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.Contact.Email = null.StringFrom("Test@Example.COM")
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{
							Field:              models.ContactEmail,
							Operator:           models.RuleOperatorEquals,
							Value:              "test@example.com",
							FieldType:          models.FieldTypeConversationField,
							CaseSensitiveMatch: false,
						},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionAssignUser, Value: []string{"100"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "Case insensitive comparison should match")
}

// Test: Invalid operator handling
func TestInvalidOperator(t *testing.T) {
	mockStore := new(mockConversationStore)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation()

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: "invalid_operator", Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetStatus, Value: []string{"2"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 0, mockStore.callCount, "Invalid operator should not trigger action")
}

// Test: Contradictory conditions
func TestContradictoryConditions(t *testing.T) {
	mockStore := new(mockConversationStore)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1)
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorAnd,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "2", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetStatus, Value: []string{"3"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 0, mockStore.callCount, "Contradictory conditions should never match")
}

// Test: Tautology - always true condition
func TestTautologyCondition(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1)
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
						{Field: models.ConversationStatus, Operator: models.RuleOperatorNotEqual, Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetPriority, Value: []string{"2"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "Tautology condition should always match")
}

// Test: Custom attribute type conversion
func TestCustomAttributeTypes(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	customAttrs := map[string]interface{}{
		"age":        25,          // int
		"score":      98.5,        // float64
		"is_premium": true,        // bool
		"name":       "TestUser",  // string
	}
	customJSON, _ := json.Marshal(customAttrs)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.Contact.CustomAttributes = customJSON
	})

	t.Run("Integer comparison", func(t *testing.T) {
		rules := []models.Rule{
			{
				Groups: []models.RuleGroup{
					{
						LogicalOp: models.OperatorOR,
						Rules: []models.RuleDetail{
							{Field: "age", Operator: models.RuleOperatorEquals, Value: "25", FieldType: models.FieldTypeContactCustomAttribute},
						},
					},
				},
				Actions: []models.RuleAction{
					{Type: models.ActionSetStatus, Value: []string{"2"}},
				},
				GroupOperator: models.OperatorOR,
				ExecutionMode: models.ExecutionModeAll,
			},
		}
		mockStore.appliedActions = nil
		mockStore.callCount = 0
		engine.evalConversationRules(rules, conversation)
		assert.Equal(t, 1, mockStore.callCount, "Integer custom attribute should be compared correctly")
	})

	t.Run("Float comparison", func(t *testing.T) {
		rules := []models.Rule{
			{
				Groups: []models.RuleGroup{
					{
						LogicalOp: models.OperatorOR,
						Rules: []models.RuleDetail{
							{Field: "score", Operator: models.RuleOperatorEquals, Value: "98", FieldType: models.FieldTypeContactCustomAttribute},
						},
					},
				},
				Actions: []models.RuleAction{
					{Type: models.ActionSetStatus, Value: []string{"2"}},
				},
				GroupOperator: models.OperatorOR,
				ExecutionMode: models.ExecutionModeAll,
			},
		}
		mockStore.appliedActions = nil
		mockStore.callCount = 0
		engine.evalConversationRules(rules, conversation)
		assert.Equal(t, 1, mockStore.callCount, "Float custom attribute should be converted to int for comparison")
	})

	t.Run("Boolean comparison", func(t *testing.T) {
		rules := []models.Rule{
			{
				Groups: []models.RuleGroup{
					{
						LogicalOp: models.OperatorOR,
						Rules: []models.RuleDetail{
							{Field: "is_premium", Operator: models.RuleOperatorEquals, Value: "true", FieldType: models.FieldTypeContactCustomAttribute},
						},
					},
				},
				Actions: []models.RuleAction{
					{Type: models.ActionSetStatus, Value: []string{"2"}},
				},
				GroupOperator: models.OperatorOR,
				ExecutionMode: models.ExecutionModeAll,
			},
		}
		mockStore.appliedActions = nil
		mockStore.callCount = 0
		engine.evalConversationRules(rules, conversation)
		assert.Equal(t, 1, mockStore.callCount, "Boolean custom attribute should be compared correctly")
	})
}

// Test: Hours since fields with null values
func TestHoursSinceFields_NullHandling(t *testing.T) {
	mockStore := new(mockConversationStore)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.ResolvedAt = null.Time{} // Not resolved yet
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationHoursSinceResolved, Operator: models.RuleOperatorGreaterThan, Value: "24", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSendCSAT, Value: []string{"0"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 0, mockStore.callCount, "Should not trigger action when time field is null")
}

// Test: Multiple actions per rule
func TestMultipleActions(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(4)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(1)
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "1", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetStatus, Value: []string{"2"}},
				{Type: models.ActionSetPriority, Value: []string{"3"}},
				{Type: models.ActionAssignUser, Value: []string{"100"}},
				{Type: models.ActionSendPrivateNote, Value: []string{"Multiple actions triggered"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 4, mockStore.callCount, "All actions should be executed")
	assert.Equal(t, models.ActionSetStatus, mockStore.appliedActions[0].Type)
	assert.Equal(t, models.ActionSetPriority, mockStore.appliedActions[1].Type)
	assert.Equal(t, models.ActionAssignUser, mockStore.appliedActions[2].Type)
	assert.Equal(t, models.ActionSendPrivateNote, mockStore.appliedActions[3].Type)
}

// Test: evaluateFinalResult function
func TestEvaluateFinalResult(t *testing.T) {
	t.Run("AND operator - all true", func(t *testing.T) {
		result := evaluateFinalResult([]bool{true, true, true}, models.OperatorAnd)
		assert.True(t, result, "AND with all true should return true")
	})

	t.Run("AND operator - one false", func(t *testing.T) {
		result := evaluateFinalResult([]bool{true, false, true}, models.OperatorAnd)
		assert.False(t, result, "AND with one false should return false")
	})

	t.Run("OR operator - all false", func(t *testing.T) {
		result := evaluateFinalResult([]bool{false, false, false}, models.OperatorOR)
		assert.False(t, result, "OR with all false should return false")
	})

	t.Run("OR operator - one true", func(t *testing.T) {
		result := evaluateFinalResult([]bool{false, true, false}, models.OperatorOR)
		assert.True(t, result, "OR with one true should return true")
	})

	t.Run("Invalid operator", func(t *testing.T) {
		result := evaluateFinalResult([]bool{true, true}, "INVALID")
		assert.False(t, result, "Invalid operator should return false")
	})
}

// Test: Contains operator with text normalization
func TestContainsOperator_TextNormalization(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.LastMessage = null.StringFrom("This   has    multiple     spaces")
	})

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationContent, Operator: models.RuleOperatorContains, Value: "has multiple spaces", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSetStatus, Value: []string{"2"}},
			},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 1, mockStore.callCount, "Contains should normalize whitespace and match")
}

// Test: Mock verification precision (The Mock Verifier's Gauntlet)
func TestMockVerificationPrecision(t *testing.T) {
	mockStore := new(mockConversationStore)
	
	expectedAction := models.RuleAction{
		Type:  models.ActionReply,
		Value: []string{"<p>Test reply automation!</p>"},
	}
	
	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.Contact.Email = null.StringFrom("libredesk.io@gmail.com")
	})
	
	// Set up precise expectation
	mockStore.On("ApplyAction", expectedAction, conversation, umodels.User{}).Return(nil).Once()
	
	engine := createTestEngine(mockStore)

	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ContactEmail, Operator: models.RuleOperatorEquals, Value: "libredesk.io@gmail.com", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{expectedAction},
			GroupOperator: models.OperatorOR,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	// This will verify the exact parameters were passed
	mockStore.AssertExpectations(t)
	assert.Equal(t, 1, mockStore.callCount, "Action should be called exactly once")
	assert.Equal(t, expectedAction, mockStore.appliedActions[0], "Action should match exactly")
}

// Test: Complex real-world scenario with multiple groups and operators
func TestComplexRealWorldScenario(t *testing.T) {
	mockStore := new(mockConversationStore)
	mockStore.On("ApplyAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	engine := createTestEngine(mockStore)

	customAttrs := map[string]interface{}{
		"client_id": "XXXXXX",
		"tier":      "premium",
	}
	customJSON, _ := json.Marshal(customAttrs)

	conversation := createTestConversation(func(c *cmodels.Conversation) {
		c.StatusID = null.IntFrom(5) // Resolved
		c.CreatedAt = time.Now().Add(-48 * time.Hour)
		c.Contact.CustomAttributes = customJSON
		c.InboxID = 11
	})

	// Complex rule: (status=resolved AND client is premium) AND (created > 24h ago OR inbox=11)
	rules := []models.Rule{
		{
			Groups: []models.RuleGroup{
				{
					LogicalOp: models.OperatorAnd,
					Rules: []models.RuleDetail{
						{Field: models.ConversationStatus, Operator: models.RuleOperatorEquals, Value: "5", FieldType: models.FieldTypeConversationField},
						{Field: "tier", Operator: models.RuleOperatorEquals, Value: "premium", FieldType: models.FieldTypeContactCustomAttribute},
					},
				},
				{
					LogicalOp: models.OperatorOR,
					Rules: []models.RuleDetail{
						{Field: models.ConversationHoursSinceCreated, Operator: models.RuleOperatorGreaterThan, Value: "24", FieldType: models.FieldTypeConversationField},
						{Field: models.ConversationInbox, Operator: models.RuleOperatorEquals, Value: "11", FieldType: models.FieldTypeConversationField},
					},
				},
			},
			Actions: []models.RuleAction{
				{Type: models.ActionSendCSAT, Value: []string{"0"}},
				{Type: models.ActionSetTags, Value: []string{"premium-resolved"}},
			},
			GroupOperator: models.OperatorAnd,
			ExecutionMode: models.ExecutionModeAll,
		},
	}

	engine.evalConversationRules(rules, conversation)
	
	assert.Equal(t, 2, mockStore.callCount, "Complex conditions met, both actions should trigger")
	assert.Equal(t, models.ActionSendCSAT, mockStore.appliedActions[0].Type)
	assert.Equal(t, models.ActionSetTags, mockStore.appliedActions[1].Type)
}