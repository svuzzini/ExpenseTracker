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
	"github.com/shopspring/decimal"
)

// ExpenseController handles expense-related operations
type ExpenseController struct{}

// CreateExpenseRequest represents the expense creation request
type CreateExpenseRequest struct {
	Amount       string               `json:"amount" binding:"required"`
	Currency     string               `json:"currency" binding:"required,len=3"`
	Description  string               `json:"description" binding:"required,min=1,max=255"`
	CategoryID   uint                 `json:"category_id" binding:"required"`
	Date         time.Time            `json:"date" binding:"required"`
	Location     string               `json:"location" binding:"max=200"`
	Vendor       string               `json:"vendor" binding:"max=100"`
	Notes        string               `json:"notes" binding:"max=500"`
	SplitType    string               `json:"split_type" binding:"required,oneof=equal percentage custom weighted"`
	Participants []ExpenseParticipant `json:"participants"`
}

// ExpenseParticipant represents a participant in expense splitting
type ExpenseParticipant struct {
	UserID     uint   `json:"user_id" binding:"required"`
	Amount     string `json:"amount,omitempty"`     // For custom split
	Percentage string `json:"percentage,omitempty"` // For percentage split
	Weight     string `json:"weight,omitempty"`     // For weighted split
}

// ReviewExpenseRequest represents the expense review request
type ReviewExpenseRequest struct {
	Action          string `json:"action" binding:"required,oneof=approve reject"`
	RejectionReason string `json:"rejection_reason"`
}

// UpdateExpenseRequest represents the expense update request
type UpdateExpenseRequest struct {
	Amount      string    `json:"amount"`
	Description string    `json:"description" binding:"omitempty,min=1,max=255"`
	CategoryID  uint      `json:"category_id"`
	Date        time.Time `json:"date"`
	Location    string    `json:"location" binding:"omitempty,max=200"`
	Vendor      string    `json:"vendor" binding:"omitempty,max=100"`
	Notes       string    `json:"notes" binding:"omitempty,max=500"`
}

// ExpenseResponse represents the expense response with calculated shares
type ExpenseResponse struct {
	models.Expense
	Shares []models.ExpenseShare `json:"shares"`
}

// NewExpenseController creates a new expense controller
func NewExpenseController() *ExpenseController {
	return &ExpenseController{}
}

// CreateExpense creates a new expense
func (ec *ExpenseController) CreateExpense(c *gin.Context) {
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
	participation, exists := middleware.GetParticipation(c)
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You are not a participant in this event",
			"code":  "NOT_PARTICIPANT",
		})
		return
	}

	var req CreateExpenseRequest
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

	// Validate category exists
	var category models.ExpenseCategory
	if err := database.DB.First(&category, req.CategoryID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category",
			"code":  "INVALID_CATEGORY",
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

	// Determine initial status
	status := "pending"

	// Check for auto-approval
	var event models.Event
	if err := tx.First(&event, uint(eventID)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Event not found",
			"code":  "EVENT_NOT_FOUND",
		})
		return
	}

	// Auto-approve if conditions are met
	if !event.RequireApproval ||
		(event.AutoApprovalLimit.GreaterThan(decimal.Zero) && amount.LessThanOrEqual(event.AutoApprovalLimit)) ||
		participation.CanApproveExpenses() {
		status = "approved"
	}

	// Create expense
	expense := models.Expense{
		EventID:     uint(eventID),
		SubmittedBy: user.ID,
		CategoryID:  req.CategoryID,
		Amount:      amount,
		Currency:    strings.ToUpper(req.Currency),
		Description: req.Description,
		Date:        req.Date,
		Location:    req.Location,
		Vendor:      req.Vendor,
		Notes:       req.Notes,
		SplitType:   req.SplitType,
		Status:      status,
	}

	if status == "approved" {
		now := time.Now()
		expense.ReviewedBy = &user.ID
		expense.ReviewedAt = &now
	}

	if err := tx.Create(&expense).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create expense",
			"code":  "EXPENSE_CREATION_ERROR",
		})
		return
	}

	// Calculate and create expense shares
	shares, err := ec.calculateExpenseShares(expense, req.Participants, uint(eventID))
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "SHARE_CALCULATION_ERROR",
		})
		return
	}

	// Create expense shares
	for _, share := range shares {
		share.ExpenseID = expense.ID
		if err := tx.Create(&share).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create expense shares",
				"code":  "SHARE_CREATION_ERROR",
			})
			return
		}
	}

	// Update event total expenses if approved
	if status == "approved" {
		if err := tx.Model(&event).Update("total_expenses",
			database.DB.Raw("total_expenses + ?", amount)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update event totals",
				"code":  "UPDATE_ERROR",
			})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to complete expense creation",
			"code":  "TRANSACTION_ERROR",
		})
		return
	}

	// Load expense with related data
	database.DB.Preload("Submitter").
		Preload("Category").
		Preload("Shares").
		Preload("Shares.User").
		First(&expense, expense.ID)

	response := ExpenseResponse{
		Expense: expense,
		Shares:  expense.Shares,
	}

	c.JSON(http.StatusCreated, response)
}

