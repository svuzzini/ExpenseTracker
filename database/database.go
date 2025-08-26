package database

import (
	"expense-tracker/models"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Config represents database configuration
type Config struct {
	Path            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	LogLevel        logger.LogLevel
}

// DefaultConfig returns default database configuration
func DefaultConfig() *Config {
	return &Config{
		Path:            "expense_tracker.db",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		LogLevel:        logger.Info,
	}
}

// Initialize initializes the database connection and runs migrations
func Initialize(config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	var err error

	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(config.LogLevel),
	}

	// Connect to SQLite database
	DB, err = gorm.Open(sqlite.Open(config.Path), gormConfig)
	if err != nil {
		return err
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	log.Println("Database connected successfully")

	// Optimize SQLite settings
	if err := optimizeSQLite(); err != nil {
		log.Printf("Warning: Failed to optimize SQLite settings: %v", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return err
	}

	// Seed default data
	if err := seedDefaultData(); err != nil {
		return err
	}

	log.Println("Database initialization completed")
	return nil
}

// optimizeSQLite applies SQLite-specific optimizations
func optimizeSQLite() error {
	// Enable WAL mode for better concurrency
	if err := DB.Exec("PRAGMA journal_mode = WAL;").Error; err != nil {
		return err
	}

	// Set synchronous mode to NORMAL for better performance
	if err := DB.Exec("PRAGMA synchronous = NORMAL;").Error; err != nil {
		return err
	}

	// Set cache size to 64MB
	if err := DB.Exec("PRAGMA cache_size = -64000;").Error; err != nil {
		return err
	}

	// Use memory for temporary tables
	if err := DB.Exec("PRAGMA temp_store = MEMORY;").Error; err != nil {
		return err
	}

	// Enable memory-mapped I/O (256MB)
	if err := DB.Exec("PRAGMA mmap_size = 268435456;").Error; err != nil {
		return err
	}

	// Set busy timeout to 30 seconds
	if err := DB.Exec("PRAGMA busy_timeout = 30000;").Error; err != nil {
		return err
	}

	// Enable foreign keys
	if err := DB.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		return err
	}

	log.Println("SQLite optimizations applied")
	return nil
}

// runMigrations runs database migrations
func runMigrations() error {
	log.Println("Running database migrations...")

	// Auto-migrate all models
	err := DB.AutoMigrate(
		&models.User{},
		&models.Event{},
		&models.Participation{},
		&models.ExpenseCategory{},
		&models.Contribution{},
		&models.Expense{},
		&models.ExpenseShare{},
		&models.Settlement{},
		&models.AuditLog{},
	)

	if err != nil {
		return err
	}

	// Note: Event sourcing tables would be migrated separately to avoid import cycles
	log.Println("Basic database migrations completed")

	// Create additional indexes for performance
	if err := createIndexes(); err != nil {
		log.Printf("Warning: Failed to create some indexes: %v", err)
	}

	log.Println("Database migrations completed")
	return nil
}

// Advanced event sourcing migration would be handled separately to avoid import cycles

// createIndexes creates additional database indexes for performance
func createIndexes() error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_events_code ON events(code);",
		"CREATE INDEX IF NOT EXISTS idx_events_created_by ON events(created_by);",
		"CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);",
		"CREATE INDEX IF NOT EXISTS idx_participations_user_event ON participations(user_id, event_id);",
		"CREATE INDEX IF NOT EXISTS idx_participations_role ON participations(role);",
		"CREATE INDEX IF NOT EXISTS idx_contributions_event_user ON contributions(event_id, user_id);",
		"CREATE INDEX IF NOT EXISTS idx_contributions_timestamp ON contributions(timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_expenses_event_status ON expenses(event_id, status);",
		"CREATE INDEX IF NOT EXISTS idx_expenses_submitted_by ON expenses(submitted_by);",
		"CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(date);",
		"CREATE INDEX IF NOT EXISTS idx_expense_shares_expense_user ON expense_shares(expense_id, user_id);",
		"CREATE INDEX IF NOT EXISTS idx_settlements_event ON settlements(event_id);",
		"CREATE INDEX IF NOT EXISTS idx_settlements_users ON settlements(from_user_id, to_user_id);",
		"CREATE INDEX IF NOT EXISTS idx_settlements_status ON settlements(status);",
		// Audit log indexes - commented out due to table creation issues
		// "CREATE INDEX IF NOT EXISTS idx_audit_log_table_record ON audit_log(table_name, record_id);",
		// "CREATE INDEX IF NOT EXISTS idx_audit_log_changed_by ON audit_log(changed_by);",
		// "CREATE INDEX IF NOT EXISTS idx_audit_log_changed_at ON audit_log(changed_at);",
	}

	for _, indexSQL := range indexes {
		if err := DB.Exec(indexSQL).Error; err != nil {
			log.Printf("Failed to create index: %s, error: %v", indexSQL, err)
		}
	}

	log.Println("Database indexes created")
	return nil
}

// seedDefaultData seeds the database with default data
func seedDefaultData() error {
	log.Println("Seeding default data...")

	// Check if categories already exist
	var categoryCount int64
	DB.Model(&models.ExpenseCategory{}).Count(&categoryCount)

	if categoryCount == 0 {
		categories := models.GetDefaultCategories()
		for _, category := range categories {
			if err := DB.Create(&category).Error; err != nil {
				return err
			}
		}
		log.Printf("Created %d default expense categories", len(categories))
	}

	// Check if demo user already exists
	var userCount int64
	DB.Model(&models.User{}).Where("email = ?", "demo@example.com").Count(&userCount)

	if userCount == 0 {
		// Create demo user
		demoUser := models.User{
			Username:    "demo",
			Email:       "demo@example.com",
			DisplayName: "Demo User",
		}

		// Hash password "Demo123!" (meets password requirements)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Demo123!"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		demoUser.Password = string(hashedPassword)

		if err := DB.Create(&demoUser).Error; err != nil {
			return err
		}

		log.Println("Created demo user: demo@example.com / Demo123!")
	}

	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// Backup creates a backup of the database
func Backup(backupPath string) error {
	log.Printf("Creating database backup at: %s", backupPath)

	// Use SQLite's backup API
	query := "VACUUM INTO ?"
	return DB.Exec(query, backupPath).Error
}

// HealthCheck performs a database health check
func HealthCheck() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}

// GetStats returns database statistics
func GetStats() map[string]interface{} {
	sqlDB, err := DB.DB()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	stats := sqlDB.Stats()

	return map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}
