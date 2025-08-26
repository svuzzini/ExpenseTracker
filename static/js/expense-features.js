// Enhanced ExpenseTracker functionality for Splitwise-like features

// Load and display contributions
async function loadContributions() {
    try {
        const response = await apiCall(`/api/v1/events/${eventId}/contributions`);
        if (response.ok) {
            const contributions = await response.json();
            displayContributions(contributions);
        }
    } catch (error) {
        console.error('Failed to load contributions:', error);
    }
}

function displayContributions(contributions) {
    const container = document.getElementById('contributions-container');
    
    if (!contributions || contributions.length === 0) {
        container.innerHTML = `
            <div class="px-6 py-8 text-center text-gray-500">
                <i class="fas fa-plus-circle text-3xl mb-4 opacity-50"></i>
                <p>No contributions yet</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = contributions.map(contribution => `
        <div class="px-6 py-4">
            <div class="flex items-center justify-between">
                <div class="flex items-center space-x-3">
                    <div class="w-8 h-8 bg-green-100 rounded-full flex items-center justify-center">
                        <i class="fas fa-plus text-green-600"></i>
                    </div>
                    <div>
                        <p class="text-sm font-medium text-gray-900">${contribution.user_name}</p>
                        <p class="text-xs text-gray-500">${formatDate(contribution.timestamp)}</p>
                    </div>
                </div>
                <div class="text-right">
                    <p class="text-sm font-semibold text-green-600">
                        +${formatCurrency(contribution.amount, currentEvent?.currency || 'USD')}
                    </p>
                </div>
            </div>
        </div>
    `).join('');
}

// Load and display participants
async function loadParticipants() {
    try {
        const response = await apiCall(`/api/v1/events/${eventId}`);
        if (response.ok) {
            const event = await response.json();
            displayParticipants(event.participants || []);
        }
    } catch (error) {
        console.error('Failed to load participants:', error);
    }
}

function displayParticipants(participants) {
    const container = document.getElementById('participants-container');
    
    if (!participants || participants.length === 0) {
        container.innerHTML = `
            <div class="text-center text-gray-500">
                <i class="fas fa-users text-2xl mb-2 opacity-50"></i>
                <p>No participants</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = `
        <div class="space-y-3">
            ${participants.map(participant => `
                <div class="flex items-center space-x-3">
                    <div class="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
                        <span class="text-sm font-medium text-blue-600">
                            ${participant.first_name?.[0] || participant.username[0]}
                        </span>
                    </div>
                    <div class="flex-1">
                        <p class="text-sm font-medium text-gray-900">
                            ${participant.first_name} ${participant.last_name}
                        </p>
                        <p class="text-xs text-gray-500">${participant.role}</p>
                    </div>
                </div>
            `).join('')}
        </div>
    `;
}

// Load and display balances - Core Splitwise feature
async function loadBalances() {
    try {
        const response = await apiCall(`/api/v1/settlements/event/${eventId}/balances`);
        if (response.ok) {
            const balances = await response.json();
            displayBalances(balances);
        }
    } catch (error) {
        console.error('Failed to load balances:', error);
    }
}

function displayBalances(balances) {
    const container = document.getElementById('balances-container');
    
    if (!balances || balances.length === 0) {
        container.innerHTML = `
            <div class="text-center text-gray-500">
                <i class="fas fa-balance-scale text-2xl mb-2 opacity-50"></i>
                <p>No balances to show</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = `
        <div class="space-y-3">
            ${balances.map(balance => `
                <div class="flex items-center justify-between">
                    <div class="flex items-center space-x-3">
                        <div class="w-8 h-8 bg-gray-100 rounded-full flex items-center justify-center">
                            <span class="text-sm font-medium text-gray-600">
                                ${balance.user_name?.[0] || 'U'}
                            </span>
                        </div>
                        <span class="text-sm font-medium text-gray-900">${balance.user_name}</span>
                    </div>
                    <div class="text-right">
                        <span class="text-sm font-semibold ${balance.net_balance >= 0 ? 'text-green-600' : 'text-red-600'}">
                            ${balance.net_balance >= 0 ? '+' : ''}${formatCurrency(balance.net_balance, currentEvent?.currency || 'USD')}
                        </span>
                        <div class="text-xs text-gray-500">
                            ${balance.net_balance >= 0 ? 'gets back' : 'owes'}
                        </div>
                    </div>
                </div>
            `).join('')}
        </div>
    `;
}

// Load settlements - Core Splitwise feature for "Settle Up"
async function loadSettlements() {
    try {
        const response = await apiCall(`/api/v1/settlements/event/${eventId}/`);
        if (response.ok) {
            const settlements = await response.json();
            displaySettlements(settlements);
        }
    } catch (error) {
        console.error('Failed to load settlements:', error);
    }
}

function displaySettlements(settlements) {
    const container = document.getElementById('settlements-container');
    
    if (!settlements || settlements.length === 0) {
        container.innerHTML = `
            <div class="text-center text-gray-500">
                <i class="fas fa-handshake text-2xl mb-2 opacity-50"></i>
                <p>No settlements yet</p>
                <button onclick="generateSettlements()" class="mt-2 text-sm bg-blue-600 text-white px-3 py-1 rounded-lg hover:bg-blue-700">
                    <i class="fas fa-calculator mr-1"></i>Calculate Settlements
                </button>
            </div>
        `;
        return;
    }
    
    container.innerHTML = `
        <div class="space-y-3">
            ${settlements.map(settlement => `
                <div class="p-3 bg-blue-50 rounded-lg">
                    <div class="flex items-center justify-between">
                        <div class="text-sm">
                            <span class="font-medium">${settlement.from_user_name}</span>
                            <span class="text-gray-600"> owes </span>
                            <span class="font-medium">${settlement.to_user_name}</span>
                        </div>
                        <div class="text-sm font-semibold text-blue-600">
                            ${formatCurrency(settlement.amount, currentEvent?.currency || 'USD')}
                        </div>
                    </div>
                    <div class="mt-2 flex items-center justify-between">
                        <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusBadgeClass(settlement.status)}">
                            ${settlement.status}
                        </span>
                        ${settlement.status === 'pending' ? `
                            <button onclick="markSettlementPaid(${settlement.id})" class="text-xs bg-green-600 text-white px-2 py-1 rounded hover:bg-green-700">
                                Mark as Paid
                            </button>
                        ` : ''}
                    </div>
                </div>
            `).join('')}
        </div>
    `;
}

// Enhanced form submission handlers
async function setupExpenseFormHandlers() {
    // Add Expense Form
    document.getElementById('add-expense-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const formData = new FormData(e.target);
        const expenseData = {
            amount: parseFloat(formData.get('amount')),
            description: formData.get('description'),
            category_id: parseInt(formData.get('category_id')),
            date: formData.get('date'),
            split_type: formData.get('split_type') || 'equal',
            location: formData.get('location') || '',
            vendor: formData.get('vendor') || '',
            notes: formData.get('notes') || ''
        };
        
        try {
            const response = await apiCall(`/api/v1/expenses/event/${eventId}/`, {
                method: 'POST',
                body: JSON.stringify(expenseData)
            });
            
            if (response.ok) {
                showToast('Expense added successfully! 💰', 'success');
                closeAddExpenseModal();
                await refreshAllData();
            } else {
                const error = await response.json();
                showToast(error.error || 'Failed to add expense', 'error');
            }
        } catch (error) {
            showToast('Failed to add expense', 'error');
        }
    });
    
    // Add Contribution Form
    document.getElementById('add-contribution-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const formData = new FormData(e.target);
        const contributionData = {
            amount: parseFloat(formData.get('amount')),
            notes: formData.get('notes') || ''
        };
        
        try {
            const response = await apiCall(`/api/v1/events/${eventId}/contributions`, {
                method: 'POST',
                body: JSON.stringify(contributionData)
            });
            
            if (response.ok) {
                showToast('Contribution added successfully! 💚', 'success');
                closeAddContributionModal();
                await refreshAllData();
            } else {
                const error = await response.json();
                showToast(error.error || 'Failed to add contribution', 'error');
            }
        } catch (error) {
            showToast('Failed to add contribution', 'error');
        }
    });
}

// Refresh all data - like Splitwise real-time updates
async function refreshAllData() {
    await Promise.all([
        loadEventData(),
        loadExpenses(),
        loadContributions(),
        loadParticipants(),
        loadBalances(),
        loadSettlements()
    ]);
}

// Core Splitwise feature - Generate optimal settlements
async function generateSettlementsFromFeatures() {
    try {
        console.log('DEBUG: generateSettlements function called');
        console.log('DEBUG: eventId =', eventId);
        showToast('Calculating optimal settlements...', 'info');
        
        const response = await apiCall(`/api/v1/settlements/event/${eventId}/generate`, {
            method: 'POST'
        });
        
        if (response.ok) {
            const result = await response.json();
            showToast(`Generated ${result.settlements_created || 0} settlements! 🎯`, 'success');
            await loadSettlements();
            await loadBalances();
        } else {
            const error = await response.json();
            showToast(error.error || 'Failed to generate settlements', 'error');
        }
    } catch (error) {
        console.error('DEBUG: Error in generateSettlementsFromFeatures:', error);
        showToast('Failed to generate settlements', 'error');
    }
}

// Note: generateSettlements function is now defined in the template to avoid conflicts

// Mark settlement as paid - like Splitwise
async function markSettlementPaid(settlementId) {
    try {
        const response = await apiCall(`/api/v1/settlements/${settlementId}/complete`, {
            method: 'POST'
        });
        
        if (response.ok) {
            showToast('Settlement marked as paid! ✅', 'success');
            await loadSettlements();
            await loadBalances();
        } else {
            const error = await response.json();
            showToast(error.error || 'Failed to mark settlement as paid', 'error');
        }
    } catch (error) {
        showToast('Failed to mark settlement as paid', 'error');
    }
}

// Utility functions
function getCategoryIcon(categoryId) {
    const category = categories.find(c => c.id === categoryId);
    return category ? category.icon : '📦';
}

function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', { 
        month: 'short', 
        day: 'numeric',
        year: date.getFullYear() !== new Date().getFullYear() ? 'numeric' : undefined
    });
}

function getStatusBadgeClass(status) {
    switch (status) {
        case 'approved':
        case 'completed':
            return 'bg-green-100 text-green-800';
        case 'pending':
            return 'bg-yellow-100 text-yellow-800';
        case 'rejected':
            return 'bg-red-100 text-red-800';
        default:
            return 'bg-gray-100 text-gray-800';
    }
}

// Additional features
function refreshExpenses() {
    loadExpenses();
}

function refreshContributions() {
    loadContributions();
}

async function showExpenseDetails(expenseId) {
    const token = localStorage.getItem('token');
    
    // Check if token exists
    if (!token) {
        showToast('Please log in to view expense details', 'error');
        return;
    }
    
    // Make sure we have the user role
    if (currentUserRole === null && typeof getCurrentUserRole === 'function') {
        console.log('Loading user role first...');
        await getCurrentUserRole();
    }
    
    console.log('Fetching expense details for ID:', expenseId);
    
    try {
        const response = await fetch(`/api/v1/expenses/${expenseId}`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        console.log('Response status:', response.status);
        
        if (response.status === 401) {
            throw new Error('Authentication failed. Please log in again.');
        }
        
        if (response.status === 403) {
            throw new Error('You do not have permission to view this expense.');
        }
        
        if (response.status === 404) {
            throw new Error('Expense not found.');
        }
        
        if (!response.ok) {
            throw new Error(`Server error: ${response.status}`);
        }
        
        const expense = await response.json();
        console.log('Expense data received:', expense);
        showExpenseModal(expense);
    } catch (error) {
        console.error('Error fetching expense details:', error);
        showToast(error.message || 'Failed to load expense details', 'error');
    }
}

function showExpenseModal(expense) {
    const modal = document.createElement('div');
    modal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4';
    
    // Check if user can approve this expense
    const canApprove = expense.status === 'pending' && (currentUserRole === 'owner' || currentUserRole === 'admin');
    
    console.log('Expense status:', expense.status, 'User role:', currentUserRole, 'Can approve:', canApprove);
    
    modal.innerHTML = `
        <div class="bg-white rounded-lg max-w-2xl w-full max-h-90vh overflow-y-auto">
            <div class="p-6 border-b border-gray-200 flex items-center justify-between">
                <h3 class="text-lg font-semibold text-gray-900">Expense Details</h3>
                <button onclick="closeExpenseModal()" class="text-gray-400 hover:text-gray-600">
                    <i class="fas fa-times text-xl"></i>
                </button>
            </div>
            
            <div class="p-6">
                <!-- Expense Info -->
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
                    <div>
                        <h4 class="text-sm font-medium text-gray-500 mb-2">Basic Information</h4>
                        <div class="space-y-3">
                            <div>
                                <span class="text-sm text-gray-500">Description:</span>
                                <p class="font-medium">${expense.description}</p>
                            </div>
                            <div>
                                <span class="text-sm text-gray-500">Amount:</span>
                                <p class="font-medium text-lg">${formatCurrency(expense.amount, expense.currency)}</p>
                            </div>
                            <div>
                                <span class="text-sm text-gray-500">Date:</span>
                                <p class="font-medium">${formatDate(expense.date)}</p>
                            </div>
                            <div>
                                <span class="text-sm text-gray-500">Status:</span>
                                <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusBadgeClass(expense.status)}">
                                    ${expense.status}
                                </span>
                            </div>
                        </div>
                    </div>
                    
                    <div>
                        <h4 class="text-sm font-medium text-gray-500 mb-2">Additional Details</h4>
                        <div class="space-y-3">
                            <div>
                                <span class="text-sm text-gray-500">Submitted by:</span>
                                <p class="font-medium">${expense.submitter?.username || 'Unknown'}</p>
                            </div>
                            <div>
                                <span class="text-sm text-gray-500">Category:</span>
                                <p class="font-medium">${expense.category?.name || 'Uncategorized'}</p>
                            </div>
                            ${expense.location ? `
                            <div>
                                <span class="text-sm text-gray-500">Location:</span>
                                <p class="font-medium">${expense.location}</p>
                            </div>
                            ` : ''}
                            ${expense.vendor ? `
                            <div>
                                <span class="text-sm text-gray-500">Vendor:</span>
                                <p class="font-medium">${expense.vendor}</p>
                            </div>
                            ` : ''}
                        </div>
                    </div>
                </div>
                
                <!-- Expense Shares -->
                ${expense.shares && expense.shares.length > 0 ? `
                <div class="mb-6">
                    <h4 class="text-sm font-medium text-gray-500 mb-3">Expense Split</h4>
                    <div class="bg-gray-50 rounded-lg p-4">
                        <div class="space-y-2">
                            ${expense.shares.map(share => `
                                <div class="flex justify-between items-center">
                                    <span class="text-sm">${share.user?.username || 'Unknown'}</span>
                                    <span class="text-sm font-medium">${formatCurrency(share.amount, expense.currency)} (${share.percentage}%)</span>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                </div>
                ` : ''}
                
                <!-- Notes -->
                ${expense.notes ? `
                <div class="mb-6">
                    <h4 class="text-sm font-medium text-gray-500 mb-2">Notes</h4>
                    <p class="text-sm text-gray-700 bg-gray-50 rounded-lg p-3">${expense.notes}</p>
                </div>
                ` : ''}
                
                <!-- Rejection Reason -->
                ${expense.status === 'rejected' && expense.rejection_reason ? `
                <div class="mb-6">
                    <h4 class="text-sm font-medium text-red-600 mb-2">Rejection Reason</h4>
                    <p class="text-sm text-red-700 bg-red-50 rounded-lg p-3">${expense.rejection_reason}</p>
                </div>
                ` : ''}
            </div>
            
            <!-- Action Buttons -->
            ${canApprove ? `
            <div class="px-6 py-4 bg-gray-50 border-t border-gray-200 flex items-center justify-between">
                <div class="flex items-center space-x-3">
                    <button onclick="approveExpense(${expense.id})" class="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 transition duration-150">
                        <i class="fas fa-check mr-2"></i>Approve
                    </button>
                    <button onclick="showRejectModal(${expense.id})" class="bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700 transition duration-150">
                        <i class="fas fa-times mr-2"></i>Reject
                    </button>
                </div>
                <button onclick="closeExpenseModal()" class="text-gray-500 hover:text-gray-700 px-4 py-2">
                    Close
                </button>
            </div>
            ` : `
            <div class="px-6 py-4 bg-gray-50 border-t border-gray-200 flex justify-end">
                <button onclick="closeExpenseModal()" class="bg-gray-600 text-white px-4 py-2 rounded-lg hover:bg-gray-700 transition duration-150">
                    Close
                </button>
            </div>
            `}
        </div>
    `;
    
    document.body.appendChild(modal);
}

function closeExpenseModal() {
    const modal = document.querySelector('.fixed.inset-0.bg-black.bg-opacity-50');
    if (modal) {
        modal.remove();
    }
}

function approveExpense(expenseId) {
    fetch(`/api/v1/expenses/${expenseId}/review`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
            action: 'approve'
        })
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to approve expense');
        }
        return response.json();
    })
    .then(data => {
        showToast('Expense approved successfully!', 'success');
        closeExpenseModal();
        // Refresh the expenses list and balances
        if (typeof loadExpenses === 'function') loadExpenses();
        if (typeof viewBalances === 'function') viewBalances();
    })
    .catch(error => {
        console.error('Error approving expense:', error);
        showToast('Failed to approve expense', 'error');
    });
}

