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
async function generateSettlements() {
    try {
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
        showToast('Failed to generate settlements', 'error');
    }
}

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

function showExpenseDetails(expenseId) {
    // TODO: Implement expense details modal with split breakdown
    showToast('Expense details - Coming soon!', 'info');
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

