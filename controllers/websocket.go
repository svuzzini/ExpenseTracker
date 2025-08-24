package controllers

import (
	"encoding/json"
	"expense-tracker/database"
	"expense-tracker/middleware"
	"expense-tracker/models"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketController handles WebSocket connections for real-time updates
type WebSocketController struct {
	// Map of event ID to map of user ID to WebSocket connections
	connections map[uint]map[uint]*websocket.Conn
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	EventID   uint        `json:"event_id,omitempty"`
	UserID    uint        `json:"user_id"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// MessageType represents different types of WebSocket messages
type MessageType string

const (
	MessageTypeExpenseAdded        MessageType = "expense_added"
	MessageTypeExpenseApproved     MessageType = "expense_approved"
	MessageTypeExpenseRejected     MessageType = "expense_rejected"
	MessageTypeContributionAdded   MessageType = "contribution_added"
	MessageTypeUserJoined          MessageType = "user_joined"
	MessageTypeBalanceUpdated      MessageType = "balance_updated"
	MessageTypeCommentAdded        MessageType = "comment_added"
	MessageTypeSettlementCreated   MessageType = "settlement_created"
	MessageTypeSettlementCompleted MessageType = "settlement_completed"
)

// NewWebSocketController creates a new WebSocket controller
func NewWebSocketController() *WebSocketController {
	return &WebSocketController{
		connections: make(map[uint]map[uint]*websocket.Conn),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// HandleWebSocket upgrades HTTP connection to WebSocket and handles messages
func (wsc *WebSocketController) HandleWebSocket(c *gin.Context) {
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

	// Check if user is a participant in the event
	var participation models.Participation
	if err := database.DB.Where("user_id = ? AND event_id = ?", user.ID, uint(eventID)).
		First(&participation).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You are not a participant in this event",
			"code":  "NOT_PARTICIPANT",
		})
		return
	}

	// Upgrade connection to WebSocket
	conn, err := wsc.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Add connection to the map
	wsc.addConnection(uint(eventID), user.ID, conn)

	// Send welcome message
	welcomeMsg := WebSocketMessage{
		Type:      "connected",
		EventID:   uint(eventID),
		UserID:    user.ID,
		Data:      map[string]string{"message": "Connected successfully"},
		Timestamp: time.Now(),
	}
	wsc.sendToConnection(conn, welcomeMsg)

	// Notify other participants that user joined
	joinMsg := WebSocketMessage{
		Type:    string(MessageTypeUserJoined),
		EventID: uint(eventID),
		UserID:  user.ID,
		Data: map[string]interface{}{
			"user":    user,
			"message": user.DisplayName + " is now online",
		},
		Timestamp: time.Now(),
	}
	wsc.BroadcastToEvent(uint(eventID), joinMsg, user.ID)

	// Handle connection cleanup when function returns
	defer func() {
		wsc.removeConnection(uint(eventID), user.ID)
		conn.Close()
	}()

	// Set up ping/pong to keep connection alive
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	// Handle messages in goroutine
	go func() {
		for {
			select {
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Read messages from client
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process incoming message
		wsc.handleIncomingMessage(uint(eventID), user.ID, message)
	}
}

// addConnection adds a WebSocket connection to the map
func (wsc *WebSocketController) addConnection(eventID, userID uint, conn *websocket.Conn) {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if wsc.connections[eventID] == nil {
		wsc.connections[eventID] = make(map[uint]*websocket.Conn)
	}

	wsc.connections[eventID][userID] = conn
	log.Printf("Added WebSocket connection for user %d in event %d", userID, eventID)
}

// removeConnection removes a WebSocket connection from the map
func (wsc *WebSocketController) removeConnection(eventID, userID uint) {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if eventConnections, exists := wsc.connections[eventID]; exists {
		delete(eventConnections, userID)
		if len(eventConnections) == 0 {
			delete(wsc.connections, eventID)
		}
	}

	log.Printf("Removed WebSocket connection for user %d in event %d", userID, eventID)
}

// sendToConnection sends a message to a specific connection
func (wsc *WebSocketController) sendToConnection(conn *websocket.Conn, message WebSocketMessage) error {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(message)
}

// BroadcastToEvent broadcasts a message to all connections in an event
func (wsc *WebSocketController) BroadcastToEvent(eventID uint, message WebSocketMessage, excludeUserID ...uint) {
	wsc.mu.RLock()
	eventConnections := wsc.connections[eventID]
	wsc.mu.RUnlock()

	if eventConnections == nil {
		return
	}

	excludeMap := make(map[uint]bool)
	for _, userID := range excludeUserID {
		excludeMap[userID] = true
	}

	for userID, conn := range eventConnections {
		if excludeMap[userID] {
			continue
		}

		if err := wsc.sendToConnection(conn, message); err != nil {
			log.Printf("Failed to send message to user %d: %v", userID, err)
			// Remove failed connection
			wsc.removeConnection(eventID, userID)
		}
	}
}

// BroadcastToUser sends a message to a specific user
func (wsc *WebSocketController) BroadcastToUser(eventID, userID uint, message WebSocketMessage) {
	wsc.mu.RLock()
	eventConnections := wsc.connections[eventID]
	wsc.mu.RUnlock()

	if eventConnections == nil {
		return
	}

	if conn, exists := eventConnections[userID]; exists {
		if err := wsc.sendToConnection(conn, message); err != nil {
			log.Printf("Failed to send message to user %d: %v", userID, err)
			wsc.removeConnection(eventID, userID)
		}
	}
}

// handleIncomingMessage processes incoming WebSocket messages
func (wsc *WebSocketController) handleIncomingMessage(eventID, userID uint, messageData []byte) {
	var message WebSocketMessage
	if err := json.Unmarshal(messageData, &message); err != nil {
		log.Printf("Failed to unmarshal WebSocket message: %v", err)
		return
	}

	// Set metadata
	message.EventID = eventID
	message.UserID = userID
	message.Timestamp = time.Now()

	// Process different message types
	switch message.Type {
	case "ping":
		// Respond with pong
		pongMsg := WebSocketMessage{
			Type:      "pong",
			EventID:   eventID,
			UserID:    userID,
			Timestamp: time.Now(),
		}
		if conn, exists := wsc.connections[eventID][userID]; exists {
			wsc.sendToConnection(conn, pongMsg)
		}

	case "typing":
		// Broadcast typing indicator to other users
		wsc.BroadcastToEvent(eventID, message, userID)

	case "comment":
		// Handle comment messages (if you implement comments)
		wsc.BroadcastToEvent(eventID, message, userID)

	default:
		log.Printf("Unknown message type: %s", message.Type)
	}
}

// NotifyExpenseAdded notifies all event participants about a new expense
func (wsc *WebSocketController) NotifyExpenseAdded(eventID uint, expense models.Expense) {
	message := WebSocketMessage{
		Type:    string(MessageTypeExpenseAdded),
		EventID: eventID,
		UserID:  expense.SubmittedBy,
		Data: map[string]interface{}{
			"expense": expense,
			"message": expense.Submitter.DisplayName + " added a new expense: " + expense.Description,
		},
		Timestamp: time.Now(),
	}

	wsc.BroadcastToEvent(eventID, message)
}

// NotifyExpenseApproved notifies all event participants about an approved expense
func (wsc *WebSocketController) NotifyExpenseApproved(eventID uint, expense models.Expense, reviewerID uint) {
	message := WebSocketMessage{
		Type:    string(MessageTypeExpenseApproved),
		EventID: eventID,
		UserID:  reviewerID,
		Data: map[string]interface{}{
			"expense": expense,
			"message": "Expense approved: " + expense.Description,
		},
		Timestamp: time.Now(),
	}

	wsc.BroadcastToEvent(eventID, message)
}

// NotifyExpenseRejected notifies all event participants about a rejected expense
func (wsc *WebSocketController) NotifyExpenseRejected(eventID uint, expense models.Expense, reviewerID uint) {
	message := WebSocketMessage{
		Type:    string(MessageTypeExpenseRejected),
		EventID: eventID,
		UserID:  reviewerID,
		Data: map[string]interface{}{
			"expense": expense,
			"message": "Expense rejected: " + expense.Description,
			"reason":  expense.RejectionReason,
		},
		Timestamp: time.Now(),
	}

	wsc.BroadcastToEvent(eventID, message)
}

// NotifyContributionAdded notifies all event participants about a new contribution
func (wsc *WebSocketController) NotifyContributionAdded(eventID uint, contribution models.Contribution) {
	message := WebSocketMessage{
		Type:    string(MessageTypeContributionAdded),
		EventID: eventID,
		UserID:  contribution.UserID,
		Data: map[string]interface{}{
			"contribution": contribution,
			"message":      contribution.User.DisplayName + " added a contribution of " + contribution.Amount.String(),
		},
		Timestamp: time.Now(),
	}

	wsc.BroadcastToEvent(eventID, message)
}

// NotifyBalanceUpdated notifies about balance changes
func (wsc *WebSocketController) NotifyBalanceUpdated(eventID uint, balances []models.UserBalance) {
	message := WebSocketMessage{
		Type:    string(MessageTypeBalanceUpdated),
		EventID: eventID,
		Data: map[string]interface{}{
			"balances": balances,
			"message":  "Balances updated",
		},
		Timestamp: time.Now(),
	}

	wsc.BroadcastToEvent(eventID, message)
}

// NotifySettlementCreated notifies about new settlements
func (wsc *WebSocketController) NotifySettlementCreated(eventID uint, settlement models.Settlement) {
	message := WebSocketMessage{
		Type:    string(MessageTypeSettlementCreated),
		EventID: eventID,
		Data: map[string]interface{}{
			"settlement": settlement,
			"message":    "New settlement created",
		},
		Timestamp: time.Now(),
	}

	wsc.BroadcastToEvent(eventID, message)
}

// NotifySettlementCompleted notifies about completed settlements
func (wsc *WebSocketController) NotifySettlementCompleted(eventID uint, settlement models.Settlement) {
	message := WebSocketMessage{
		Type:    string(MessageTypeSettlementCompleted),
		EventID: eventID,
		Data: map[string]interface{}{
			"settlement": settlement,
			"message":    "Settlement completed",
		},
		Timestamp: time.Now(),
	}

	wsc.BroadcastToEvent(eventID, message)
}

// GetActiveConnections returns the number of active connections per event
func (wsc *WebSocketController) GetActiveConnections() map[uint]int {
	wsc.mu.RLock()
	defer wsc.mu.RUnlock()

	result := make(map[uint]int)
	for eventID, eventConnections := range wsc.connections {
		result[eventID] = len(eventConnections)
	}

	return result
}

// GetEventParticipants returns active participants for an event
func (wsc *WebSocketController) GetEventParticipants(eventID uint) []uint {
	wsc.mu.RLock()
	defer wsc.mu.RUnlock()

	var participants []uint
	if eventConnections, exists := wsc.connections[eventID]; exists {
		for userID := range eventConnections {
			participants = append(participants, userID)
		}
	}

	return participants
}