function showRejectModal(expenseId) {
    const rejectModal = document.createElement('div');
    rejectModal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-60 p-4';
    
    rejectModal.innerHTML = `
        <div class="bg-white rounded-lg max-w-md w-full">
            <div class="p-6 border-b border-gray-200">
                <h3 class="text-lg font-semibold text-gray-900">Reject Expense</h3>
            </div>
            
            <div class="p-6">
                <p class="text-sm text-gray-600 mb-4">Please provide a reason for rejecting this expense:</p>
                <textarea id="rejection-reason" rows="3" class="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-red-500" placeholder="Enter rejection reason..."></textarea>
            </div>
            
            <div class="px-6 py-4 bg-gray-50 border-t border-gray-200 flex items-center justify-end space-x-3">
                <button onclick="closeRejectModal()" class="text-gray-500 hover:text-gray-700 px-4 py-2">
                    Cancel
                </button>
                <button onclick="rejectExpense(${expenseId})" class="bg-red-600 text-white px-4 py-2 rounded-lg hover:bg-red-700 transition duration-150">
                    Reject Expense
                </button>
            </div>
        </div>
    `;
    
    document.body.appendChild(rejectModal);
}

function closeRejectModal() {
    const modal = document.querySelector('.fixed.inset-0.bg-black.bg-opacity-50.z-60');
    if (modal) {
        modal.remove();
    }
}

