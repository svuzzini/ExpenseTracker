package eventsourcing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Demo implementation of Phases 4-6 without complex dependencies
// This shows the architectural patterns and can be integrated with the existing system

// Phase 4: Saga Pattern Demo
type ExpenseProcessingSaga struct {
	ID          string    `json:"id"`
	EventID     uint      `json:"event_id"`
	ExpenseID   uint      `json:"expense_id"`
	Status      string    `json:"status"` // pending, processing, completed, failed
	CurrentStep int       `json:"current_step"`
	TotalSteps  int       `json:"total_steps"`
	Data        string    `json:"data"` // JSON serialized workflow data
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SagaManager struct {
	mu    sync.RWMutex
	sagas map[string]*ExpenseProcessingSaga
}

func NewSagaManager() *SagaManager {
	return &SagaManager{
		sagas: make(map[string]*ExpenseProcessingSaga),
	}
}

func (sm *SagaManager) StartExpenseProcessing(ctx context.Context, eventID, expenseID uint, amount float64) (*ExpenseProcessingSaga, error) {
	sagaID := fmt.Sprintf("expense_%d_%d_%d", eventID, expenseID, time.Now().Unix())
	
	saga := &ExpenseProcessingSaga{
		ID:          sagaID,
		EventID:     eventID,
		ExpenseID:   expenseID,
		Status:      "pending",
		CurrentStep: 0,
		TotalSteps:  5,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	workflowData := map[string]interface{}{
		"amount":          amount,
		"approval_required": amount > 100.0,
		"participants":    []uint{},
		"notifications":   []string{},
	}
	
	dataBytes, _ := json.Marshal(workflowData)
	saga.Data = string(dataBytes)

	sm.mu.Lock()
	sm.sagas[sagaID] = saga
	sm.mu.Unlock()

	log.Printf("🔄 Started expense processing saga: %s (Amount: $%.2f)", sagaID, amount)
	
	// Start the workflow asynchronously
	go sm.processWorkflow(ctx, saga)
	
	return saga, nil
}

func (sm *SagaManager) processWorkflow(ctx context.Context, saga *ExpenseProcessingSaga) {
	for saga.CurrentStep < saga.TotalSteps {
		switch saga.CurrentStep {
		case 0:
			sm.validateExpense(saga)
		case 1:
			sm.requestApprovals(saga)
		case 2:
			sm.calculateSplits(saga)
		case 3:
			sm.notifyParticipants(saga)
		case 4:
			sm.updateBalances(saga)
		}
		
		saga.CurrentStep++
		saga.UpdatedAt = time.Now()
		
		// Simulate processing time
		time.Sleep(100 * time.Millisecond)
	}
	
	saga.Status = "completed"
	log.Printf("✅ Completed expense processing saga: %s", saga.ID)
}

func (sm *SagaManager) validateExpense(saga *ExpenseProcessingSaga) {
	log.Printf("Step 1: Validating expense for saga %s", saga.ID)
}

func (sm *SagaManager) requestApprovals(saga *ExpenseProcessingSaga) {
	log.Printf("Step 2: Requesting approvals for saga %s", saga.ID)
}

func (sm *SagaManager) calculateSplits(saga *ExpenseProcessingSaga) {
	log.Printf("Step 3: Calculating splits for saga %s", saga.ID)
}

func (sm *SagaManager) notifyParticipants(saga *ExpenseProcessingSaga) {
	log.Printf("Step 4: Notifying participants for saga %s", saga.ID)
}

func (sm *SagaManager) updateBalances(saga *ExpenseProcessingSaga) {
	log.Printf("Step 5: Updating balances for saga %s", saga.ID)
}

func (sm *SagaManager) GetSaga(id string) (*ExpenseProcessingSaga, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	saga, exists := sm.sagas[id]
	return saga, exists
}

// Phase 5: CQRS Query Pattern Demo
type QueryProcessor struct {
	mu      sync.RWMutex
	queries map[string]interface{}
}

type UserEventsQuery struct {
	UserID uint   `json:"user_id"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type EventDetailsQuery struct {
	EventID uint `json:"event_id"`
	UserID  uint `json:"user_id"`
}

type QueryResult struct {
	Data      interface{} `json:"data"`
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
}

func NewQueryProcessor() *QueryProcessor {
	return &QueryProcessor{
		queries: make(map[string]interface{}),
	}
}

func (qp *QueryProcessor) ExecuteUserEventsQuery(ctx context.Context, query UserEventsQuery) (*QueryResult, error) {
	log.Printf("🔍 Executing UserEventsQuery for user %d", query.UserID)
	
	// Simulate database query
	mockEvents := []map[string]interface{}{
		{
			"id":              1,
			"name":            "Weekend Trip",
			"status":          "active",
			"total_expenses":  450.75,
			"user_balance":    -25.50,
			"participants":    4,
		},
		{
			"id":              2,
			"name":            "Team Lunch",
			"status":          "completed",
			"total_expenses":  280.00,
			"user_balance":    0.00,
			"participants":    6,
		},
	}

	// Apply filters
	if query.Status != "" {
		filtered := []map[string]interface{}{}
		for _, event := range mockEvents {
			if event["status"] == query.Status {
				filtered = append(filtered, event)
			}
		}
		mockEvents = filtered
	}

	// Apply limit
	if query.Limit > 0 && len(mockEvents) > query.Limit {
		mockEvents = mockEvents[:query.Limit]
	}

	return &QueryResult{
		Data:      mockEvents,
		Success:   true,
		Message:   fmt.Sprintf("Found %d events for user %d", len(mockEvents), query.UserID),
		Timestamp: time.Now(),
	}, nil
}

func (qp *QueryProcessor) ExecuteEventDetailsQuery(ctx context.Context, query EventDetailsQuery) (*QueryResult, error) {
	log.Printf("🔍 Executing EventDetailsQuery for event %d", query.EventID)
	
	// Simulate complex query with joins
	eventDetails := map[string]interface{}{
		"id":          query.EventID,
		"name":        "Weekend Trip",
		"description": "Fun mountain getaway",
		"status":      "active",
		"participants": []map[string]interface{}{
			{"id": 1, "name": "John Doe", "balance": -25.50},
			{"id": 2, "name": "Jane Smith", "balance": 15.25},
			{"id": 3, "name": "Bob Johnson", "balance": 10.25},
		},
		"summary": map[string]interface{}{
			"total_expenses":      450.75,
			"total_contributions": 400.00,
			"net_balance":        -50.75,
			"expense_count":       8,
		},
	}

	return &QueryResult{
		Data:      eventDetails,
		Success:   true,
		Message:   fmt.Sprintf("Retrieved details for event %d", query.EventID),
		Timestamp: time.Now(),
	}, nil
}

// Phase 6: Performance Monitoring Demo
type PerformanceMetrics struct {
	Timestamp         time.Time `json:"timestamp"`
	EventsPerSecond   float64   `json:"events_per_second"`
	AverageLatencyMS  float64   `json:"average_latency_ms"`
	MemoryUsageMB     float64   `json:"memory_usage_mb"`
	ActiveConnections int       `json:"active_connections"`
	CacheHitRatio     float64   `json:"cache_hit_ratio"`
	ErrorRate         float64   `json:"error_rate"`
}

type PerformanceMonitor struct {
	mu          sync.RWMutex
	metrics     *PerformanceMetrics
	alerts      []string
	isRunning   bool
	stopChannel chan bool
}

func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics:     &PerformanceMetrics{},
		alerts:      []string{},
		stopChannel: make(chan bool),
	}
}

func (pm *PerformanceMonitor) Start(ctx context.Context) {
	pm.mu.Lock()
	pm.isRunning = true
	pm.mu.Unlock()
	
	log.Println("📊 Starting performance monitoring...")
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			pm.collectMetrics()
			pm.checkAlerts()
		case <-pm.stopChannel:
			log.Println("📊 Performance monitoring stopped")
			return
		case <-ctx.Done():
			return
		}
	}
}

func (pm *PerformanceMonitor) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.isRunning {
		pm.isRunning = false
		pm.stopChannel <- true
	}
}

func (pm *PerformanceMonitor) collectMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Simulate metric collection
	pm.metrics = &PerformanceMetrics{
		Timestamp:         time.Now(),
		EventsPerSecond:   float64(50 + (time.Now().Unix() % 50)),
		AverageLatencyMS:  float64(100 + (time.Now().Unix() % 200)),
		MemoryUsageMB:     float64(256 + (time.Now().Unix() % 128)),
		ActiveConnections: int(10 + (time.Now().Unix() % 20)),
		CacheHitRatio:     0.85 + float64(time.Now().Unix()%15)/100,
		ErrorRate:         float64(time.Now().Unix()%5) / 100,
	}
	
	log.Printf("📊 Metrics collected: EPS=%.1f, Latency=%.1fms, Memory=%.1fMB", 
		pm.metrics.EventsPerSecond, pm.metrics.AverageLatencyMS, pm.metrics.MemoryUsageMB)
}

func (pm *PerformanceMonitor) checkAlerts() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Check for alert conditions
	if pm.metrics.MemoryUsageMB > 500 {
		alert := fmt.Sprintf("🚨 HIGH MEMORY USAGE: %.1f MB", pm.metrics.MemoryUsageMB)
		pm.alerts = append(pm.alerts, alert)
		log.Println(alert)
	}
	
	if pm.metrics.AverageLatencyMS > 500 {
		alert := fmt.Sprintf("🚨 HIGH LATENCY: %.1f ms", pm.metrics.AverageLatencyMS)
		pm.alerts = append(pm.alerts, alert)
		log.Println(alert)
	}
	
	if pm.metrics.ErrorRate > 0.05 {
		alert := fmt.Sprintf("🚨 HIGH ERROR RATE: %.2f%%", pm.metrics.ErrorRate*100)
		pm.alerts = append(pm.alerts, alert)
		log.Println(alert)
	}
}

func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	// Return a copy
	metricsCopy := *pm.metrics
	return &metricsCopy
}

func (pm *PerformanceMonitor) GetAlerts() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	return append([]string{}, pm.alerts...)
}

// Integration Demo
type AdvancedEventSourcingDemo struct {
	sagaManager      *SagaManager
	queryProcessor   *QueryProcessor
	perfMonitor      *PerformanceMonitor
}

func NewAdvancedEventSourcingDemo() *AdvancedEventSourcingDemo {
	return &AdvancedEventSourcingDemo{
		sagaManager:    NewSagaManager(),
		queryProcessor: NewQueryProcessor(),
		perfMonitor:    NewPerformanceMonitor(),
	}
}

func (demo *AdvancedEventSourcingDemo) Start(ctx context.Context) {
	log.Println("🚀 Starting Advanced Event Sourcing Demo (Phases 4-6)")
	
	// Start performance monitoring
	go demo.perfMonitor.Start(ctx)
	
	// Demo Phase 4: Saga Workflows
	log.Println("\n🔄 === PHASE 4: SAGA WORKFLOWS ===")
	saga1, _ := demo.sagaManager.StartExpenseProcessing(ctx, 1, 101, 75.50)
	saga2, _ := demo.sagaManager.StartExpenseProcessing(ctx, 1, 102, 150.00) // Requires approval
	
	time.Sleep(1 * time.Second)
	
	// Demo Phase 5: CQRS Queries
	log.Println("\n🔍 === PHASE 5: CQRS QUERIES ===")
	
	userQuery := UserEventsQuery{UserID: 1, Status: "active", Limit: 5}
	result1, _ := demo.queryProcessor.ExecuteUserEventsQuery(ctx, userQuery)
	log.Printf("User events query result: %s", result1.Message)
	
	eventQuery := EventDetailsQuery{EventID: 1, UserID: 1}
	result2, _ := demo.queryProcessor.ExecuteEventDetailsQuery(ctx, eventQuery)
	log.Printf("Event details query result: %s", result2.Message)
	
	// Demo Phase 6: Performance Monitoring
	log.Println("\n📊 === PHASE 6: PERFORMANCE MONITORING ===")
	
	time.Sleep(2 * time.Second)
	
	metrics := demo.perfMonitor.GetMetrics()
	log.Printf("Current metrics: EPS=%.1f, Latency=%.1fms, Memory=%.1fMB", 
		metrics.EventsPerSecond, metrics.AverageLatencyMS, metrics.MemoryUsageMB)
	
	alerts := demo.perfMonitor.GetAlerts()
	if len(alerts) > 0 {
		log.Printf("Active alerts: %v", alerts)
	}
	
	// Check saga status
	if saga, exists := demo.sagaManager.GetSaga(saga1.ID); exists {
		log.Printf("Saga %s status: %s (Step %d/%d)", saga.ID, saga.Status, saga.CurrentStep, saga.TotalSteps)
	}
	if saga, exists := demo.sagaManager.GetSaga(saga2.ID); exists {
		log.Printf("Saga %s status: %s (Step %d/%d)", saga.ID, saga.Status, saga.CurrentStep, saga.TotalSteps)
	}
	
	log.Println("\n✅ Advanced Event Sourcing Demo completed successfully!")
}

func (demo *AdvancedEventSourcingDemo) Stop() {
	demo.perfMonitor.Stop()
	log.Println("🛑 Advanced Event Sourcing Demo stopped")
}

// Getter methods for accessing components
func (demo *AdvancedEventSourcingDemo) GetSagaManager() *SagaManager {
	return demo.sagaManager
}

func (demo *AdvancedEventSourcingDemo) GetQueryProcessor() *QueryProcessor {
	return demo.queryProcessor
}

func (demo *AdvancedEventSourcingDemo) GetPerfMonitor() *PerformanceMonitor {
	return demo.perfMonitor
}
