package projections

import (
	"time"

	"gorm.io/gorm"
)

// Read model for user queries
type UserReadModel struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex"`
	Email     string    `json:"email" gorm:"uniqueIndex"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int       `json:"version"`
}

func (UserReadModel) TableName() string {
	return "user_projections"
}

// User projection handler
type UserProjectionHandler struct {
	db *gorm.DB
}

func NewUserProjectionHandler(db *gorm.DB) *UserProjectionHandler {
	return &UserProjectionHandler{db: db}
}

func (h *UserProjectionHandler) Handle(eventType, aggregateID string, eventData map[string]interface{}, timestamp time.Time, version int) error {
	switch eventType {
	case "UserRegistered":
		return h.handleUserRegistered(aggregateID, eventData, timestamp, version)
	case "UserProfileUpdated":
		return h.handleUserProfileUpdated(aggregateID, eventData, timestamp, version)
	}
	return nil
}

func (h *UserProjectionHandler) handleUserRegistered(aggregateID string, eventData map[string]interface{}, timestamp time.Time, version int) error {
	username, _ := eventData["username"].(string)
	email, _ := eventData["email"].(string)
	firstName, _ := eventData["first_name"].(string)
	lastName, _ := eventData["last_name"].(string)
	currency, _ := eventData["currency"].(string)

	readModel := UserReadModel{
		ID:        aggregateID,
		Username:  username,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Currency:  currency,
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
		Version:   version,
	}

	return h.db.Create(&readModel).Error
}

func (h *UserProjectionHandler) handleUserProfileUpdated(aggregateID string, eventData map[string]interface{}, timestamp time.Time, version int) error {
	updates := map[string]interface{}{
		"updated_at": timestamp,
		"version":    version,
	}

	// Add specific field updates from event data
	for key, value := range eventData {
		updates[key] = value
	}

	return h.db.Model(&UserReadModel{}).
		Where("id = ?", aggregateID).
		Updates(updates).Error
}

// Query methods
func (h *UserProjectionHandler) GetUser(id string) (*UserReadModel, error) {
	var user UserReadModel
	err := h.db.Where("id = ?", id).First(&user).Error
	return &user, err
}

func (h *UserProjectionHandler) GetUserByEmail(email string) (*UserReadModel, error) {
	var user UserReadModel
	err := h.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (h *UserProjectionHandler) GetUserByUsername(username string) (*UserReadModel, error) {
	var user UserReadModel
	err := h.db.Where("username = ?", username).First(&user).Error
	return &user, err
}
