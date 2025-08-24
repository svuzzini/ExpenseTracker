package middleware

import (
	"expense-tracker/database"
	"expense-tracker/models"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWT secret key - in production, this should be loaded from environment variables
var jwtSecret = []byte(getJWTSecret())

// Claims represents JWT token claims
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// getJWTSecret gets JWT secret from environment or returns default
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-super-secret-jwt-key-change-in-production"
	}
	return secret
}

// GenerateToken generates a JWT token for a user
func GenerateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "expense-tracker",
			Subject:   user.Email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken validates a JWT token and returns claims
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

// RefreshToken generates a new token for a user
func RefreshToken(userID uint) (string, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return "", err
	}

	return GenerateToken(&user)
}

// AuthMiddleware validates JWT token and sets user context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
				"code":  "AUTH_HEADER_MISSING",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization format. Expected: Bearer <token>",
				"code":  "INVALID_AUTH_FORMAT",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := ValidateToken(tokenString)
		if err != nil {
			var errorCode string
			var errorMessage string

			switch err {
			case jwt.ErrTokenExpired:
				errorCode = "TOKEN_EXPIRED"
				errorMessage = "Token has expired"
			case jwt.ErrTokenNotValidYet:
				errorCode = "TOKEN_NOT_VALID_YET"
				errorMessage = "Token is not valid yet"
			case jwt.ErrSignatureInvalid:
				errorCode = "INVALID_SIGNATURE"
				errorMessage = "Invalid token signature"
			default:
				errorCode = "INVALID_TOKEN"
				errorMessage = "Invalid token"
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": errorMessage,
				"code":  errorCode,
			})
			c.Abort()
			return
		}

		// Check if user still exists
		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not found",
				"code":  "USER_NOT_FOUND",
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("user", user)

		c.Next()
	}
}

// OptionalAuthMiddleware validates JWT token if present but doesn't require it
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.Next()
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("user", user)

		c.Next()
	}
}

// GetCurrentUser retrieves the current user from the context
func GetCurrentUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	u, ok := user.(models.User)
	return &u, ok
}

// GetCurrentUserID retrieves the current user ID from the context
func GetCurrentUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	id, ok := userID.(uint)
	return id, ok
}

// RequireEventParticipation middleware ensures user is a participant in the event
func RequireEventParticipation() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetCurrentUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
				"code":  "USER_NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		eventID := c.Param("eventId")
		if eventID == "" {
			eventID = c.Param("id")
		}

		var participation models.Participation
		err := database.DB.Where("user_id = ? AND event_id = ?", userID, eventID).
			First(&participation).Error

		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You are not a participant in this event",
				"code":  "NOT_EVENT_PARTICIPANT",
			})
			c.Abort()
			return
		}

		// Set participation info in context
		c.Set("participation", participation)
		c.Next()
	}
}

// RequireEventAdmin middleware ensures user is an admin in the event
func RequireEventAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetCurrentUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
				"code":  "USER_NOT_AUTHENTICATED",
			})
			c.Abort()
			return
		}

		eventID := c.Param("eventId")
		if eventID == "" {
			eventID = c.Param("id")
		}

		var participation models.Participation
		err := database.DB.Where("user_id = ? AND event_id = ?", userID, eventID).
			First(&participation).Error

		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You are not a participant in this event",
				"code":  "NOT_EVENT_PARTICIPANT",
			})
			c.Abort()
			return
		}

		if !participation.CanApproveExpenses() {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions. Admin role required.",
				"code":  "INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Set("participation", participation)
		c.Next()
	}
}

// GetParticipation retrieves participation info from context
func GetParticipation(c *gin.Context) (*models.Participation, bool) {
	participation, exists := c.Get("participation")
	if !exists {
		return nil, false
	}

	p, ok := participation.(models.Participation)
	return &p, ok
}

// IsEventAdmin checks if the current user is an admin in the specified event
func IsEventAdmin(c *gin.Context, eventID uint) bool {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		return false
	}

	var participation models.Participation
	err := database.DB.Where("user_id = ? AND event_id = ?", userID, eventID).
		First(&participation).Error

	return err == nil && participation.CanApproveExpenses()
}

// CanApproveExpenses checks if the current user can approve expenses in the specified event
func CanApproveExpenses(c *gin.Context, eventID uint) bool {
	return IsEventAdmin(c, eventID)
}