// GetEventExpenses returns all expenses for an event
func (ec *ExpenseController) GetEventExpenses(c *gin.Context) {
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

	// Get query parameters
	status := c.Query("status")
	categoryID := c.Query("category_id")
	submittedBy := c.Query("submitted_by")

	// Build query
	query := database.DB.Where("event_id = ?", uint(eventID))

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	if submittedBy != "" {
		query = query.Where("submitted_by = ?", submittedBy)
	}

	var expenses []models.Expense
	if err := query.Preload("Submitter").
		Preload("Category").
		Preload("Reviewer").
		Preload("Shares").
		Preload("Shares.User").
		Order("submitted_at DESC").
		Find(&expenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch expenses",
			"code":  "FETCH_EXPENSES_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, expenses)
}

// GetExpenseDetails returns detailed information about an expense
func (ec *ExpenseController) GetExpenseDetails(c *gin.Context) {
	expenseID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid expense ID",
			"code":  "INVALID_EXPENSE_ID",
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

	// Get expense
	var expense models.Expense
	if err := database.DB.Preload("Submitter").
		Preload("Category").
		Preload("Reviewer").
		Preload("Shares").
		Preload("Shares.User").
		Preload("Event").
		First(&expense, uint(expenseID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Expense not found",
			"code":  "EXPENSE_NOT_FOUND",
		})
		return
	}

	// Check if user is a participant in the event
	if !ec.isParticipant(user.ID, expense.EventID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You are not a participant in this event",
			"code":  "NOT_PARTICIPANT",
		})
		return
	}

	response := ExpenseResponse{
		Expense: expense,
		Shares:  expense.Shares,
	}

	c.JSON(http.StatusOK, response)
}

// ReviewExpense allows admins to approve or reject expenses
func (ec *ExpenseController) ReviewExpense(c *gin.Context) {
	expenseID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid expense ID",
			"code":  "INVALID_EXPENSE_ID",
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

	var req ReviewExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
			"code":    "INVALID_REQUEST",
		})
		return
	}

	// Get expense
	var expense models.Expense
	if err := database.DB.First(&expense, uint(expenseID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Expense not found",
			"code":  "EXPENSE_NOT_FOUND",
		})
		return
	}

	// Check if user can approve expenses in this event
	if !middleware.CanApproveExpenses(c, expense.EventID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions to review expenses",
			"code":  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Check if expense is pending
	if expense.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Expense is not pending review",
			"code":  "EXPENSE_NOT_PENDING",
		})
		return
	}

	// Validate rejection reason if rejecting
	if req.Action == "reject" && strings.TrimSpace(req.RejectionReason) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Rejection reason is required",
			"code":  "REJECTION_REASON_REQUIRED",
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

	// Update expense
	now := time.Now()
	expense.Status = req.Action + "d" // "approved" or "rejected"
	expense.ReviewedBy = &user.ID
	expense.ReviewedAt = &now
	if req.Action == "reject" {
		expense.RejectionReason = req.RejectionReason
	}

	if err := tx.Save(&expense).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update expense",
			"code":  "UPDATE_ERROR",
		})
		return
	}

	// Update event total expenses if approved
	if req.Action == "approve" {
		var event models.Event
		if err := tx.First(&event, expense.EventID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to find event",
				"code":  "EVENT_NOT_FOUND",
			})
			return
		}

		if err := tx.Model(&event).Update("total_expenses",
			database.DB.Raw("total_expenses + ?", expense.Amount)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update event totals",
				"code":  "UPDATE_ERROR",
			})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to complete expense review",
			"code":  "TRANSACTION_ERROR",
		})
		return
	}

	// Load updated expense
	database.DB.Preload("Submitter").
		Preload("Category").
		Preload("Reviewer").
		First(&expense, expense.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Expense " + expense.Status + " successfully",
		"expense": expense,
	})
}

