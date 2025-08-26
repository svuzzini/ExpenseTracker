package services

import (
	"expense-tracker/database"
	"expense-tracker/models"
	"fmt"
	"sort"

	"github.com/shopspring/decimal"
)

// SettlementService handles settlement calculations and optimizations
type SettlementService struct{}

// NewSettlementService creates a new settlement service
func NewSettlementService() *SettlementService {
	return &SettlementService{}
}

// CalculateUserBalances calculates balances for all users in an event
func (s *SettlementService) CalculateUserBalances(eventID uint) ([]models.UserBalance, error) {
	// Get all participants
	var participants []models.Participation
	if err := database.DB.Where("event_id = ?", eventID).
		Preload("User").Find(&participants).Error; err != nil {
		return nil, fmt.Errorf("failed to get participants: %v", err)
	}

	balances := make([]models.UserBalance, len(participants))

	for i, p := range participants {
		// Calculate total contributions
		var totalContributions decimal.Decimal
		database.DB.Model(&models.Contribution{}).
			Where("event_id = ? AND user_id = ?", eventID, p.UserID).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&totalContributions)

		// Calculate total expenses (approved and pending)
		var totalExpenses decimal.Decimal
		database.DB.Model(&models.Expense{}).
			Where("event_id = ? AND submitted_by = ? AND status IN (?)", eventID, p.UserID, []string{"approved", "pending"}).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&totalExpenses)

		// Calculate share of total expenses (what they should pay)
		var totalEventExpenses decimal.Decimal
		database.DB.Model(&models.ExpenseShare{}).
			Joins("JOIN expenses ON expense_shares.expense_id = expenses.id").
			Where("expenses.event_id = ? AND expense_shares.user_id = ? AND expenses.status IN (?)",
				eventID, p.UserID, []string{"approved", "pending"}).
			Select("COALESCE(SUM(expense_shares.amount), 0)").
			Scan(&totalEventExpenses)

		netBalance := totalContributions.Add(totalExpenses).Sub(totalEventExpenses)

		var owesAmount decimal.Decimal
		var owedAmount decimal.Decimal

		if netBalance.IsNegative() {
			owesAmount = netBalance.Abs()
		} else if netBalance.IsPositive() {
			owedAmount = netBalance
		}

		balances[i] = models.UserBalance{
			UserID:      p.UserID,
			Username:    p.User.Username,
			DisplayName: p.User.DisplayName,
			Contributed: totalContributions,
			Spent:       totalExpenses,
			NetBalance:  netBalance,
			OwesAmount:  owesAmount,
			OwedAmount:  owedAmount,
		}
	}

	return balances, nil
}

// GenerateOptimalSettlements generates optimal settlement transactions
func (s *SettlementService) GenerateOptimalSettlements(eventID uint) ([]models.Settlement, error) {
	// Get user balances
	balances, err := s.CalculateUserBalances(eventID)
	if err != nil {
		return nil, err
	}

	// Get event details for currency
	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		return nil, fmt.Errorf("failed to get event: %v", err)
	}

	// Separate debtors and creditors
	var debtors []models.UserBalance
	var creditors []models.UserBalance

	for _, balance := range balances {
		if balance.NetBalance.IsNegative() {
			debtors = append(debtors, balance)
		} else if balance.NetBalance.IsPositive() {
			creditors = append(creditors, balance)
		}
	}

	// Sort debtors by amount owed (descending)
	sort.Slice(debtors, func(i, j int) bool {
		return debtors[i].OwesAmount.GreaterThan(debtors[j].OwesAmount)
	})

	// Sort creditors by amount owed (descending)
	sort.Slice(creditors, func(i, j int) bool {
		return creditors[i].OwedAmount.GreaterThan(creditors[j].OwedAmount)
	})

	// Generate settlements using greedy algorithm
	var settlements []models.Settlement

	for len(debtors) > 0 && len(creditors) > 0 {
		debtor := &debtors[0]
		creditor := &creditors[0]

		// Calculate settlement amount (minimum of what debtor owes and creditor is owed)
		settlementAmount := decimal.Min(debtor.OwesAmount, creditor.OwedAmount)

		// Create settlement
		settlement := models.Settlement{
			EventID:    eventID,
			FromUserID: debtor.UserID,
			ToUserID:   creditor.UserID,
			Amount:     settlementAmount,
			Currency:   event.Currency,
			Status:     "pending",
		}
		settlements = append(settlements, settlement)

		// Update balances
		debtor.OwesAmount = debtor.OwesAmount.Sub(settlementAmount)
		creditor.OwedAmount = creditor.OwedAmount.Sub(settlementAmount)

		// Remove settled parties
		if debtor.OwesAmount.IsZero() {
			debtors = debtors[1:]
		}
		if creditor.OwedAmount.IsZero() {
			creditors = creditors[1:]
		}
	}

	return settlements, nil
}

