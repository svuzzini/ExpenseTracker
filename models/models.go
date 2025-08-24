package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"unique;not null;size:50"`
	Email     string    `json:"email" gorm:"unique;not null;size:100"`
	Password  string    `json:"-" gorm:"not null;size:255"` // Hidden from JSON
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// User preferences
	Currency      string `json:"currency" gorm:"default:'USD';size:3"`
	Notifications bool   `json:"notifications" gorm:"default:true"`
	Theme         string `json:"theme" gorm:"default:'light';size:10"`
	Language      string `json:"language" gorm:"default:'en';size:5"`
	Timezone      string `json:"timezone" gorm:"default:'UTC';size:50"`

	// Profile information
	FirstName   string `json:"first_name" gorm:"size:50"`
	LastName    string `json:"last_name" gorm:"size:50"`
	DisplayName string `json:"display_name" gorm:"size:100"`
	Avatar      string `json:"avatar" gorm:"size:255"`

	// Relationships
	CreatedEvents  []Event         `json:"created_events,omitempty" gorm:"foreignKey:CreatedBy"`
	Participations []Participation `json:"participations,omitempty"`
	Contributions  []Contribution  `json:"contributions,omitempty"`
	Expenses       []Expense       `json:"expenses,omitempty" gorm:"foreignKey:SubmittedBy"`
}

// Event represents a group expense event
type Event struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null;size:100"`
	Description string    `json:"description" gorm:"size:500"`
	Code        string    `json:"code" gorm:"unique;not null;size:8"`
	CreatedBy   uint      `json:"created_by" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Event settings
	Currency           string          `json:"currency" gorm:"default:'USD';size:3"`
	Status             string          `json:"status" gorm:"default:'active';size:20"` // active, completed, archived
	TotalContributions decimal.Decimal `json:"total_contributions" gorm:"type:decimal(12,2);default:0"`
	TotalExpenses      decimal.Decimal `json:"total_expenses" gorm:"type:decimal(12,2);default:0"`
	EndDate            *time.Time      `json:"end_date,omitempty"`

	// Settings
	RequireApproval   bool            `json:"require_approval" gorm:"default:true"`
	AutoApprovalLimit decimal.Decimal `json:"auto_approval_limit" gorm:"type:decimal(10,2);default:0"`

	// Relationships
	Creator       User            `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Participants  []Participation `json:"participants,omitempty"`
	Contributions []Contribution  `json:"contributions,omitempty"`
	Expenses      []Expense       `json:"expenses,omitempty"`
	Settlements   []Settlement    `json:"settlements,omitempty"`
}

// Participation represents user participation in an event
type Participation struct {
	ID       uint      `json:"id" gorm:"primaryKey"`
	UserID   uint      `json:"user_id" gorm:"not null"`
	EventID  uint      `json:"event_id" gorm:"not null"`
	Role     string    `json:"role" gorm:"default:'participant';size:20"` // owner, admin, moderator, participant, viewer
	JoinedAt time.Time `json:"joined_at"`

	// Relationships
	User  User  `json:"user,omitempty"`
	Event Event `json:"event,omitempty"`
}

// Contribution represents money contributed by a user to an event
type Contribution struct {
	ID        uint            `json:"id" gorm:"primaryKey"`
	EventID   uint            `json:"event_id" gorm:"not null"`
	UserID    uint            `json:"user_id" gorm:"not null"`
	Amount    decimal.Decimal `json:"amount" gorm:"type:decimal(10,2);not null"`
	Currency  string          `json:"currency" gorm:"size:3;not null"`
	Notes     string          `json:"notes" gorm:"size:500"`
	Timestamp time.Time       `json:"timestamp"`

	// Relationships
	Event Event `json:"event,omitempty"`
	User  User  `json:"user,omitempty"`
}

// ExpenseCategory represents expense categories
type ExpenseCategory struct {
	ID   uint   `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"unique;not null;size:50"`
	Icon string `json:"icon" gorm:"size:10"`

	// Relationships
	Expenses []Expense `json:"expenses,omitempty" gorm:"foreignKey:CategoryID"`
}

