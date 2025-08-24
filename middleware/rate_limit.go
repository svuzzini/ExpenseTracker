package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter represents a simple token bucket rate limiter
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     time.Duration
	capacity int
}

// Visitor represents a unique visitor with their rate limit info
type Visitor struct {
	tokens     int
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate time.Duration, capacity int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		capacity: capacity,
	}

	// Cleanup visitors every 10 minutes
	go rl.cleanupVisitors()

	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	visitor, exists := rl.visitors[identifier]
	if !exists {
		visitor = &Visitor{
			tokens:     rl.capacity,
			lastUpdate: time.Now(),
		}
		rl.visitors[identifier] = visitor
	}
	rl.mu.Unlock()

	visitor.mu.Lock()
	defer visitor.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(visitor.lastUpdate)
	tokensToAdd := int(elapsed / rl.rate)

	if tokensToAdd > 0 {
		visitor.tokens += tokensToAdd
		if visitor.tokens > rl.capacity {
			visitor.tokens = rl.capacity
		}
		visitor.lastUpdate = now
	}

	if visitor.tokens > 0 {
		visitor.tokens--
		return true
	}

	return false
}

// cleanupVisitors removes old visitors to prevent memory leaks
func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(10 * time.Minute)

		rl.mu.Lock()
		for identifier, visitor := range rl.visitors {
			visitor.mu.Lock()
			if time.Since(visitor.lastUpdate) > time.Hour {
				delete(rl.visitors, identifier)
			}
			visitor.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// Global rate limiters
var (
	// General API rate limiter: 100 requests per minute
	generalLimiter = NewRateLimiter(600*time.Millisecond, 100)

	// Auth rate limiter: 5 requests per minute for auth endpoints
	authLimiter = NewRateLimiter(12*time.Second, 5)

	// Upload rate limiter: 10 uploads per hour
	uploadLimiter = NewRateLimiter(6*time.Minute, 10)
)

// RateLimit returns a general rate limiting middleware
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := getClientIdentifier(c)

		if !generalLimiter.Allow(identifier) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
				"code":  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuthRateLimit returns a rate limiting middleware for authentication endpoints
func AuthRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := getClientIdentifier(c)

		if !authLimiter.Allow(identifier) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many authentication attempts. Please try again later.",
				"code":  "AUTH_RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// UploadRateLimit returns a rate limiting middleware for upload endpoints
func UploadRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := getClientIdentifier(c)

		if !uploadLimiter.Allow(identifier) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Upload rate limit exceeded. Please try again later.",
				"code":  "UPLOAD_RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getClientIdentifier returns a unique identifier for the client
func getClientIdentifier(c *gin.Context) string {
	// Try to get user ID first (for authenticated users)
	if userID, exists := c.Get("user_id"); exists {
		return "user_" + string(rune(userID.(uint)))
	}

	// Fall back to IP address
	return c.ClientIP()
}