function rejectExpense(expenseId) {
    const reason = document.getElementById('rejection-reason').value.trim();
    
    if (!reason) {
        showToast('Please provide a rejection reason', 'error');
        return;
    }
    
    fetch(`/api/v1/expenses/${expenseId}/review`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
            action: 'reject',
            rejection_reason: reason
        })
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to reject expense');
        }
        return response.json();
    })
    .then(data => {
        showToast('Expense rejected', 'warning');
        closeRejectModal();
        closeExpenseModal();
        // Refresh the expenses list and balances
        if (typeof loadExpenses === 'function') loadExpenses();
        if (typeof viewBalances === 'function') viewBalances();
    })
    .catch(error => {
        console.error('Error rejecting expense:', error);
        showToast('Failed to reject expense', 'error');
    });
}

// Event sharing
function shareEvent() {
    if (currentEvent) {
        const shareText = `Join my expense group: ${currentEvent.name}\nCode: ${currentEvent.code}\n\nTrack and split expenses easily!`;
        if (navigator.share) {
            navigator.share({
                title: 'Join ExpenseTracker Event',
                text: shareText,
                url: window.location.href
            });
        } else {
            navigator.clipboard.writeText(shareText);
            showToast('Event details copied to clipboard! 📋', 'success');
        }
    }
}

