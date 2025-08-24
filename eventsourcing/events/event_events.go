package events

import (
	"time"
	"expense-tracker/eventsourcing"
)

// Event domain events
type EventCreated struct {
	eventsourcing.BaseEvent
	Name              string `json:"name"`
	Description       string `json:"description"`
	Code              string `json:"code"`
	CreatedBy         string `json:"created_by"`
	Currency          string `json:"currency"`
	RequireApproval   bool   `json:"require_approval"`
}

func NewEventCreated(eventID, name, description, code, createdBy, currency string, requireApproval bool) *EventCreated {
	return &EventCreated{
		BaseEvent: eventsourcing.BaseEvent{
			AggregateID: eventID,
			EventType:   "EventCreated",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"name":             name,
				"description":      description,
				"code":             code,
				"created_by":       createdBy,
				"currency":         currency,
				"require_approval": requireApproval,
			},
		},
		Name:            name,
		Description:     description,
		Code:            code,
		CreatedBy:       createdBy,
		Currency:        currency,
		RequireApproval: requireApproval,
	}
}

type UserJoinedEvent struct {
	eventsourcing.BaseEvent
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func NewUserJoinedEvent(eventID, userID, role string) *UserJoinedEvent {
	return &UserJoinedEvent{
		BaseEvent: eventsourcing.BaseEvent{
			AggregateID: eventID,
			EventType:   "UserJoinedEvent",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"user_id": userID,
				"role":    role,
			},
		},
		UserID: userID,
		Role:   role,
	}
}

type ExpenseAdded struct {
	eventsourcing.BaseEvent
	ExpenseID   string  `json:"expense_id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	SubmittedBy string  `json:"submitted_by"`
}

func NewExpenseAdded(eventID, expenseID, description, category, submittedBy string, amount float64) *ExpenseAdded {
	return &ExpenseAdded{
		BaseEvent: eventsourcing.BaseEvent{
			AggregateID: eventID,
			EventType:   "ExpenseAdded",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"expense_id":   expenseID,
				"amount":       amount,
				"description":  description,
				"category":     category,
				"submitted_by": submittedBy,
			},
		},
		ExpenseID:   expenseID,
		Amount:      amount,
		Description: description,
		Category:    category,
		SubmittedBy: submittedBy,
	}
}