// CreateSettlements creates settlement records in the database
func (s *SettlementService) CreateSettlements(eventID uint) ([]models.Settlement, error) {
	fmt.Printf("DEBUG: Starting CreateSettlements for eventID: %d\n", eventID)

	// Clear existing pending settlements
	deleteResult := database.DB.Where("event_id = ? AND status = ?", eventID, "pending").Delete(&models.Settlement{})
	fmt.Printf("DEBUG: Cleared %d existing pending settlements\n", deleteResult.RowsAffected)

	// Generate optimal settlements
	settlements, err := s.GenerateOptimalSettlements(eventID)
	if err != nil {
		fmt.Printf("DEBUG: Error generating optimal settlements: %v\n", err)
		return nil, err
	}
	fmt.Printf("DEBUG: Generated %d settlements\n", len(settlements))

	// Create settlements in database
	for i, settlement := range settlements {
		fmt.Printf("DEBUG: Creating settlement %d: From=%d To=%d Amount=%s\n", i+1, settlement.FromUserID, settlement.ToUserID, settlement.Amount)
		if err := database.DB.Create(&settlement).Error; err != nil {
			fmt.Printf("DEBUG: Error creating settlement: %v\n", err)
			return nil, fmt.Errorf("failed to create settlement: %v", err)
		}
		settlements[i] = settlement
		fmt.Printf("DEBUG: Successfully created settlement with ID: %d\n", settlement.ID)
	}

	// Load settlements with user details
	var result []models.Settlement
	fmt.Printf("DEBUG: Loading settlements with user details for eventID: %d\n", eventID)
	if err := database.DB.Where("event_id = ? AND status = ?", eventID, "pending").
		Preload("FromUser").
		Preload("ToUser").
		Find(&result).Error; err != nil {
		fmt.Printf("DEBUG: Error loading settlements: %v\n", err)
		return nil, fmt.Errorf("failed to load settlements: %v", err)
	}
	fmt.Printf("DEBUG: Loaded %d settlements with user details\n", len(result))

	return result, nil
}

// GetEventSettlements returns all settlements for an event
func (s *SettlementService) GetEventSettlements(eventID uint) ([]models.Settlement, error) {
	fmt.Printf("DEBUG: GetEventSettlements called for eventID: %d\n", eventID)

	var settlements []models.Settlement
	if err := database.DB.Where("event_id = ?", eventID).
		Preload("FromUser").
		Preload("ToUser").
		Order("created_at DESC").
		Find(&settlements).Error; err != nil {
		fmt.Printf("DEBUG: Error fetching settlements: %v\n", err)
		return nil, fmt.Errorf("failed to get settlements: %v", err)
	}

	fmt.Printf("DEBUG: Found %d settlements for eventID %d\n", len(settlements), eventID)
	for i, settlement := range settlements {
		fmt.Printf("DEBUG: Settlement %d: ID=%d From=%d To=%d Amount=%s Status=%s\n",
			i+1, settlement.ID, settlement.FromUserID, settlement.ToUserID, settlement.Amount, settlement.Status)
	}

	return settlements, nil
}

// MarkSettlementCompleted marks a settlement as completed
func (s *SettlementService) MarkSettlementCompleted(settlementID uint, paymentReference string) error {
	settlement := models.Settlement{}
	if err := database.DB.First(&settlement, settlementID).Error; err != nil {
		return fmt.Errorf("settlement not found: %v", err)
	}

	if settlement.Status != "pending" {
		return fmt.Errorf("settlement is not pending")
	}

	// Update settlement
	updates := map[string]interface{}{
		"status":            "completed",
		"payment_reference": paymentReference,
		"settled_at":        database.DB.NowFunc(),
	}

	if err := database.DB.Model(&settlement).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update settlement: %v", err)
	}

	return nil
}

