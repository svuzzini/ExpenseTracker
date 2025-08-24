package eventsourcing

import (
	"expense-tracker/eventsourcing/projections"
	"expense-tracker/eventsourcing/handlers"
	"gorm.io/gorm"
)

// Service layer for event sourcing integration
type EventSourcingService struct {
	eventStore        EventStore
	commandHandler    *handlers.BasicCommandHandler
	projectionHandler *projections.EventProjectionHandler
}

func NewEventSourcingService(db *gorm.DB) *EventSourcingService {
	eventStore := NewGormEventStore(db)
	projectionHandler := projections.NewEventProjectionHandler(db)
	commandHandler := handlers.NewBasicCommandHandler()
	
	return &EventSourcingService{
		eventStore:        eventStore,
		commandHandler:    commandHandler,
		projectionHandler: projectionHandler,
	}
}

// Methods to integrate with existing controllers
func (s *EventSourcingService) CreateEvent(eventID, name, description, currency, actorID string, requireApproval bool) error {
	data := map[string]interface{}{
		"event_id":         eventID,
		"name":            name,
		"description":     description,
		"currency":        currency,
		"actor_id":        actorID,
		"require_approval": requireApproval,
	}
	return s.commandHandler.HandleCommand("CreateEvent", data)
}

func (s *EventSourcingService) JoinEvent(eventID, eventCode, userID, actorID string) error {
	data := map[string]interface{}{
		"event_id":   eventID,
		"event_code": eventCode,
		"user_id":    userID,
		"actor_id":   actorID,
	}
	return s.commandHandler.HandleCommand("JoinEvent", data)
}

func (s *EventSourcingService) AddExpense(eventID, description, category, actorID string, amount float64) error {
	data := map[string]interface{}{
		"event_id":    eventID,
		"description": description,
		"category":    category,
		"actor_id":    actorID,
		"amount":      amount,
	}
	return s.commandHandler.HandleCommand("AddExpense", data)
}

// Query methods
func (s *EventSourcingService) GetEventByCode(code string) (*projections.EventReadModel, error) {
	return s.projectionHandler.GetEventByCode(code)
}

func (s *EventSourcingService) GetUserEvents(userID string) ([]projections.EventReadModel, error) {
	return s.projectionHandler.GetUserEvents(userID)
}
