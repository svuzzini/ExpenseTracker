package commands

import (
	"time"
	"github.com/google/uuid"
)

// Base command interface
type Command interface {
	GetCommandID() string
	GetAggregateID() string
	GetCommandType() string
	GetTimestamp() time.Time
}

// Base command struct
type BaseCommand struct {
	CommandID   string    `json:"command_id"`
	AggregateID string    `json:"aggregate_id"`
	CommandType string    `json:"command_type"`
	Timestamp   time.Time `json:"timestamp"`
	ActorID     string    `json:"actor_id"`
}

func (c BaseCommand) GetCommandID() string {
	return c.CommandID
}

func (c BaseCommand) GetAggregateID() string {
	return c.AggregateID
}

func (c BaseCommand) GetCommandType() string {
	return c.CommandType
}

func (c BaseCommand) GetTimestamp() time.Time {
	return c.Timestamp
}

// User commands
type RegisterUser struct {
	BaseCommand
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Currency  string `json:"currency"`
}

func NewRegisterUser(userID, username, email, password, firstName, lastName, currency, actorID string) *RegisterUser {
	return &RegisterUser{
		BaseCommand: BaseCommand{
			CommandID:   uuid.New().String(),
			AggregateID: userID,
			CommandType: "RegisterUser",
			Timestamp:   time.Now(),
			ActorID:     actorID,
		},
		Username:  username,
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		Currency:  currency,
	}
}

// Event commands
type CreateEvent struct {
	BaseCommand
	Name            string `json:"name"`
	Description     string `json:"description"`
	Currency        string `json:"currency"`
	RequireApproval bool   `json:"require_approval"`
}

func NewCreateEvent(eventID, name, description, currency, actorID string, requireApproval bool) *CreateEvent {
	return &CreateEvent{
		BaseCommand: BaseCommand{
			CommandID:   uuid.New().String(),
			AggregateID: eventID,
			CommandType: "CreateEvent",
			Timestamp:   time.Now(),
			ActorID:     actorID,
		},
		Name:            name,
		Description:     description,
		Currency:        currency,
		RequireApproval: requireApproval,
	}
}

type JoinEvent struct {
	BaseCommand
	EventCode string `json:"event_code"`
	UserID    string `json:"user_id"`
}

func NewJoinEvent(eventID, eventCode, userID, actorID string) *JoinEvent {
	return &JoinEvent{
		BaseCommand: BaseCommand{
			CommandID:   uuid.New().String(),
			AggregateID: eventID,
			CommandType: "JoinEvent",
			Timestamp:   time.Now(),
			ActorID:     actorID,
		},
		EventCode: eventCode,
		UserID:    userID,
	}
}

type AddExpense struct {
	BaseCommand
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
}

func NewAddExpense(eventID, description, category, actorID string, amount float64) *AddExpense {
	return &AddExpense{
		BaseCommand: BaseCommand{
			CommandID:   uuid.New().String(),
			AggregateID: eventID,
			CommandType: "AddExpense",
			Timestamp:   time.Now(),
			ActorID:     actorID,
		},
		Amount:      amount,
		Description: description,
		Category:    category,
	}
}
