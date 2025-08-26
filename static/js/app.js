// Main application JavaScript for ExpenseTracker

class ExpenseTrackerApp {
    constructor() {
        this.currentPage = null;
        this.eventId = null;
        this.user = null;
        this.init();
    }

    init() {
        this.detectCurrentPage();
        this.loadUserInfo();
        this.setupGlobalEventListeners();
        this.setupWebSocketListeners();
        this.setupPageSpecificFeatures();
    }

    detectCurrentPage() {
        const path = window.location.pathname;
        
        if (path === '/' || path === '/login') {
            this.currentPage = 'login';
        } else if (path === '/register') {
            this.currentPage = 'register';
        } else if (path === '/dashboard') {
            this.currentPage = 'dashboard';
        } else if (path.startsWith('/events/')) {
            this.currentPage = 'event';
            this.eventId = parseInt(path.split('/')[2]);
        } else if (path === '/profile') {
            this.currentPage = 'profile';
        } else if (path === '/settings') {
            this.currentPage = 'settings';
        }
    }

    loadUserInfo() {
        this.user = authManager.getUserInfo();
    }

    setupGlobalEventListeners() {
        // Handle auth state changes
        window.addEventListener('authStateChanged', (event) => {
            this.user = event.detail.user;
            this.handleAuthStateChange(event.detail.authenticated);
        });

        // Handle escape key for modals
        document.addEventListener('keydown', (event) => {
            if (event.key === 'Escape') {
                this.closeAllModals();
            }
        });

        // Handle clicks outside modals
        document.addEventListener('click', (event) => {
            if (event.target.classList.contains('modal-backdrop')) {
                this.closeAllModals();
            }
        });

        // Setup service worker
        this.setupServiceWorker();
    }

    setupWebSocketListeners() {
        if (this.eventId) {
            wsManager.setEventId(this.eventId);
            
            // Setup page-specific WebSocket event handlers
            wsManager.on('refresh_expenses', () => {
                if (typeof refreshExpenses === 'function') {
                    refreshExpenses();
                }
            });

            wsManager.on('refresh_balances', () => {
                if (typeof refreshBalances === 'function') {
                    refreshBalances();
                }
            });

            wsManager.on('refresh_participants', () => {
                if (typeof refreshParticipants === 'function') {
                    refreshParticipants();
                }
            });

            wsManager.on('update_dashboard', () => {
                if (this.currentPage === 'dashboard' && typeof loadUserEvents === 'function') {
                    loadUserEvents();
                }
            });
        }
    }

    setupPageSpecificFeatures() {
        switch (this.currentPage) {
            case 'login':
                this.setupLoginPage();
                break;
            case 'register':
                this.setupRegisterPage();
                break;
            case 'dashboard':
                this.setupDashboardPage();
                break;
            case 'event':
                this.setupEventPage();
                break;
            case 'profile':
                this.setupProfilePage();
                break;
        }
    }

    setupLoginPage() {
        // Auto-focus email field
        const emailField = document.getElementById('email');
        if (emailField) {
            emailField.focus();
        }

        // Demo account quick login
        const demoInfo = document.querySelector('.demo-account-info');
        if (demoInfo) {
            demoInfo.addEventListener('click', () => {
                document.getElementById('email').value = 'demo@expensetracker.com';
                document.getElementById('password').value = 'demo123';
            });
        }
    }

    setupRegisterPage() {
        // Real-time username availability checking
        const usernameField = document.getElementById('username');
        if (usernameField) {
            const checkUsername = debounce(async (username) => {
                if (username.length >= 3) {
                    const result = await authManager.checkUsernameAvailability(username);
                    this.updateUsernameValidation(result);
                }
            }, 500);

            usernameField.addEventListener('input', (e) => {
                checkUsername(e.target.value);
            });
        }

        // Auto-focus first name field
        const firstNameField = document.getElementById('first_name');
        if (firstNameField) {
            firstNameField.focus();
        }
    }

    setupDashboardPage() {
        // Setup QR scanner if supported
        this.setupQRScanner();
        
        // Setup offline detection
        this.setupOfflineDetection();
        
        // Load recent activity
        this.loadRecentActivity();
    }

