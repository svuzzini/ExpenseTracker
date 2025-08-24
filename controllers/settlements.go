package controllers

import (
	"expense-tracker/database"
	"expense-tracker/middleware"
	"expense-tracker/models"
	"expense-tracker/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// SettlementController handles settlement-related operations
type SettlementController struct {
	settlementService *services.SettlementService
}

// CreateSettlementRequest represents a custom settlement creation request
type CreateSettlementRequest struct {
	ToUserID uint   `json:"to_user_id" binding:"required"`
	Amount   string `json:"amount" binding:"required"`
	Currency string `json:"currency" binding:"required,len=3"`
	Method   string `json:"method" binding:"max=50"`
}

// CompleteSettlementRequest represents a settlement completion request
type CompleteSettlementRequest struct {
	PaymentReference string `json:"payment_reference" binding:"required,max=100"`
	Method           string `json:"method" binding:"max=50"`
}

// NewSettlementController creates a new settlement controller
func NewSettlementController() *SettlementController {
	return &SettlementController{
		settlementService: services.NewSettlementService(),
	}
}

// GetEventSettlements returns all settlements for an event
func (sc *SettlementController) GetEventSettlements(c *gin.Context) {
	eventID, err := strconv.ParseUint(c.Param("eventId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	settlements, err := sc.settlementService.GetEventSettlements(uint(eventID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get settlements",
			"code":  "FETCH_SETTLEMENTS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, settlements)
}

// GetUserSettlements returns settlements involving the current user
func (sc *SettlementController) GetUserSettlements(c *gin.Context) {
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

	settlements, err := sc.settlementService.GetUserSettlements(uint(eventID), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user settlements",
			"code":  "FETCH_USER_SETTLEMENTS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, settlements)
}

// GetEventBalances returns user balances for an event
func (sc *SettlementController) GetEventBalances(c *gin.Context) {
	eventID, err := strconv.ParseUint(c.Param("eventId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	balances, err := sc.settlementService.CalculateUserBalances(uint(eventID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate balances",
			"code":  "CALCULATE_BALANCES_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, balances)
}

// GetSettlementSummary returns a comprehensive settlement summary
func (sc *SettlementController) GetSettlementSummary(c *gin.Context) {
	eventID, err := strconv.ParseUint(c.Param("eventId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	summary, err := sc.settlementService.GetSettlementSummary(uint(eventID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get settlement summary",
			"code":  "SETTLEMENT_SUMMARY_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GenerateOptimalSettlements generates optimal settlements for an event (admin only)
func (sc *SettlementController) GenerateOptimalSettlements(c *gin.Context) {
	eventID, err := strconv.ParseUint(c.Param("eventId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event ID",
			"code":  "INVALID_EVENT_ID",
		})
		return
	}

	// Generate and create settlements
	settlements, err := sc.settlementService.CreateSettlements(uint(eventID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate settlements",
			"code":  "GENERATE_SETTLEMENTS_ERROR",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Optimal settlements generated successfully",
		"settlements": settlements,
		"count":       len(settlements),
	})
}

// CreateCustomSettlement creates a custom settlement between users
func (sc *SettlementController) CreateCustomSettlement(c *gin.Context) {
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

	var req CreateSettlementRequest
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

	// Create settlement
	settlement, err := sc.settlementService.CreateCustomSettlement(
		uint(eventID),
		user.ID,
		req.ToUserID,
		amount,
		strings.ToUpper(req.Currency),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "SETTLEMENT_CREATION_ERROR",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Settlement created successfully",
		"settlement": settlement,
	})
}

// CompleteSettlement marks a settlement as completed
func (sc *SettlementController) CompleteSettlement(c *gin.Context) {
	settlementID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid settlement ID",
			"code":  "INVALID_SETTLEMENT_ID",
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

	var req CompleteSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
			"code":    "INVALID_REQUEST",
		})
		return
	}

	// Get settlement to verify ownership
	var settlement models.Settlement
	if err := database.DB.First(&settlement, uint(settlementID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Settlement not found",
			"code":  "SETTLEMENT_NOT_FOUND",
		})
		return
	}

	// Verify that the user is either the payer or payee
	if settlement.FromUserID != user.ID && settlement.ToUserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You are not authorized to complete this settlement",
			"code":  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Complete settlement
	if err := sc.settlementService.MarkSettlementCompleted(uint(settlementID), req.PaymentReference); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"code":  "COMPLETE_SETTLEMENT_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settlement marked as completed successfully",
	})
}

// GetSettlementDetails returns detailed information about a settlement
func (sc *SettlementController) GetSettlementDetails(c *gin.Context) {
	settlementID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid settlement ID",
			"code":  "INVALID_SETTLEMENT_ID",
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

	// Get settlement with related data
	var settlement models.Settlement
	if err := database.DB.Preload("FromUser").
		Preload("ToUser").
		Preload("Event").
		First(&settlement, uint(settlementID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Settlement not found",
			"code":  "SETTLEMENT_NOT_FOUND",
		})
		return
	}

	// Check if user is a participant in the event
	if !sc.isParticipant(user.ID, settlement.EventID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You are not a participant in this event",
			"code":  "NOT_PARTICIPANT",
		})
		return
	}

	c.JSON(http.StatusOK, settlement)
}

// Helper functions

// isParticipant checks if a user is a participant in an event
func (sc *SettlementController) isParticipant(userID, eventID uint) bool {
	var participation models.Participation
	err := database.DB.Where("user_id = ? AND event_id = ?", userID, eventID).First(&participation).Error
	return err == nil
}
