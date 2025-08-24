// WebSocket manager for real-time updates in ExpenseTracker

class WebSocketManager {
    constructor() {
        this.socket = null;
        this.eventId = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.heartbeatInterval = null;
        this.isConnecting = false;
        this.listeners = new Map();
        
        this.init();
    }

    init() {
        // Auto-connect when event ID is available
        this.checkForEventId();
        
        // Setup page visibility handling
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                this.handlePageHidden();
            } else {
                this.handlePageVisible();
            }
        });

        // Setup beforeunload cleanup
        window.addEventListener('beforeunload', () => {
            this.disconnect();
        });
    }

    checkForEventId() {
        // Extract event ID from URL
        const pathParts = window.location.pathname.split('/');
        if (pathParts.includes('events') && pathParts.length > 2) {
            const eventId = pathParts[pathParts.indexOf('events') + 1];
            if (eventId && !isNaN(eventId)) {
                this.eventId = parseInt(eventId);
                this.connect();
            }
        }
    }

    connect() {
        if (!this.eventId || !authManager.isAuthenticated() || this.isConnecting) {
            return;
        }

        this.isConnecting = true;
        
        try {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const host = window.location.host;
            const token = authManager.token;
            
            // Construct WebSocket URL with auth token
            const wsUrl = `${protocol}//${host}/api/v1/ws/events/${this.eventId}?token=${token}`;
            
            this.socket = new WebSocket(wsUrl);
            
            this.socket.onopen = (event) => {
                console.log('WebSocket connected');
                this.isConnecting = false;
                this.reconnectAttempts = 0;
                this.startHeartbeat();
                this.emit('connected', { eventId: this.eventId });
                this.showConnectionStatus('connected');
            };

            this.socket.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this.handleMessage(message);
                } catch (error) {
                    console.error('Error parsing WebSocket message:', error);
                }
            };

            this.socket.onclose = (event) => {
                console.log('WebSocket disconnected:', event.code, event.reason);
                this.isConnecting = false;
                this.stopHeartbeat();
                this.emit('disconnected', { code: event.code, reason: event.reason });
                this.showConnectionStatus('disconnected');
                
                // Attempt to reconnect if not a normal closure
                if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
                    this.scheduleReconnect();
                }
            };

            this.socket.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.isConnecting = false;
                this.emit('error', error);
                this.showConnectionStatus('error');
            };

        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
            this.isConnecting = false;
        }
    }

    disconnect() {
        if (this.socket) {
            this.socket.close(1000, 'Manual disconnect');
            this.socket = null;
        }
        this.stopHeartbeat();
        this.reconnectAttempts = 0;
    }

    reconnect() {
        this.disconnect();
        setTimeout(() => {
            this.connect();
        }, this.reconnectDelay);
    }

    scheduleReconnect() {
        this.reconnectAttempts++;
        const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
        
        console.log(`Scheduling reconnect attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`);
        
        setTimeout(() => {
            if (this.reconnectAttempts <= this.maxReconnectAttempts) {
                this.connect();
            }
        }, delay);
    }

    send(message) {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify(message));
            return true;
        } else {
            console.warn('WebSocket not connected, message not sent:', message);
            return false;
        }
    }

    handleMessage(message) {
        console.log('WebSocket message received:', message);
        
        switch (message.type) {
            case 'connected':
                break;
                
            case 'pong':
                // Heartbeat response
                break;
                
            case 'expense_added':
                this.handleExpenseAdded(message);
                break;
                
            case 'expense_approved':
                this.handleExpenseApproved(message);
                break;
                
            case 'expense_rejected':
                this.handleExpenseRejected(message);
                break;
                
            case 'contribution_added':
                this.handleContributionAdded(message);
                break;
                
            case 'user_joined':
                this.handleUserJoined(message);
                break;
                
            case 'balance_updated':
                this.handleBalanceUpdated(message);
                break;
                
            case 'settlement_created':
                this.handleSettlementCreated(message);
                break;
                
            case 'settlement_completed':
                this.handleSettlementCompleted(message);
                break;
                
            case 'typing':
                this.handleTyping(message);
                break;
                
            case 'comment':
                this.handleComment(message);
                break;
                
            default:
                console.log('Unknown message type:', message.type);
        }
        
        // Emit event for custom handlers
        this.emit(message.type, message);
    }

    handleExpenseAdded(message) {
        const { expense } = message.data;
        
        showToast(`New expense added: ${expense.description}`, 'info');
        
        // Refresh expenses list if on expenses page
        if (window.location.pathname.includes('expenses')) {
            this.emit('refresh_expenses');
        }
        
        // Update dashboard if visible
        this.emit('update_dashboard');
    }

    handleExpenseApproved(message) {
        const { expense } = message.data;
        
        showToast(`Expense approved: ${expense.description}`, 'success');
        
        // Refresh relevant UI components
        this.emit('refresh_expenses');
        this.emit('refresh_balances');
        this.emit('update_dashboard');
    }

    handleExpenseRejected(message) {
        const { expense, reason } = message.data;
        
        showToast(`Expense rejected: ${expense.description}`, 'warning', 5000);
        
        // Show rejection reason if available
        if (reason) {
            setTimeout(() => {
                showToast(`Reason: ${reason}`, 'info', 3000);
            }, 1000);
        }
        
        this.emit('refresh_expenses');
    }

    handleContributionAdded(message) {
        const { contribution } = message.data;
        
        showToast(message.data.message, 'success');
        
        // Refresh balances and dashboard
        this.emit('refresh_balances');
        this.emit('refresh_contributions');
        this.emit('update_dashboard');
    }

    handleUserJoined(message) {
        const { user } = message.data;
        
        showToast(`${user.display_name || user.username} joined the event`, 'info');
        
        // Update participants list
        this.emit('refresh_participants');
    }

    handleBalanceUpdated(message) {
        const { balances } = message.data;
        
        // Update balance displays
        this.emit('refresh_balances', balances);
    }

    handleSettlementCreated(message) {
        showToast('New settlement created', 'info');
        this.emit('refresh_settlements');
    }

    handleSettlementCompleted(message) {
        showToast('Settlement completed', 'success');
        this.emit('refresh_settlements');
        this.emit('refresh_balances');
    }

    handleTyping(message) {
        // Handle typing indicators
        this.emit('user_typing', message);
    }

    handleComment(message) {
        // Handle comment messages
        this.emit('new_comment', message);
    }

    startHeartbeat() {
        this.stopHeartbeat();
        this.heartbeatInterval = setInterval(() => {
            this.send({ type: 'ping' });
        }, 30000); // Send ping every 30 seconds
    }

    stopHeartbeat() {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
    }

    handlePageHidden() {
        // Don't disconnect immediately, just stop heartbeat
        this.stopHeartbeat();
    }

    handlePageVisible() {
        // Restart heartbeat if connected
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.startHeartbeat();
        } else if (!this.isConnecting) {
            // Reconnect if not connected
            this.connect();
        }
    }

    showConnectionStatus(status) {
        const statusElement = document.getElementById('connection-status');
        if (!statusElement) return;

        const statusConfig = {
            connected: {
                text: 'Connected',
                class: 'bg-green-100 text-green-800',
                icon: 'fas fa-wifi'
            },
            disconnected: {
                text: 'Disconnected',
                class: 'bg-red-100 text-red-800',
                icon: 'fas fa-wifi-slash'
            },
            error: {
                text: 'Connection Error',
                class: 'bg-red-100 text-red-800',
                icon: 'fas fa-exclamation-triangle'
            }
        };

        const config = statusConfig[status];
        if (config) {
            statusElement.className = `px-3 py-1 rounded-full text-xs font-medium ${config.class}`;
            statusElement.innerHTML = `<i class="${config.icon} mr-1"></i>${config.text}`;
            
            // Auto-hide success status after 3 seconds
            if (status === 'connected') {
                setTimeout(() => {
                    statusElement.classList.add('hidden');
                }, 3000);
            } else {
                statusElement.classList.remove('hidden');
            }
        }
    }

    // Event listener management
    on(event, callback) {
        if (!this.listeners.has(event)) {
            this.listeners.set(event, []);
        }
        this.listeners.get(event).push(callback);
    }

    off(event, callback) {
        if (this.listeners.has(event)) {
            const callbacks = this.listeners.get(event);
            const index = callbacks.indexOf(callback);
            if (index > -1) {
                callbacks.splice(index, 1);
            }
        }
    }

    emit(event, data) {
        if (this.listeners.has(event)) {
            this.listeners.get(event).forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error('Error in WebSocket event callback:', error);
                }
            });
        }
    }

    // Public methods for sending messages
    sendTyping() {
        this.send({
            type: 'typing',
            data: { typing: true }
        });
    }

    sendComment(comment) {
        this.send({
            type: 'comment',
            data: { comment }
        });
    }

    // Utility methods
    isConnected() {
        return this.socket && this.socket.readyState === WebSocket.OPEN;
    }

    getConnectionState() {
        if (!this.socket) return 'disconnected';
        
        switch (this.socket.readyState) {
            case WebSocket.CONNECTING:
                return 'connecting';
            case WebSocket.OPEN:
                return 'connected';
            case WebSocket.CLOSING:
                return 'disconnecting';
            case WebSocket.CLOSED:
                return 'disconnected';
            default:
                return 'unknown';
        }
    }

    setEventId(eventId) {
        if (this.eventId !== eventId) {
            this.disconnect();
            this.eventId = eventId;
            if (eventId) {
                this.connect();
            }
        }
    }
}

// Create global WebSocket manager instance
const wsManager = new WebSocketManager();

// Auto-setup connection status indicator
document.addEventListener('DOMContentLoaded', function() {
    // Create connection status indicator if it doesn't exist
    if (!document.getElementById('connection-status')) {
        const statusElement = document.createElement('div');
        statusElement.id = 'connection-status';
        statusElement.className = 'fixed top-4 left-4 z-50 hidden';
        document.body.appendChild(statusElement);
    }
});

// Export for global use
window.wsManager = wsManager;
