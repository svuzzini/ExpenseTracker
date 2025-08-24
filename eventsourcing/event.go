package eventsourcing

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Core event envelope for all domain events
type EventEnvelope struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	EventID       string    `json:"event_id" gorm:"uniqueIndex;not null"`
	AggregateID   string    `json:"aggregate_id" gorm:"index;not null"`
	AggregateType string    `json:"aggregate_type" gorm:"index;not null"`
	EventType     string    `json:"event_type" gorm:"index;not null"`
	EventData     string    `json:"event_data" gorm:"type:text;not null"`
	Metadata      string    `json:"metadata" gorm:"type:text"`
	Version       int       `json:"version" gorm:"not null"`
	Timestamp     time.Time `json:"timestamp" gorm:"not null"`
	ActorID       string    `json:"actor_id" gorm:"index"`
	CreatedAt     time.Time `json:"created_at"`
}

func (EventEnvelope) TableName() string {
	return "event_store"
}

// Snapshot for aggregate state optimization
type Snapshot struct {
	ID            uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AggregateID   string    `json:"aggregate_id" gorm:"uniqueIndex;not null"`
	AggregateType string    `json:"aggregate_type" gorm:"not null"`
	Version       int       `json:"version" gorm:"not null"`
	Data          string    `json:"data" gorm:"type:text;not null"`
	CreatedAt     time.Time `json:"created_at"`
}

// Outbox pattern for reliable event publishing
type OutboxEvent struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	EventID     string    `json:"event_id" gorm:"uniqueIndex;not null"`
	EventType   string    `json:"event_type" gorm:"not null"`
	Payload     string    `json:"payload" gorm:"type:text;not null"`
	Headers     string    `json:"headers" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at"`
	RetryCount  int       `json:"retry_count" gorm:"default:0"`
}

// Base event interface
type Event interface {
	GetEventType() string
	GetAggregateID() string
	GetEventData() map[string]interface{}
}

// Base event struct
type BaseEvent struct {
	AggregateID string                 `json:"aggregate_id"`
	EventType   string                 `json:"event_type"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time             `json:"timestamp"`
}

func (e BaseEvent) GetEventType() string {
	return e.EventType
}

func (e BaseEvent) GetAggregateID() string {
	return e.AggregateID
}

func (e BaseEvent) GetEventData() map[string]interface{} {
	return e.Data
}

// Event store interface
type EventStore interface {
	SaveEvents(aggregateID string, events []Event, expectedVersion int) error
	GetEvents(aggregateID string) ([]EventEnvelope, error)
	GetEventsFromVersion(aggregateID string, version int) ([]EventEnvelope, error)
	SaveSnapshot(snapshot *Snapshot) error
	GetSnapshot(aggregateID string) (*Snapshot, error)
}

// GORM-based event store implementation
type GormEventStore struct {
	db *gorm.DB
}

func NewGormEventStore(db *gorm.DB) *GormEventStore {
	return &GormEventStore{db: db}
}

func (es *GormEventStore) SaveEvents(aggregateID string, events []Event, expectedVersion int) error {
	return es.db.Transaction(func(tx *gorm.DB) error {
		// Check current version for optimistic concurrency control
		var currentVersion int
		err := tx.Model(&EventEnvelope{}).
			Where("aggregate_id = ?", aggregateID).
			Select("COALESCE(MAX(version), 0)").
			Scan(&currentVersion).Error
		
		if err != nil {
			return err
		}
		
		if currentVersion != expectedVersion {
			return ErrConcurrencyConflict
		}
		
		// Save events
		for i, event := range events {
			eventData, _ := json.Marshal(event.GetEventData())
			
			envelope := EventEnvelope{
				EventID:       uuid.New().String(),
				AggregateID:   aggregateID,
				AggregateType: getAggregateType(aggregateID),
				EventType:     event.GetEventType(),
				EventData:     string(eventData),
				Version:       expectedVersion + i + 1,
				Timestamp:     time.Now(),
				CreatedAt:     time.Now(),
			}
			
			if err := tx.Create(&envelope).Error; err != nil {
				return err
			}
		}
		
		return nil
	})
}

func (es *GormEventStore) GetEvents(aggregateID string) ([]EventEnvelope, error) {
	var events []EventEnvelope
	err := es.db.Where("aggregate_id = ?", aggregateID).
		Order("version ASC").
		Find(&events).Error
	return events, err
}

func (es *GormEventStore) GetEventsFromVersion(aggregateID string, version int) ([]EventEnvelope, error) {
	var events []EventEnvelope
	err := es.db.Where("aggregate_id = ? AND version > ?", aggregateID, version).
		Order("version ASC").
		Find(&events).Error
	return events, err
}

func (es *GormEventStore) SaveSnapshot(snapshot *Snapshot) error {
	return es.db.Save(snapshot).Error
}

func (es *GormEventStore) GetSnapshot(aggregateID string) (*Snapshot, error) {
	var snapshot Snapshot
	err := es.db.Where("aggregate_id = ?", aggregateID).
		Order("version DESC").
		First(&snapshot).Error
	return &snapshot, err
}

// Helper functions
func getAggregateType(aggregateID string) string {
	if len(aggregateID) > 0 {
		switch aggregateID[0] {
		case 'u':
			return "User"
		case 'e':
			return "Event"
		case 'x':
			return "Expense"
		default:
			return "Unknown"
		}
	}
	return "Unknown"
}

// Custom errors
var (
	ErrConcurrencyConflict = gorm.ErrInvalidTransaction
)