// Expense represents an expense submitted by a user
type Expense struct {
	ID          uint            `json:"id" gorm:"primaryKey"`
	EventID     uint            `json:"event_id" gorm:"not null"`
	SubmittedBy uint            `json:"submitted_by" gorm:"not null"`
	CategoryID  uint            `json:"category_id" gorm:"not null"`
	Amount      decimal.Decimal `json:"amount" gorm:"type:decimal(10,2);not null"`
	Currency    string          `json:"currency" gorm:"size:3;not null"`
	Description string          `json:"description" gorm:"not null;size:255"`
	Date        time.Time       `json:"date" gorm:"not null"`
	Status      string          `json:"status" gorm:"default:'pending';size:20"` // pending, approved, rejected

	// Review information
	ReviewedBy      *uint      `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty" gorm:"size:500"`

	// Optional fields
	ReceiptURL  string    `json:"receipt_url,omitempty" gorm:"size:500"`
	Location    string    `json:"location,omitempty" gorm:"size:200"`
	Vendor      string    `json:"vendor,omitempty" gorm:"size:100"`
	Notes       string    `json:"notes,omitempty" gorm:"size:500"`
	SubmittedAt time.Time `json:"submitted_at"`

	// Splitting information
	SplitType string `json:"split_type" gorm:"default:'equal';size:20"` // equal, percentage, custom, weighted

	// Relationships
	Event     Event           `json:"event,omitempty"`
	Submitter User            `json:"submitter,omitempty" gorm:"foreignKey:SubmittedBy"`
	Reviewer  *User           `json:"reviewer,omitempty" gorm:"foreignKey:ReviewedBy"`
	Category  ExpenseCategory `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Shares    []ExpenseShare  `json:"shares,omitempty"`
}

// ExpenseShare represents how an expense is split among participants
type ExpenseShare struct {
	ID         uint            `json:"id" gorm:"primaryKey"`
	ExpenseID  uint            `json:"expense_id" gorm:"not null"`
	UserID     uint            `json:"user_id" gorm:"not null"`
	Amount     decimal.Decimal `json:"amount" gorm:"type:decimal(10,2);not null"`
	Percentage decimal.Decimal `json:"percentage" gorm:"type:decimal(5,2)"` // For percentage splits

	// Relationships
	Expense Expense `json:"expense,omitempty"`
	User    User    `json:"user,omitempty"`
}

// Settlement represents settlement transactions between users
type Settlement struct {
	ID         uint            `json:"id" gorm:"primaryKey"`
	EventID    uint            `json:"event_id" gorm:"not null"`
	FromUserID uint            `json:"from_user_id" gorm:"not null"`
	ToUserID   uint            `json:"to_user_id" gorm:"not null"`
	Amount     decimal.Decimal `json:"amount" gorm:"type:decimal(10,2);not null"`
	Currency   string          `json:"currency" gorm:"size:3;not null"`
	Status     string          `json:"status" gorm:"default:'pending';size:20"` // pending, completed, cancelled
	Method     string          `json:"method,omitempty" gorm:"size:50"`         // cash, bank_transfer, venmo, paypal, etc.

	// Tracking information
	PaymentReference string     `json:"payment_reference,omitempty" gorm:"size:100"`
	SettledAt        *time.Time `json:"settled_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`

	// Relationships
	Event    Event `json:"event,omitempty"`
	FromUser User  `json:"from_user,omitempty" gorm:"foreignKey:FromUserID"`
	ToUser   User  `json:"to_user,omitempty" gorm:"foreignKey:ToUserID"`
}

// UserBalance represents calculated balance for a user in an event
type UserBalance struct {
	UserID      uint            `json:"user_id"`
	Username    string          `json:"username"`
	DisplayName string          `json:"display_name"`
	Contributed decimal.Decimal `json:"contributed"`
	Spent       decimal.Decimal `json:"spent"`
	NetBalance  decimal.Decimal `json:"net_balance"`
	OwesAmount  decimal.Decimal `json:"owes_amount"`
	OwedAmount  decimal.Decimal `json:"owed_amount"`
}

// AuditLog represents audit trail for all changes
type AuditLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TableName string    `json:"table_name" gorm:"not null;size:50"`
	RecordID  uint      `json:"record_id" gorm:"not null"`
	Action    string    `json:"action" gorm:"not null;size:20"` // INSERT, UPDATE, DELETE
	OldValues string    `json:"old_values,omitempty" gorm:"type:text"`
	NewValues string    `json:"new_values,omitempty" gorm:"type:text"`
	ChangedBy uint      `json:"changed_by" gorm:"not null"`
	ChangedAt time.Time `json:"changed_at"`
	IPAddress string    `json:"ip_address" gorm:"size:45"`
	UserAgent string    `json:"user_agent" gorm:"size:500"`
	SessionID string    `json:"session_id" gorm:"size:255"`

	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:ChangedBy"`
}

// GetDefaultCategories returns the default expense categories
func GetDefaultCategories() []ExpenseCategory {
	return []ExpenseCategory{
		{Name: "Food & Dining", Icon: "🍽️"},
		{Name: "Transportation", Icon: "🚗"},
		{Name: "Accommodation", Icon: "🏨"},
		{Name: "Entertainment", Icon: "🎬"},
		{Name: "Shopping", Icon: "🛍️"},
		{Name: "Groceries", Icon: "🛒"},
		{Name: "Utilities", Icon: "💡"},
		{Name: "Health & Medical", Icon: "🏥"},
		{Name: "Education", Icon: "📚"},
		{Name: "Other", Icon: "📦"},
	}
}

// BeforeCreate hook for User
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.DisplayName == "" {
		u.DisplayName = u.Username
	}
	return nil
}

// BeforeCreate hook for Expense
func (e *Expense) BeforeCreate(tx *gorm.DB) error {
	if e.SubmittedAt.IsZero() {
		e.SubmittedAt = time.Now()
	}
	return nil
}

// BeforeCreate hook for Contribution
func (c *Contribution) BeforeCreate(tx *gorm.DB) error {
	if c.Timestamp.IsZero() {
		c.Timestamp = time.Now()
	}
	return nil
}

// BeforeCreate hook for Participation
func (p *Participation) BeforeCreate(tx *gorm.DB) error {
	if p.JoinedAt.IsZero() {
		p.JoinedAt = time.Now()
	}
	return nil
}

// IsAdmin checks if a user is an admin in an event
func (p *Participation) IsAdmin() bool {
	return p.Role == "admin" || p.Role == "owner"
}

// CanApproveExpenses checks if a user can approve expenses
func (p *Participation) CanApproveExpenses() bool {
	return p.Role == "admin" || p.Role == "owner" || p.Role == "moderator"
}

// CanManageParticipants checks if a user can manage participants
func (p *Participation) CanManageParticipants() bool {
	return p.Role == "admin" || p.Role == "owner"
}