// UpdateExpense updates an expense (only by submitter, and only if pending)
func (ec *ExpenseController) UpdateExpense(c *gin.Context) {
	expenseID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid expense ID",
			"code":  "INVALID_EXPENSE_ID",
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

	var req UpdateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
			"code":    "INVALID_REQUEST",
		})
		return
	}

	// Get expense
	var expense models.Expense
	if err := database.DB.First(&expense, uint(expenseID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Expense not found",
			"code":  "EXPENSE_NOT_FOUND",
		})
		return
	}

	// Check if user is the submitter
	if expense.SubmittedBy != user.ID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You can only update your own expenses",
			"code":  "NOT_EXPENSE_OWNER",
		})
		return
	}

	// Check if expense is pending
	if expense.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "You can only update pending expenses",
			"code":  "EXPENSE_NOT_PENDING",
		})
		return
	}

	// Update fields
	if req.Amount != "" {
		amount, err := decimal.NewFromString(req.Amount)
		if err != nil || amount.LessThanOrEqual(decimal.Zero) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid amount",
				"code":  "INVALID_AMOUNT",
			})
			return
		}
		expense.Amount = amount
	}

	if req.Description != "" {
		expense.Description = req.Description
	}

	if req.CategoryID != 0 {
		// Validate category exists
		var category models.ExpenseCategory
		if err := database.DB.First(&category, req.CategoryID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid category",
				"code":  "INVALID_CATEGORY",
			})
			return
		}
		expense.CategoryID = req.CategoryID
	}

	if !req.Date.IsZero() {
		expense.Date = req.Date
	}

	expense.Location = req.Location
	expense.Vendor = req.Vendor
	expense.Notes = req.Notes

	// Save changes
	if err := database.DB.Save(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update expense",
			"code":  "UPDATE_ERROR",
		})
		return
	}

	// Load updated expense
	database.DB.Preload("Submitter").
		Preload("Category").
		Preload("Shares").
		Preload("Shares.User").
		First(&expense, expense.ID)

	response := ExpenseResponse{
		Expense: expense,
		Shares:  expense.Shares,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteExpense deletes an expense (only by submitter, and only if pending)
func (ec *ExpenseController) DeleteExpense(c *gin.Context) {
	expenseID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid expense ID",
			"code":  "INVALID_EXPENSE_ID",
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

	// Get expense
	var expense models.Expense
	if err := database.DB.First(&expense, uint(expenseID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Expense not found",
			"code":  "EXPENSE_NOT_FOUND",
		})
		return
	}

	// Check if user is the submitter or an admin
	isOwner := expense.SubmittedBy == user.ID
	isAdmin := middleware.CanApproveExpenses(c, expense.EventID)

	if !isOwner && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions to delete this expense",
			"code":  "INSUFFICIENT_PERMISSIONS",
		})
		return
	}

	// Check if expense is pending (only pending expenses can be deleted)
	if expense.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Only pending expenses can be deleted",
			"code":  "EXPENSE_NOT_PENDING",
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

	// Delete expense shares first
	if err := tx.Where("expense_id = ?", expenseID).Delete(&models.ExpenseShare{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete expense shares",
			"code":  "DELETE_SHARES_ERROR",
		})
		return
	}

	// Delete expense
	if err := tx.Delete(&expense).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete expense",
			"code":  "DELETE_ERROR",
		})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to complete expense deletion",
			"code":  "TRANSACTION_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Expense deleted successfully",
	})
}