    setupEventPage() {
        // Load event data
        this.loadEventData();
        
        // Setup real-time features
        this.setupRealTimeFeatures();
        
        // Setup expense form if present
        this.setupExpenseForm();
        
        // Setup settlement features
        this.setupSettlementFeatures();
    }

    setupProfilePage() {
        // Load user profile
        this.loadUserProfile();
        
        // Setup profile form validation
        this.setupProfileFormValidation();
    }

    handleAuthStateChange(authenticated) {
        if (!authenticated && this.requiresAuth()) {
            window.location.href = '/';
        } else if (authenticated && this.isAuthPage()) {
            window.location.href = '/dashboard';
        }
    }

    requiresAuth() {
        return ['dashboard', 'event', 'profile', 'settings'].includes(this.currentPage);
    }

    isAuthPage() {
        return ['login', 'register'].includes(this.currentPage);
    }

    closeAllModals() {
        const modals = document.querySelectorAll('.modal, [id$="-modal"]');
        modals.forEach(modal => {
            modal.classList.add('hidden');
        });
    }

    async setupQRScanner() {
        if ('mediaDevices' in navigator && 'getUserMedia' in navigator.mediaDevices) {
            const scanButton = document.getElementById('scan-qr-btn');
            if (scanButton) {
                scanButton.addEventListener('click', () => {
                    this.openQRScanner();
                });
            }
        } else {
            // Hide QR scanner option if not supported
            const scanButton = document.getElementById('scan-qr-btn');
            if (scanButton) {
                scanButton.style.display = 'none';
            }
        }
    }

    async openQRScanner() {
        try {
            const stream = await navigator.mediaDevices.getUserMedia({
                video: { facingMode: 'environment' }
            });

            // Create QR scanner modal
            const modal = this.createQRScannerModal();
            document.body.appendChild(modal);

            const video = modal.querySelector('#scanner-video');
            video.srcObject = stream;

            // Setup QR code detection (would need QR scanning library)
            this.startQRDetection(video, stream, modal);

        } catch (error) {
            showToast('Camera access denied or not available', 'error');
        }
    }

    createQRScannerModal() {
        const modal = document.createElement('div');
        modal.className = 'fixed inset-0 bg-black bg-opacity-75 z-50 flex items-center justify-center p-4';
        modal.innerHTML = `
            <div class="bg-white rounded-xl max-w-md w-full p-6">
                <div class="flex justify-between items-center mb-4">
                    <h3 class="text-lg font-semibold">Scan QR Code</h3>
                    <button onclick="this.closest('.modal').remove()" class="text-gray-400 hover:text-gray-600">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <div class="relative">
                    <video id="scanner-video" class="w-full rounded-lg" autoplay></video>
                    <div class="absolute inset-0 border-2 border-white border-dashed rounded-lg"></div>
                </div>
                <p class="text-sm text-gray-600 mt-4 text-center">
                    Position the QR code within the frame
                </p>
            </div>
        `;
        return modal;
    }

    startQRDetection(video, stream, modal) {
        // This would integrate with a QR code scanning library
        // For now, just a placeholder
        setTimeout(() => {
            stream.getTracks().forEach(track => track.stop());
            modal.remove();
            showToast('QR scanning feature coming soon!', 'info');
        }, 5000);
    }

    setupOfflineDetection() {
        window.addEventListener('online', () => {
            showToast('Back online! Syncing data...', 'success');
            this.syncOfflineData();
        });

        window.addEventListener('offline', () => {
            showToast('You are offline. Changes will sync when reconnected.', 'warning');
        });
    }

    async syncOfflineData() {
        // Sync any offline changes
        const offlineData = storage.get('offline_changes', []);
        
        for (const change of offlineData) {
            try {
                await this.syncChange(change);
            } catch (error) {
                console.error('Failed to sync change:', error);
            }
        }
        
        // Clear offline data after successful sync
        storage.remove('offline_changes');
    }