// GetSettlementSummary returns a summary of settlements for an event
func (s *SettlementService) GetSettlementSummary(eventID uint) (map[string]interface{}, error) {
	balances, err := s.CalculateUserBalances(eventID)
	if err != nil {
		return nil, err
	}

	settlements, err := s.GetEventSettlements(eventID)
	if err != nil {
		return nil, err
	}

	// Calculate summary statistics
	totalOwed := decimal.Zero
	totalOwes := decimal.Zero
	usersInDebt := 0
	usersInCredit := 0

	for _, balance := range balances {
		if balance.NetBalance.IsNegative() {
			totalOwes = totalOwes.Add(balance.OwesAmount)
			usersInDebt++
		} else if balance.NetBalance.IsPositive() {
			totalOwed = totalOwed.Add(balance.OwedAmount)
			usersInCredit++
		}
	}

	pendingSettlements := 0
	completedSettlements := 0
	totalPendingAmount := decimal.Zero

	for _, settlement := range settlements {
		if settlement.Status == "pending" {
			pendingSettlements++
			totalPendingAmount = totalPendingAmount.Add(settlement.Amount)
		} else if settlement.Status == "completed" {
			completedSettlements++
		}
	}

	return map[string]interface{}{
		"total_owed":            totalOwed,
		"total_owes":            totalOwes,
		"users_in_debt":         usersInDebt,
		"users_in_credit":       usersInCredit,
		"pending_settlements":   pendingSettlements,
		"completed_settlements": completedSettlements,
		"total_pending_amount":  totalPendingAmount,
		"balances":              balances,
		"settlements":           settlements,
	}, nil
}

// ValidateSettlement validates if a settlement can be created
func (s *SettlementService) ValidateSettlement(eventID, fromUserID, toUserID uint, amount decimal.Decimal) error {
	// Check if users are participants
	var fromParticipation, toParticipation models.Participation

	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, fromUserID).
		First(&fromParticipation).Error; err != nil {
		return fmt.Errorf("from user is not a participant in this event")
	}

	if err := database.DB.Where("event_id = ? AND user_id = ?", eventID, toUserID).
		First(&toParticipation).Error; err != nil {
		return fmt.Errorf("to user is not a participant in this event")
	}

	// Check if amount is positive
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("settlement amount must be positive")
	}

	// Get user balances to validate settlement amount
	balances, err := s.CalculateUserBalances(eventID)
	if err != nil {
		return fmt.Errorf("failed to calculate balances: %v", err)
	}

	var fromBalance, toBalance *models.UserBalance
	for _, balance := range balances {
		if balance.UserID == fromUserID {
			fromBalance = &balance
		}
		if balance.UserID == toUserID {
			toBalance = &balance
		}
	}

	if fromBalance == nil || toBalance == nil {
		return fmt.Errorf("unable to find user balances")
	}

	// Check if fromUser actually owes money
	if !fromBalance.NetBalance.IsNegative() {
		return fmt.Errorf("from user does not owe money")
	}

	// Check if toUser is owed money
	if !toBalance.NetBalance.IsPositive() {
		return fmt.Errorf("to user is not owed money")
	}

	// Check if settlement amount doesn't exceed what is owed
	if amount.GreaterThan(fromBalance.OwesAmount) {
		return fmt.Errorf("settlement amount exceeds what from user owes")
	}

	if amount.GreaterThan(toBalance.OwedAmount) {
		return fmt.Errorf("settlement amount exceeds what to user is owed")
	}

	return nil
}

// CreateCustomSettlement creates a custom settlement between two users
func (s *SettlementService) CreateCustomSettlement(eventID, fromUserID, toUserID uint, amount decimal.Decimal, currency string) (*models.Settlement, error) {
	// Validate settlement
	if err := s.ValidateSettlement(eventID, fromUserID, toUserID, amount); err != nil {
		return nil, err
	}

	// Create settlement
	settlement := models.Settlement{
		EventID:    eventID,
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Amount:     amount,
		Currency:   currency,
		Status:     "pending",
	}

	if err := database.DB.Create(&settlement).Error; err != nil {
		return nil, fmt.Errorf("failed to create settlement: %v", err)
	}

	// Load settlement with user details
	database.DB.Preload("FromUser").Preload("ToUser").First(&settlement, settlement.ID)

	return &settlement, nil
}

// GetUserSettlements returns settlements involving a specific user
func (s *SettlementService) GetUserSettlements(eventID, userID uint) ([]models.Settlement, error) {
	var settlements []models.Settlement
	if err := database.DB.Where("event_id = ? AND (from_user_id = ? OR to_user_id = ?)",
		eventID, userID, userID).
		Preload("FromUser").
		Preload("ToUser").
		Order("created_at DESC").
		Find(&settlements).Error; err != nil {
		return nil, fmt.Errorf("failed to get user settlements: %v", err)
	}

	return settlements, nil
}
