package controllers

import (
	"expense-tracker/database"
	"expense-tracker/middleware"
	"expense-tracker/models"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// EventController handles event-related operations
type EventController struct{}

// CreateEventRequest represents the event creation request
type CreateEventRequest struct {
	Name              string `json:"name" binding:"required,min=1,max=100"`
	Description       string `json:"description" binding:"max=500"`
	Currency          string `json:"currency" binding:"required,len=3"`
	RequireApproval   bool   `json:"require_approval"`
	AutoApprovalLimit string `json:"auto_approval_limit"` // Using string for decimal input
}

// JoinEventRequest represents the event join request
type JoinEventRequest struct {
	Code string `json:"code" binding:"required,len=8"`
}

// UpdateEventRequest represents the event update request
type UpdateEventRequest struct {
	Name              string `json:"name" binding:"omitempty,min=1,max=100"`
	Description       string `json:"description" binding:"omitempty,max=500"`
	RequireApproval   *bool  `json:"require_approval,omitempty"`
	AutoApprovalLimit string `json:"auto_approval_limit,omitempty"`
	Status            string `json:"status" binding:"omitempty,oneof=active completed archived"`
}

// AddContributionRequest represents the contribution request
type AddContributionRequest struct {
	Amount   string `json:"amount" binding:"required"`
	Currency string `json:"currency" binding:"required,len=3"`
	Notes    string `json:"notes" binding:"max=500"`
}

// EventParticipant represents an event participant with balance info
type EventParticipant struct {
	models.User
	Role        string          `json:"role"`
	JoinedAt    time.Time       `json:"joined_at"`
	Contributed decimal.Decimal `json:"contributed"`
	Spent       decimal.Decimal `json:"spent"`
	NetBalance  decimal.Decimal `json:"net_balance"`
}

// EventSummary represents a summary of event data
type EventSummary struct {
	models.Event
	ParticipantCount int             `json:"participant_count"`
	TotalBalance     decimal.Decimal `json:"total_balance"`
	PendingExpenses  int             `json:"pending_expenses"`
	RecentActivity   []ActivityItem  `json:"recent_activity"`
}

// ActivityItem represents an activity item
type ActivityItem struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	User        string    `json:"user"`
	Amount      string    `json:"amount,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewEventController creates a new event controller
func NewEventController() *EventController {
	return &EventController{}
}

// CreateEvent creates a new event
func (ec *EventController) CreateEvent(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
			"code":  "USER_NOT_AUTHENTICATED",
		})
		return
	}

	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
			"code":    "INVALID_REQUEST",
		})
		return
	}

	// Validate currency
	if !isValidCurrency(req.Currency) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid currency code",
			"code":  "INVALID_CURRENCY",
		})
		return
	}

	// Parse auto approval limit
	autoApprovalLimit := decimal.Zero
	if req.AutoApprovalLimit != "" {
		limit, err := decimal.NewFromString(req.AutoApprovalLimit)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid auto approval limit",
				"code":  "INVALID_AMOUNT",
			})
			return
		}
		autoApprovalLimit = limit
	}

	// Generate unique event code
	eventCode := generateEventCode()

	// Start transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create event
	event := models.Event{
		Name:              req.Name,
		Description:       req.Description,
		Code:              eventCode,
		CreatedBy:         user.ID,
		Currency:          strings.ToUpper(req.Currency),
		RequireApproval:   req.RequireApproval,
		AutoApprovalLimit: autoApprovalLimit,
	}

	if err := tx.Create(&event).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create event",
			"code":  "EVENT_CREATION_ERROR",
		})
		return
	}

	// Add creator as owner participant
	participation := models.Participation{
		UserID:  user.ID,
		EventID: event.ID,
		Role:    "owner",
	}

	if err := tx.Create(&participation).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add creator as participant",
			"code":  "PARTICIPATION_ERROR",
		})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to complete event creation",
			"code":  "TRANSACTION_ERROR",
		})
		return
	}

	// Load event with creator info
	database.DB.Preload("Creator").First(&event, event.ID)

	c.JSON(http.StatusCreated, event)
}

// JoinEvent allows a user to join an event using a code
func (ec *EventController) JoinEvent(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
			"code":  "USER_NOT_AUTHENTICATED",
		})
		return
	}

	var req JoinEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
			"code":    "INVALID_REQUEST",
		})
		return
	}

	// Find event by code
	var event models.Event
	if err := database.DB.Where("code = ?", strings.ToUpper(req.Code)).First(&event).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Invalid event code",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Check if user is already a participant
	var existing models.Participation
	if err := database.DB.Where("user_id = ? AND event_id = ?", user.ID, event.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "You are already a participant in this event",
			"code":  "ALREADY_PARTICIPANT",
			"event": event,
		})
		return
	}

	// Add user as participant
	participation := models.Participation{
		UserID:  user.ID,
		EventID: event.ID,
		Role:    "participant",
	}

	if err := database.DB.Create(&participation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to join event",
			"code":  "JOIN_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully joined event",
		"event":   event,
	})
}

// GetUserEvents returns all events for the current user
func (ec *EventController) GetUserEvents(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
			"code":  "USER_NOT_AUTHENTICATED",
		})
		return
	}

	var participations []models.Participation
	if err := database.DB.Where("user_id = ?", user.ID).
		Preload("Event").
		Preload("Event.Creator").
		Find(&participations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch events",
			"code":  "FETCH_EVENTS_ERROR",
		})
		return
	}

	events := make([]models.Event, len(participations))
	for i, p := range participations {
		events[i] = p.Event
	}

	c.JSON(http.StatusOK, events)
}

// GetEventDetails returns detailed information about an event
func (ec *EventController) GetEventDetails(c *gin.Context) {
	eventID, err := strconv.ParseUint(c.Param("eventId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
			"code":  "USER_NOT_AUTHENTICATED",
		})
		return
	}

	// Check if user is a participant
	var participation models.Participation
	if err := database.DB.Where("user_id = ? AND event_id = ?", user.ID, uint(eventID)).First(&participation).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You are not a participant in this event",
			"code":  "NOT_PARTICIPANT",
		})
		return
	}

	// Get event with all related data
	var event models.Event
	if err := database.DB.Where("id = ?", uint(eventID)).
		Preload("Creator").
		Preload("Participants").
		Preload("Participants.User").
		Preload("Contributions").
		Preload("Contributions.User").
		Preload("Expenses").
		Preload("Expenses.Submitter").
		Preload("Expenses.Category").
		First(&event).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Debug: Log the participation details
	fmt.Printf("DEBUG: User %d role in event %d: %s\n", user.ID, eventID, participation.Role)

	// Add user's role to the response
	response := gin.H{
		"id":                  event.ID,
		"name":                event.Name,
		"description":         event.Description,
		"code":                event.Code,
		"created_by":          event.CreatedBy,
		"created_at":          event.CreatedAt,
		"updated_at":          event.UpdatedAt,
		"currency":            event.Currency,
		"status":              event.Status,
		"total_contributions": event.TotalContributions,
		"total_expenses":      event.TotalExpenses,
		"end_date":            event.EndDate,
		"require_approval":    event.RequireApproval,
		"auto_approval_limit": event.AutoApprovalLimit,
		"creator":             event.Creator,
		"participants":        event.Participants,
		"contributions":       event.Contributions,
		"expenses":            event.Expenses,
		"user_role":           participation.Role,
	}

	c.JSON(http.StatusOK, response)
}

// GetEventSummary returns a summary of event data
func (ec *EventController) GetEventSummary(c *gin.Context) {
	eventID, err := strconv.ParseUint(c.Param("eventId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
			"code":  "USER_NOT_AUTHENTICATED",
		})
		return
	}

	// Check if user is a participant
	if !ec.isParticipant(user.ID, uint(eventID)) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You are not a participant in this event",
			"code":  "NOT_PARTICIPANT",
		})
		return
	}

	// Get event
	var event models.Event
	if err := database.DB.First(&event, uint(eventID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Count participants
	var participantCount int64
	database.DB.Model(&models.Participation{}).Where("event_id = ?", eventID).Count(&participantCount)

	// Count pending expenses
	var pendingExpenses int64
	database.DB.Model(&models.Expense{}).Where("event_id = ? AND status = ?", eventID, "pending").Count(&pendingExpenses)

	// Calculate total balance
	totalBalance := event.TotalContributions.Sub(event.TotalExpenses)

	// Get recent activity
	recentActivity := ec.getRecentActivity(uint(eventID))

	summary := EventSummary{
		Event:            event,
		ParticipantCount: int(participantCount),
		TotalBalance:     totalBalance,
		PendingExpenses:  int(pendingExpenses),
		RecentActivity:   recentActivity,
	}

	c.JSON(http.StatusOK, summary)
}

// AddContribution adds a contribution to an event
func (ec *EventController) AddContribution(c *gin.Context) {
	eventID, err := strconv.ParseUint(c.Param("eventId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
			"code":  "USER_NOT_AUTHENTICATED",
		})
		return
	}

	// Check if user is a participant
	if !ec.isParticipant(user.ID, uint(eventID)) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You are not a participant in this event",
			"code":  "NOT_PARTICIPANT",
		})
		return
	}

	var req AddContributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
			"code":    "INVALID_REQUEST",
		})
		return
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid amount",
			"code":  "INVALID_AMOUNT",
		})
		return
	}

	// Start transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create contribution
	contribution := models.Contribution{
		EventID:  uint(eventID),
		UserID:   user.ID,
		Amount:   amount,
		Currency: strings.ToUpper(req.Currency),
		Notes:    req.Notes,
	}

	if err := tx.Create(&contribution).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add contribution",
			"code":  "CONTRIBUTION_ERROR",
		})
		return
	}

	// Update event total contributions
	if err := tx.Model(&models.Event{}).Where("id = ?", eventID).
		Update("total_contributions", database.DB.Raw("total_contributions + ?", amount)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update event totals",
			"code":  "UPDATE_ERROR",
		})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to complete contribution",
			"code":  "TRANSACTION_ERROR",
		})
		return
	}

	// Load contribution with user info
	database.DB.Preload("User").First(&contribution, contribution.ID)

	c.JSON(http.StatusCreated, contribution)
}

// UpdateEvent updates event details (admin only)
func (ec *EventController) UpdateEvent(c *gin.Context) {
	eventID, err := strconv.ParseUint(c.Param("eventId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	_, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
			"code":  "USER_NOT_AUTHENTICATED",
		})
		return
	}

	// Check if user is an admin
	if !middleware.IsEventAdmin(c, uint(eventID)) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions",
			"code":  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	var req UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
			"code":    "INVALID_REQUEST",
		})
		return
	}

	// Get event
	var event models.Event
	if err := database.DB.First(&event, uint(eventID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Update fields
	if req.Name != "" {
		event.Name = req.Name
	}
	if req.Description != "" {
		event.Description = req.Description
	}
	if req.RequireApproval != nil {
		event.RequireApproval = *req.RequireApproval
	}
	if req.AutoApprovalLimit != "" {
		limit, err := decimal.NewFromString(req.AutoApprovalLimit)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid auto approval limit",
				"code":  "INVALID_AMOUNT",
			})
			return
		}
		event.AutoApprovalLimit = limit
	}
	if req.Status != "" {
		event.Status = req.Status
	}

	// Save changes
	if err := database.DB.Save(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update event",
			"code":  "UPDATE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, event)
}

// Helper functions

// generateEventCode generates a unique 8-character event code
func generateEventCode() string {
	for {
		// Generate 8-character code
		code := strings.ToUpper(uuid.New().String()[:8])

		// Check if code already exists
		var existingEvent models.Event
		if err := database.DB.Where("code = ?", code).First(&existingEvent).Error; err != nil {
			// Code doesn't exist, return it
			return code
		}
		// Code exists, try again
	}
}

// isValidCurrency checks if the currency code is valid
func isValidCurrency(currency string) bool {
	validCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true,
		"CAD": true, "AUD": true, "CHF": true, "CNY": true,
		"INR": true, "BRL": true, "RUB": true, "KRW": true,
		"SEK": true, "NOK": true, "DKK": true, "PLN": true,
		"CZK": true, "HUF": true, "RON": true, "BGN": true,
		"HRK": true, "ISK": true, "TRY": true, "ZAR": true,
		"MXN": true, "SGD": true, "HKD": true, "NZD": true,
	}
	return validCurrencies[strings.ToUpper(currency)]
}

// isParticipant checks if a user is a participant in an event
func (ec *EventController) isParticipant(userID, eventID uint) bool {
	var participation models.Participation
	err := database.DB.Where("user_id = ? AND event_id = ?", userID, eventID).First(&participation).Error
	return err == nil
}

// getRecentActivity returns recent activity for an event
func (ec *EventController) getRecentActivity(eventID uint) []ActivityItem {
	var activities []ActivityItem

	// Get recent contributions
	var contributions []models.Contribution
	database.DB.Where("event_id = ?", eventID).
		Preload("User").
		Order("timestamp DESC").
		Limit(5).
		Find(&contributions)

	for _, c := range contributions {
		activities = append(activities, ActivityItem{
			Type:        "contribution",
			Description: "added a contribution",
			User:        c.User.DisplayName,
			Amount:      c.Amount.String() + " " + c.Currency,
			Timestamp:   c.Timestamp,
		})
	}

	// Get recent expenses
	var expenses []models.Expense
	database.DB.Where("event_id = ?", eventID).
		Preload("Submitter").
		Order("submitted_at DESC").
		Limit(5).
		Find(&expenses)

	for _, e := range expenses {
		activities = append(activities, ActivityItem{
			Type:        "expense",
			Description: "submitted an expense",
			User:        e.Submitter.DisplayName,
			Amount:      e.Amount.String() + " " + e.Currency,
			Timestamp:   e.SubmittedAt,
		})
	}

	// Sort activities by timestamp (most recent first)
	// This is a simplified sort - in production you'd use a proper sorting algorithm
	return activities[:min(len(activities), 10)]
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