    async syncChange(change) {
        // Implement sync logic based on change type
        switch (change.type) {
            case 'expense':
                return await this.syncExpense(change.data);
            case 'contribution':
                return await this.syncContribution(change.data);
            default:
                console.warn('Unknown change type:', change.type);
        }
    }

    async loadRecentActivity() {
        // Load recent activity for dashboard
        try {
            const response = await apiCall('/api/v1/recent-activity');
            if (response.ok) {
                const activities = await response.json();
                this.displayRecentActivity(activities);
            }
        } catch (error) {
            console.error('Failed to load recent activity:', error);
        }
    }

    displayRecentActivity(activities) {
        const container = document.getElementById('recent-activity');
        if (!container || activities.length === 0) return;

        container.innerHTML = activities.map(activity => `
            <div class="flex items-center space-x-3 p-3 hover:bg-gray-50 rounded-lg">
                <div class="flex-shrink-0">
                    <i class="${this.getActivityIcon(activity.type)} text-gray-400"></i>
                </div>
                <div class="flex-1 min-w-0">
                    <p class="text-sm text-gray-900">${activity.description}</p>
                    <p class="text-xs text-gray-500">${formatRelativeTime(activity.timestamp)}</p>
                </div>
                ${activity.amount ? `<div class="text-sm font-medium text-gray-900">${formatCurrency(activity.amount)}</div>` : ''}
            </div>
        `).join('');
    }

    getActivityIcon(type) {
        const icons = {
            expense: 'fas fa-receipt',
            contribution: 'fas fa-plus-circle',
            approval: 'fas fa-check-circle',
            settlement: 'fas fa-handshake'
        };
        return icons[type] || 'fas fa-circle';
    }

    async loadEventData() {
        if (!this.eventId) return;

        try {
            const response = await apiCall(`/api/v1/events/${this.eventId}`);
            if (response.ok) {
                const event = await response.json();
                this.updateEventDisplay(event);
            }
        } catch (error) {
            console.error('Failed to load event data:', error);
            showToast('Failed to load event data', 'error');
        }
    }

    updateEventDisplay(event) {
        // Update event name and details
        const eventNameElements = document.querySelectorAll('[data-event-name]');
        eventNameElements.forEach(el => el.textContent = event.name);

        const eventCodeElements = document.querySelectorAll('[data-event-code]');
        eventCodeElements.forEach(el => el.textContent = event.code);

        // Update other event details as needed
        this.updateEventStats(event);
    }

    updateEventStats(event) {
        const stats = {
            total_contributions: event.total_contributions || 0,
            total_expenses: event.total_expenses || 0,
            balance: (event.total_contributions || 0) - (event.total_expenses || 0),
            participant_count: event.participants ? event.participants.length : 0
        };

        Object.keys(stats).forEach(key => {
            const elements = document.querySelectorAll(`[data-stat="${key}"]`);
            elements.forEach(el => {
                if (['total_contributions', 'total_expenses', 'balance'].includes(key)) {
                    el.textContent = formatCurrency(stats[key], event.currency);
                } else {
                    el.textContent = stats[key];
                }
            });
        });
    }

    setupRealTimeFeatures() {
        // Setup typing indicators for comments/chat
        this.setupTypingIndicators();
        
        // Setup online user indicators
        this.setupOnlineIndicators();
    }

    setupTypingIndicators() {
        const chatInput = document.getElementById('chat-input');
        if (chatInput) {
            let typingTimer;
            
            chatInput.addEventListener('input', () => {
                wsManager.sendTyping();
                
                clearTimeout(typingTimer);
                typingTimer = setTimeout(() => {
                    // Stop typing indicator
                }, 3000);
            });
        }
    }

    setupOnlineIndicators() {
        wsManager.on('user_joined', (data) => {
            this.updateOnlineUsers(data);
        });
    }

    updateOnlineUsers(data) {
        // Update online user indicators
        const onlineIndicator = document.getElementById('online-users');
        if (onlineIndicator) {
            // Implementation would show online users
        }
    }

    setupExpenseForm() {
        const expenseForm = document.getElementById('expense-form');
        if (expenseForm) {
            this.setupExpenseFormValidation();
            this.setupReceiptUpload();
            this.setupExpenseSplitting();
        }
    }

