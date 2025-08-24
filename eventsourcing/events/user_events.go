package events

import (
	"time"
	"expense-tracker/eventsourcing"
)

// User domain events
type UserRegistered struct {
	eventsourcing.BaseEvent
	Username    string `json:"username"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Currency    string `json:"currency"`
}

func NewUserRegistered(userID, username, email, firstName, lastName, currency string) *UserRegistered {
	return &UserRegistered{
		BaseEvent: eventsourcing.BaseEvent{
			AggregateID: userID,
			EventType:   "UserRegistered",
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"username":   username,
				"email":      email,
				"first_name": firstName,
				"last_name":  lastName,
				"currency":   currency,
			},
		},
		Username:  username,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Currency:  currency,
	}
}

type UserProfileUpdated struct {
	eventsourcing.BaseEvent
	Changes map[string]interface{} `json:"changes"`
}

func NewUserProfileUpdated(userID string, changes map[string]interface{}) *UserProfileUpdated {
	return &UserProfileUpdated{
		BaseEvent: eventsourcing.BaseEvent{
			AggregateID: userID,
			EventType:   "UserProfileUpdated",
			Timestamp:   time.Now(),
			Data:        changes,
		},
		Changes: changes,
	}
}
