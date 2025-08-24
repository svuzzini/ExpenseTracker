package aggregates

import (
	"expense-tracker/eventsourcing"
	"expense-tracker/eventsourcing/events"
	"time"
)

type UserAggregate struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`
	
	// Uncommitted events
	uncommittedEvents []eventsourcing.Event
}

func NewUserAggregate(id string) *UserAggregate {
	return &UserAggregate{
		ID:                id,
		Version:           0,
		uncommittedEvents: make([]eventsourcing.Event, 0),
	}
}

func (u *UserAggregate) GetID() string {
	return u.ID
}

func (u *UserAggregate) GetVersion() int {
	return u.Version
}

func (u *UserAggregate) GetUncommittedEvents() []eventsourcing.Event {
	return u.uncommittedEvents
}

func (u *UserAggregate) ClearUncommittedEvents() {
	u.uncommittedEvents = make([]eventsourcing.Event, 0)
}

// Command handlers
func (u *UserAggregate) Register(username, email, firstName, lastName, currency string) error {
	// Business logic validation
	if u.Username != "" {
		return ErrUserAlreadyExists
	}
	
	// Create and apply event
	event := events.NewUserRegistered(u.ID, username, email, firstName, lastName, currency)
	u.applyEvent(event)
	
	return nil
}

func (u *UserAggregate) UpdateProfile(changes map[string]interface{}) error {
	// Business logic validation
	if u.Username == "" {
		return ErrUserNotFound
	}
	
	// Create and apply event
	event := events.NewUserProfileUpdated(u.ID, changes)
	u.applyEvent(event)
	
	return nil
}

// Event application
func (u *UserAggregate) applyEvent(event eventsourcing.Event) {
	u.Apply(event)
	u.uncommittedEvents = append(u.uncommittedEvents, event)
}

func (u *UserAggregate) Apply(event eventsourcing.Event) {
	switch e := event.(type) {
	case *events.UserRegistered:
		u.Username = e.Username
		u.Email = e.Email
		u.FirstName = e.FirstName
		u.LastName = e.LastName
		u.Currency = e.Currency
		u.CreatedAt = e.Timestamp
		u.UpdatedAt = e.Timestamp
	case *events.UserProfileUpdated:
		if firstName, ok := e.Changes["first_name"].(string); ok {
			u.FirstName = firstName
		}
		if lastName, ok := e.Changes["last_name"].(string); ok {
			u.LastName = lastName
		}
		if currency, ok := e.Changes["currency"].(string); ok {
			u.Currency = currency
		}
		u.UpdatedAt = e.Timestamp
	}
	
	u.Version++
}

// Replay events to rebuild aggregate state
func (u *UserAggregate) ReplayEvents(events []eventsourcing.EventEnvelope) error {
	for _, envelope := range events {
		// Deserialize event based on type
		var event eventsourcing.Event
		switch envelope.EventType {
		case "UserRegistered":
			event = &events.UserRegistered{}
		case "UserProfileUpdated":
			event = &events.UserProfileUpdated{}
		default:
			continue
		}
		
		// Apply event
		u.Apply(event)
	}
	
	return nil
}

// Custom errors
var (
	ErrUserAlreadyExists = NewDomainError("user already exists")
	ErrUserNotFound      = NewDomainError("user not found")
)

type DomainError struct {
	Message string
}

func NewDomainError(message string) *DomainError {
	return &DomainError{Message: message}
}

func (e *DomainError) Error() string {
	return e.Message
}