    setupExpenseFormValidation() {
        // Real-time form validation
        const form = document.getElementById('expense-form');
        if (!form) return;

        const inputs = form.querySelectorAll('input[required], select[required], textarea[required]');
        inputs.forEach(input => {
            input.addEventListener('blur', () => {
                this.validateField(input);
            });
        });
    }

    validateField(field) {
        const isValid = field.checkValidity();
        const errorElement = document.getElementById(`${field.name}-error`);
        
        if (isValid) {
            field.classList.remove('border-red-300');
            field.classList.add('border-green-300');
            if (errorElement) errorElement.classList.add('hidden');
        } else {
            field.classList.remove('border-green-300');
            field.classList.add('border-red-300');
            if (errorElement) {
                errorElement.textContent = field.validationMessage;
                errorElement.classList.remove('hidden');
            }
        }
    }

    setupReceiptUpload() {
        const receiptInput = document.getElementById('receipt-upload');
        if (receiptInput) {
            receiptInput.addEventListener('change', (e) => {
                this.handleReceiptUpload(e.target.files[0]);
            });
        }

        // Setup drag and drop
        const dropZone = document.getElementById('receipt-drop-zone');
        if (dropZone) {
            this.setupDragAndDrop(dropZone);
        }
    }

    handleReceiptUpload(file) {
        if (!file) return;

        // Validate file
        if (!file.type.startsWith('image/')) {
            showToast('Please upload an image file', 'error');
            return;
        }

        if (file.size > 5 * 1024 * 1024) { // 5MB limit
            showToast('File size must be less than 5MB', 'error');
            return;
        }

        // Show preview
        this.showReceiptPreview(file);
        
        // Store file for later upload
        this.pendingReceipt = file;
    }

    showReceiptPreview(file) {
        const reader = new FileReader();
        reader.onload = (e) => {
            const preview = document.getElementById('receipt-preview');
            if (preview) {
                preview.innerHTML = `
                    <img src="${e.target.result}" class="max-w-full h-40 object-cover rounded-lg">
                    <button type="button" onclick="this.parentElement.innerHTML=''" class="absolute top-2 right-2 bg-red-500 text-white rounded-full w-6 h-6 flex items-center justify-center text-xs">
                        <i class="fas fa-times"></i>
                    </button>
                `;
                preview.classList.remove('hidden');
            }
        };
        reader.readAsDataURL(file);
    }

