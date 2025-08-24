package middleware

import (
	"expense-tracker/models"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger returns a logging middleware
func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

// AuditLogger logs important actions to the audit trail
func AuditLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip audit logging for GET requests and health checks
		if c.Request.Method == "GET" || c.FullPath() == "/health" {
			c.Next()
			return
		}

		// Process the request
		c.Next()

		// Log after request completion
		go func() {
			logAuditEvent(c)
		}()
	}
}

// logAuditEvent creates an audit log entry
func logAuditEvent(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		return // Don't log unauthenticated requests
	}

	// Determine the action based on method and path
	action := determineAction(c.Request.Method, c.FullPath())
	if action == "" {
		return // Skip logging for non-important actions
	}

	auditLog := &models.AuditLog{
		TableName: extractTableName(c.FullPath()),
		RecordID:  extractRecordID(c),
		Action:    action,
		ChangedBy: userID.(uint),
		ChangedAt: time.Now(),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		SessionID: c.GetHeader("X-Session-ID"), // If you implement session tracking
	}

	// Note: In a real application, you would save this to the database
	// For now, we'll just print it for demonstration
	fmt.Printf("Audit Log: %+v\n", auditLog)
}

// determineAction maps HTTP methods and paths to audit actions
func determineAction(method, path string) string {
	switch method {
	case "POST":
		if contains(path, "register") || contains(path, "login") {
			return "AUTH"
		}
		return "CREATE"
	case "PUT", "PATCH":
		return "UPDATE"
	case "DELETE":
		return "DELETE"
	default:
		return ""
	}
}

// extractTableName extracts the table name from the path
func extractTableName(path string) string {
	if contains(path, "users") {
		return "users"
	}
	if contains(path, "events") {
		return "events"
	}
	if contains(path, "expenses") {
		return "expenses"
	}
	if contains(path, "contributions") {
		return "contributions"
	}
	return "unknown"
}

// extractRecordID extracts the record ID from the context
func extractRecordID(c *gin.Context) uint {
	// Try to get ID from URL params
	if id := c.Param("id"); id != "" {
		// Convert string to uint (simplified)
		return 0 // You would implement proper conversion here
	}
	return 0
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsInner(s, substr))))
}

func containsInner(s, substr string) bool {
	for i := 1; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
