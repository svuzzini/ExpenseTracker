package aggregates

import (
	"expense-tracker/eventsourcing"
	"expense-tracker/eventsourcing/events"
	"time"
	"fmt"
)

type EventAggregate struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Code            string    `json:"code"`
	CreatedBy       string    `json:"created_by"`
	Currency        string    `json:"currency"`
	RequireApproval bool      `json:"require_approval"`
	Status          string    `json:"status"`
	Participants    []string  `json:"participants"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Version         int       `json:"version"`
	
	uncommittedEvents []eventsourcing.Event
}

func NewEventAggregate(id string) *EventAggregate {
	return &EventAggregate{
		ID:                id,
		Status:            "active",
		Participants:      make([]string, 0),
		Version:           0,
		uncommittedEvents: make([]eventsourcing.Event, 0),
	}
}

func (e *EventAggregate) GetID() string {
	return e.ID
}

func (e *EventAggregate) GetVersion() int {
	return e.Version
}

func (e *EventAggregate) GetUncommittedEvents() []eventsourcing.Event {
	return e.uncommittedEvents
}

func (e *EventAggregate) ClearUncommittedEvents() {
	e.uncommittedEvents = make([]eventsourcing.Event, 0)
}

// Command handlers
func (e *EventAggregate) CreateEvent(name, description, code, createdBy, currency string, requireApproval bool) error {
	// Business logic validation
	if e.Name != "" {
		return fmt.Errorf("event already exists")
	}
	
	// Create and apply event
	event := events.NewEventCreated(e.ID, name, description, code, createdBy, currency, requireApproval)
	e.applyEvent(event)
	
	return nil
}

func (e *EventAggregate) AddParticipant(userID, role string) error {
	// Business logic validation
	if e.Name == "" {
		return fmt.Errorf("event does not exist")
	}
	
	// Check if user is already a participant
	for _, participant := range e.Participants {
		if participant == userID {
			return fmt.Errorf("user is already a participant")
		}
	}
	
	// Create and apply event
	event := events.NewUserJoinedEvent(e.ID, userID, role)
	e.applyEvent(event)
	
	return nil
}

func (e *EventAggregate) AddExpense(expenseID, description, category, submittedBy string, amount float64) error {
	// Business logic validation
	if e.Name == "" {
		return fmt.Errorf("event does not exist")
	}
	
	// Check if user is a participant
	isParticipant := false
	for _, participant := range e.Participants {
		if participant == submittedBy {
			isParticipant = true
			break
		}
	}
	
	if !isParticipant {
		return fmt.Errorf("user is not a participant of this event")
	}
	
	// Create and apply event
	event := events.NewExpenseAdded(e.ID, expenseID, description, category, submittedBy, amount)
	e.applyEvent(event)
	
	return nil
}

// Event application
func (e *EventAggregate) applyEvent(event eventsourcing.Event) {
	e.Apply(event)
	e.uncommittedEvents = append(e.uncommittedEvents, event)
}

func (e *EventAggregate) Apply(event eventsourcing.Event) {
	switch evt := event.(type) {
	case *events.EventCreated:
		e.Name = evt.Name
		e.Description = evt.Description
		e.Code = evt.Code
		e.CreatedBy = evt.CreatedBy
		e.Currency = evt.Currency
		e.RequireApproval = evt.RequireApproval
		e.CreatedAt = evt.Timestamp
		e.UpdatedAt = evt.Timestamp
		e.Participants = append(e.Participants, evt.CreatedBy)
	case *events.UserJoinedEvent:
		e.Participants = append(e.Participants, evt.UserID)
		e.UpdatedAt = evt.Timestamp
	case *events.ExpenseAdded:
		e.UpdatedAt = evt.Timestamp
	}
	
	e.Version++
}

// Replay events to rebuild aggregate state
func (e *EventAggregate) ReplayEvents(events []eventsourcing.EventEnvelope) error {
	for _, envelope := range events {
		var event eventsourcing.Event
		switch envelope.EventType {
		case "EventCreated":
			event = &events.EventCreated{}
		case "UserJoinedEvent":
			event = &events.UserJoinedEvent{}
		case "ExpenseAdded":
			event = &events.ExpenseAdded{}
		default:
			continue
		}
		
		e.Apply(event)
	}
	
	return nil
}