// GetExpenseCategories returns all expense categories
func (ec *ExpenseController) GetExpenseCategories(c *gin.Context) {
	var categories []models.ExpenseCategory
	if err := database.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch categories",
			"code":  "FETCH_CATEGORIES_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// Helper functions

// calculateExpenseShares calculates expense shares based on split type
func (ec *ExpenseController) calculateExpenseShares(expense models.Expense, participants []ExpenseParticipant, eventID uint) ([]models.ExpenseShare, error) {
	if len(participants) == 0 {
		// Get all event participants
		var eventParticipants []models.Participation
		if err := database.DB.Where("event_id = ?", eventID).Find(&eventParticipants).Error; err != nil {
			return nil, fmt.Errorf("failed to get event participants")
		}

		// Convert to ExpenseParticipant slice for equal split
		for _, p := range eventParticipants {
			participants = append(participants, ExpenseParticipant{
				UserID: p.UserID,
			})
		}
	}

	var shares []models.ExpenseShare

	switch expense.SplitType {
	case "equal":
		return ec.calculateEqualSplit(expense, participants)
	case "percentage":
		return ec.calculatePercentageSplit(expense, participants)
	case "custom":
		return ec.calculateCustomSplit(expense, participants)
	case "weighted":
		return ec.calculateWeightedSplit(expense, participants)
	default:
		return nil, fmt.Errorf("invalid split type")
	}

	return shares, nil
}

// calculateEqualSplit calculates equal split among participants
func (ec *ExpenseController) calculateEqualSplit(expense models.Expense, participants []ExpenseParticipant) ([]models.ExpenseShare, error) {
	if len(participants) == 0 {
		return nil, fmt.Errorf("no participants specified")
	}

	shareAmount := expense.Amount.Div(decimal.NewFromInt(int64(len(participants))))
	var shares []models.ExpenseShare

	for _, p := range participants {
		shares = append(shares, models.ExpenseShare{
			UserID:     p.UserID,
			Amount:     shareAmount,
			Percentage: decimal.NewFromFloat(100.0 / float64(len(participants))),
		})
	}

	return shares, nil
}

// calculatePercentageSplit calculates percentage-based split
func (ec *ExpenseController) calculatePercentageSplit(expense models.Expense, participants []ExpenseParticipant) ([]models.ExpenseShare, error) {
	var totalPercentage decimal.Decimal
	var shares []models.ExpenseShare

	// Validate percentages and calculate total
	for _, p := range participants {
		if p.Percentage == "" {
			return nil, fmt.Errorf("percentage required for user %d", p.UserID)
		}

		percentage, err := decimal.NewFromString(p.Percentage)
		if err != nil {
			return nil, fmt.Errorf("invalid percentage for user %d", p.UserID)
		}

		totalPercentage = totalPercentage.Add(percentage)
	}

	// Check if percentages add up to 100
	if !totalPercentage.Equal(decimal.NewFromInt(100)) {
		return nil, fmt.Errorf("percentages must add up to 100, got %s", totalPercentage.String())
	}

	// Calculate amounts
	for _, p := range participants {
		percentage, _ := decimal.NewFromString(p.Percentage)
		amount := expense.Amount.Mul(percentage).Div(decimal.NewFromInt(100))

		shares = append(shares, models.ExpenseShare{
			UserID:     p.UserID,
			Amount:     amount,
			Percentage: percentage,
		})
	}

	return shares, nil
}

// calculateCustomSplit calculates custom split with specific amounts
func (ec *ExpenseController) calculateCustomSplit(expense models.Expense, participants []ExpenseParticipant) ([]models.ExpenseShare, error) {
	var totalAmount decimal.Decimal
	var shares []models.ExpenseShare

	// Validate amounts and calculate total
	for _, p := range participants {
		if p.Amount == "" {
			return nil, fmt.Errorf("amount required for user %d", p.UserID)
		}

		amount, err := decimal.NewFromString(p.Amount)
		if err != nil {
			return nil, fmt.Errorf("invalid amount for user %d", p.UserID)
		}

		totalAmount = totalAmount.Add(amount)
	}

	// Check if amounts add up to expense total
	if !totalAmount.Equal(expense.Amount) {
		return nil, fmt.Errorf("custom amounts must add up to expense total %s, got %s",
			expense.Amount.String(), totalAmount.String())
	}

	// Create shares
	for _, p := range participants {
		amount, _ := decimal.NewFromString(p.Amount)
		percentage := amount.Div(expense.Amount).Mul(decimal.NewFromInt(100))

		shares = append(shares, models.ExpenseShare{
			UserID:     p.UserID,
			Amount:     amount,
			Percentage: percentage,
		})
	}

	return shares, nil
}

// calculateWeightedSplit calculates weighted split based on weights
func (ec *ExpenseController) calculateWeightedSplit(expense models.Expense, participants []ExpenseParticipant) ([]models.ExpenseShare, error) {
	var totalWeight decimal.Decimal
	var shares []models.ExpenseShare

	// Validate weights and calculate total
	for _, p := range participants {
		if p.Weight == "" {
			return nil, fmt.Errorf("weight required for user %d", p.UserID)
		}

		weight, err := decimal.NewFromString(p.Weight)
		if err != nil {
			return nil, fmt.Errorf("invalid weight for user %d", p.UserID)
		}

		totalWeight = totalWeight.Add(weight)
	}

	// Calculate amounts based on weights
	for _, p := range participants {
		weight, _ := decimal.NewFromString(p.Weight)
		percentage := weight.Div(totalWeight).Mul(decimal.NewFromInt(100))
		amount := expense.Amount.Mul(weight).Div(totalWeight)

		shares = append(shares, models.ExpenseShare{
			UserID:     p.UserID,
			Amount:     amount,
			Percentage: percentage,
		})
	}

	return shares, nil
}

// isParticipant checks if a user is a participant in an event
func (ec *ExpenseController) isParticipant(userID, eventID uint) bool {
	var participation models.Participation
	err := database.DB.Where("user_id = ? AND event_id = ?", userID, eventID).First(&participation).Error
	return err == nil
}
