package projections

import (
	"gorm.io/gorm"
	"time"
)

// Read model for event queries
type EventReadModel struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Code            string    `json:"code" gorm:"uniqueIndex"`
	CreatedBy       string    `json:"created_by"`
	Currency        string    `json:"currency"`
	RequireApproval bool      `json:"require_approval"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Version         int       `json:"version"`
}

func (EventReadModel) TableName() string {
	return "event_projections"
}

type EventParticipationReadModel struct {
	ID       uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	EventID  string `json:"event_id" gorm:"index"`
	UserID   string `json:"user_id" gorm:"index"`
	Role     string `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

func (EventParticipationReadModel) TableName() string {
	return "event_participation_projections"
}

// Event projection handler
type EventProjectionHandler struct {
	db *gorm.DB
}

func NewEventProjectionHandler(db *gorm.DB) *EventProjectionHandler {
	return &EventProjectionHandler{db: db}
}

func (h *EventProjectionHandler) Handle(eventType, aggregateID string, eventData map[string]interface{}, timestamp time.Time, version int) error {
	switch eventType {
	case "EventCreated":
		return h.handleEventCreated(aggregateID, eventData, timestamp, version)
	case "UserJoinedEvent":
		return h.handleUserJoinedEvent(aggregateID, eventData, timestamp, version)
	case "ExpenseAdded":
		return h.handleExpenseAdded(aggregateID, eventData, timestamp, version)
	}
	return nil
}

func (h *EventProjectionHandler) handleEventCreated(aggregateID string, eventData map[string]interface{}, timestamp time.Time, version int) error {
	// Extract data from event
	name, _ := eventData["name"].(string)
	description, _ := eventData["description"].(string)
	code, _ := eventData["code"].(string)
	createdBy, _ := eventData["created_by"].(string)
	currency, _ := eventData["currency"].(string)
	requireApproval, _ := eventData["require_approval"].(bool)
	
	readModel := EventReadModel{
		ID:              aggregateID,
		Name:            name,
		Description:     description,
		Code:            code,
		CreatedBy:       createdBy,
		Currency:        currency,
		RequireApproval: requireApproval,
		Status:          "active",
		CreatedAt:       timestamp,
		UpdatedAt:       timestamp,
		Version:         version,
	}
	
	// Also create participation record for creator
	participation := EventParticipationReadModel{
		EventID:  aggregateID,
		UserID:   createdBy,
		Role:     "owner",
		JoinedAt: timestamp,
	}
	
	return h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&readModel).Error; err != nil {
			return err
		}
		return tx.Create(&participation).Error
	})
}

func (h *EventProjectionHandler) handleUserJoinedEvent(aggregateID string, eventData map[string]interface{}, timestamp time.Time, version int) error {
	userID, _ := eventData["user_id"].(string)
	role, _ := eventData["role"].(string)
	
	participation := EventParticipationReadModel{
		EventID:  aggregateID,
		UserID:   userID,
		Role:     role,
		JoinedAt: timestamp,
	}
	
	return h.db.Create(&participation).Error
}

func (h *EventProjectionHandler) handleExpenseAdded(aggregateID string, eventData map[string]interface{}, timestamp time.Time, version int) error {
	// Update event's updated_at timestamp
	return h.db.Model(&EventReadModel{}).
		Where("id = ?", aggregateID).
		Updates(map[string]interface{}{
			"updated_at": timestamp,
			"version":    version,
		}).Error
}

// Query methods
func (h *EventProjectionHandler) GetEvent(id string) (*EventReadModel, error) {
	var event EventReadModel
	err := h.db.Where("id = ?", id).First(&event).Error
	return &event, err
}

func (h *EventProjectionHandler) GetEventByCode(code string) (*EventReadModel, error) {
	var event EventReadModel
	err := h.db.Where("code = ?", code).First(&event).Error
	return &event, err
}

func (h *EventProjectionHandler) GetUserEvents(userID string) ([]EventReadModel, error) {
	var events []EventReadModel
	err := h.db.Table("event_projections").
		Joins("JOIN event_participation_projections ON event_projections.id = event_participation_projections.event_id").
		Where("event_participation_projections.user_id = ?", userID).
		Find(&events).Error
	return events, err
}