// Enhanced toast notifications
function showEnhancedToast(message, type = 'info', duration = 3000) {
    const toast = document.createElement('div');
    toast.className = `fixed top-4 right-4 p-4 rounded-lg text-white z-50 shadow-lg transform transition-all duration-300 ${
        type === 'error' ? 'bg-red-500' : 
        type === 'success' ? 'bg-green-500' : 
        type === 'warning' ? 'bg-yellow-500' : 'bg-blue-500'
    }`;
    
    const icon = type === 'error' ? '❌' : 
                type === 'success' ? '✅' : 
                type === 'warning' ? '⚠️' : 'ℹ️';
    
    toast.innerHTML = `
        <div class="flex items-center space-x-2">
            <span class="text-lg">${icon}</span>
            <span>${message}</span>
        </div>
    `;
    
    document.body.appendChild(toast);
    
    // Animate in
    setTimeout(() => toast.classList.add('translate-x-0'), 10);
    
    // Auto remove
    setTimeout(() => {
        toast.classList.add('translate-x-full', 'opacity-0');
        setTimeout(() => toast.remove(), 300);
    }, duration);
}

// Event menu functions
function toggleEventMenu() {
    const dropdown = document.getElementById('event-dropdown');
    dropdown.classList.toggle('hidden');
}

// Enhanced initialization
function initializeExpenseFeatures() {
    setupExpenseFormHandlers();
    
    // Set up filter
    const expenseFilter = document.getElementById('expense-filter');
    if (expenseFilter) {
        expenseFilter.addEventListener('change', (e) => {
            // TODO: Implement filtering
            loadExpenses();
        });
    }
    
    // Close dropdown when clicking outside
    document.addEventListener('click', function(event) {
        const menu = document.getElementById('event-menu');
        const dropdown = document.getElementById('event-dropdown');
        
        if (menu && dropdown && !menu.contains(event.target)) {
            dropdown.classList.add('hidden');
        }
    });
    
    // Auto-refresh every 30 seconds (like Splitwise)
    setInterval(refreshAllData, 30000);
}

// Override the showToast function to use enhanced version
window.showToast = showEnhancedToast;