    setupDragAndDrop(dropZone) {
        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            dropZone.addEventListener(eventName, this.preventDefaults, false);
        });

        ['dragenter', 'dragover'].forEach(eventName => {
            dropZone.addEventListener(eventName, () => {
                dropZone.classList.add('border-primary-500', 'bg-primary-50');
            }, false);
        });

        ['dragleave', 'drop'].forEach(eventName => {
            dropZone.addEventListener(eventName, () => {
                dropZone.classList.remove('border-primary-500', 'bg-primary-50');
            }, false);
        });

        dropZone.addEventListener('drop', (e) => {
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                this.handleReceiptUpload(files[0]);
            }
        }, false);
    }

    preventDefaults(e) {
        e.preventDefault();
        e.stopPropagation();
    }

    setupExpenseSplitting() {
        const splitTypeSelect = document.getElementById('split-type');
        if (splitTypeSelect) {
            splitTypeSelect.addEventListener('change', (e) => {
                this.updateSplitConfiguration(e.target.value);
            });
        }
    }

    updateSplitConfiguration(splitType) {
        const container = document.getElementById('split-configuration');
        if (!container) return;

        switch (splitType) {
            case 'equal':
                container.innerHTML = '<p class="text-sm text-gray-600">Expense will be split equally among all participants.</p>';
                break;
            case 'percentage':
                this.createPercentageSplitUI(container);
                break;
            case 'custom':
                this.createCustomSplitUI(container);
                break;
            case 'weighted':
                this.createWeightedSplitUI(container);
                break;
        }
    }

    createPercentageSplitUI(container) {
        // Implementation for percentage split UI
        container.innerHTML = `
            <div class="space-y-3">
                <p class="text-sm text-gray-600">Enter percentage for each participant:</p>
                <div id="percentage-inputs">
                    <!-- Participant percentage inputs will be generated here -->
                </div>
                <div class="text-sm text-gray-500">
                    Total: <span id="total-percentage">0</span>%
                </div>
            </div>
        `;
    }

    createCustomSplitUI(container) {
        // Implementation for custom split UI
        container.innerHTML = `
            <div class="space-y-3">
                <p class="text-sm text-gray-600">Enter amount for each participant:</p>
                <div id="custom-inputs">
                    <!-- Participant amount inputs will be generated here -->
                </div>
                <div class="text-sm text-gray-500">
                    Total: <span id="total-amount">$0.00</span>
                </div>
            </div>
        `;
    }

    createWeightedSplitUI(container) {
        // Implementation for weighted split UI
        container.innerHTML = `
            <div class="space-y-3">
                <p class="text-sm text-gray-600">Enter weight for each participant:</p>
                <div id="weight-inputs">
                    <!-- Participant weight inputs will be generated here -->
                </div>
            </div>
        `;
    }

    setupSettlementFeatures() {
        // Setup settlement generation and management
        const generateBtn = document.getElementById('generate-settlements');
        if (generateBtn) {
            generateBtn.addEventListener('click', () => {
                this.generateOptimalSettlements();
            });
        }
    }

    async generateOptimalSettlements() {
        if (!this.eventId) return;

        try {
            showLoading();
            const response = await apiCall(`/api/v1/settlements/event/${this.eventId}/generate`, {
                method: 'POST'
            });

            if (response.ok) {
                const result = await response.json();
                showToast(`Generated ${result.count} optimal settlements`, 'success');
                
                // Refresh settlements display
                if (typeof refreshSettlements === 'function') {
                    refreshSettlements();
                }
            } else {
                const error = await response.json();
                throw new Error(error.error || 'Failed to generate settlements');
            }
        } catch (error) {
            showToast('Error: ' + error.message, 'error');
        } finally {
            hideLoading();
        }
    }

    updateUsernameValidation(result) {
        const field = document.getElementById('username');
        const indicator = document.getElementById('username-availability');
        
        if (!field || !indicator) return;

        if (result.available) {
            field.classList.remove('border-red-300');
            field.classList.add('border-green-300');
            indicator.innerHTML = '<i class="fas fa-check text-green-500"></i> Available';
            indicator.className = 'text-xs text-green-600';
        } else {
            field.classList.remove('border-green-300');
            field.classList.add('border-red-300');
            indicator.innerHTML = '<i class="fas fa-times text-red-500"></i> ' + (result.error || 'Not available');
            indicator.className = 'text-xs text-red-600';
        }
    }

    async setupServiceWorker() {
        if ('serviceWorker' in navigator) {
            try {
                const registration = await navigator.serviceWorker.register('/static/sw.js');
                console.log('Service Worker registered:', registration);
                
                // Handle updates
                registration.addEventListener('updatefound', () => {
                    const newWorker = registration.installing;
                    newWorker.addEventListener('statechange', () => {
                        if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                            this.showUpdateAvailableNotification();
                        }
                    });
                });
            } catch (error) {
                console.log('Service Worker registration failed:', error);
            }
        }
    }

    showUpdateAvailableNotification() {
        const notification = document.createElement('div');
        notification.className = 'fixed bottom-4 right-4 bg-blue-600 text-white p-4 rounded-lg shadow-lg z-50';
        notification.innerHTML = `
            <div class="flex items-center space-x-3">
                <i class="fas fa-download"></i>
                <div>
                    <p class="font-medium">Update Available</p>
                    <p class="text-sm opacity-90">A new version is ready to install.</p>
                </div>
                <button onclick="window.location.reload()" class="bg-blue-700 px-3 py-1 rounded text-sm">
                    Update
                </button>
            </div>
        `;
        document.body.appendChild(notification);
    }
}

// Initialize the app when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    window.app = new ExpenseTrackerApp();
});

// Export for global use
window.ExpenseTrackerApp = ExpenseTrackerApp;
