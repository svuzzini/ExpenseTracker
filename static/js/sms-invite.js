// Enhanced SMS and Multi-Platform Invite Functionality for ExpenseTracker
// Provides comprehensive invite options including SMS, WhatsApp, Email, Telegram, etc.

window.InviteSystem = {
    
    // Generate default invite message
    getDefaultInviteMessage(eventName, eventCode) {
        return `Hey! 👋

I've created an expense group "${eventName}" on ExpenseTracker to easily split our costs.

🎯 Join with code: ${eventCode}

Just visit ${window.location.origin} and click "Join Event", then enter the code above.

Let's keep track of our expenses together! 💰✨`;
    },

    // Enhanced Invite Modal with SMS Support
    showEnhancedInviteModal(eventCode, eventName = 'Expense Event') {
        const modal = document.createElement('div');
        modal.className = 'modal-overlay invite-modal';
        modal.innerHTML = `
            <div class="modal-content" style="max-width: 600px;">
                <div class="modal-header">
                    <h3>Invite People to ${eventName}</h3>
                    <button class="btn-close" onclick="this.closest('.modal-overlay').remove()">
                        <i data-lucide="x"></i>
                    </button>
                </div>
                <div class="modal-body">
                    <!-- Event Code Section -->
                    <div class="form-group mb-6">
                        <label class="form-label">Event Code</label>
                        <div class="flex gap-2">
                            <input type="text" value="${eventCode}" readonly class="form-input flex-1" id="invite-code-input">
                            <button class="btn btn-secondary" onclick="InviteSystem.copyInviteCode()">
                                <i data-lucide="copy"></i>
                                Copy
                            </button>
                        </div>
                    </div>

                    <!-- Quick Share Options -->
                    <div class="mb-6">
                        <h4 class="form-label mb-3">Quick Invite Options</h4>
                        <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 0.75rem;">
                            <button class="btn btn-outline" onclick="InviteSystem.inviteViaSMS('${eventCode}', '${eventName}')" title="Send SMS text message">
                                <i data-lucide="message-square"></i>
                                Send SMS
                            </button>
                            <button class="btn btn-outline" onclick="InviteSystem.inviteViaWhatsApp('${eventCode}', '${eventName}')" title="Share via WhatsApp">
                                <i data-lucide="message-circle"></i>
                                WhatsApp
                            </button>
                            <button class="btn btn-outline" onclick="InviteSystem.inviteViaEmail('${eventCode}', '${eventName}')" title="Send via email">
                                <i data-lucide="mail"></i>
                                Email
                            </button>
                            <button class="btn btn-outline" onclick="InviteSystem.shareNativeAPI('${eventCode}', '${eventName}')" title="Use device sharing">
                                <i data-lucide="share-2"></i>
                                Share
                            </button>
                            <button class="btn btn-outline" onclick="InviteSystem.inviteViaTelegram('${eventCode}', '${eventName}')" title="Share via Telegram">
                                <i data-lucide="send"></i>
                                Telegram
                            </button>
                            <button class="btn btn-outline" onclick="InviteSystem.copyInviteLink('${eventCode}')" title="Copy invite link">
                                <i data-lucide="link"></i>
                                Copy Link
                            </button>
                        </div>
                    </div>

                    <!-- Custom Message Section -->
                    <div class="form-group mb-6">
                        <label class="form-label">Customize Your Invite Message</label>
                        <textarea id="invite-message" class="form-input" rows="6" style="font-family: ui-monospace, 'Courier New', monospace; font-size: 0.875rem; line-height: 1.5;" placeholder="Add a personal message...">${this.getDefaultInviteMessage(eventName, eventCode)}</textarea>
                        <div class="flex gap-2 mt-2">
                            <button class="btn btn-secondary btn-sm" onclick="InviteSystem.copyCustomMessage()" title="Copy custom message">
                                <i data-lucide="copy"></i>
                                Copy Message
                            </button>
                            <button class="btn btn-secondary btn-sm" onclick="InviteSystem.sendCustomSMS()" title="Send custom message via SMS">
                                <i data-lucide="message-square"></i>
                                Send via SMS
                            </button>
                            <button class="btn btn-secondary btn-sm" onclick="InviteSystem.resetToDefault('${eventName}', '${eventCode}')" title="Reset to default message">
                                <i data-lucide="refresh-cw"></i>
                                Reset
                            </button>
                        </div>
                    </div>

                    <!-- Instructions -->
                    <div style="background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%); border: 1px solid #bae6fd; border-radius: var(--border-radius-lg); padding: 1.25rem;">
                        <h4 style="color: #1e40af; font-weight: 600; margin-bottom: 0.75rem;">How your friends can join:</h4>
                        <ol style="color: #1e3a8a; font-size: 0.875rem; line-height: 1.5;" class="list-decimal list-inside space-y-1">
                            <li>Visit <strong style="background: #dbeafe; padding: 0.125rem 0.375rem; border-radius: 0.25rem; font-family: ui-monospace;">${window.location.origin}</strong></li>
                            <li>Click <strong>"Join Event"</strong> on the dashboard</li>
                            <li>Enter the event code: <strong style="background: #dbeafe; padding: 0.125rem 0.375rem; border-radius: 0.25rem; font-family: ui-monospace;">${eventCode}</strong></li>
                            <li>Start tracking and splitting expenses together! 🎉</li>
                        </ol>
                    </div>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        // Initialize Lucide icons for the modal
        if (typeof lucide !== 'undefined') {
            lucide.createIcons();
        }
        
        // Auto-select the code for easy copying
        const input = document.getElementById('invite-code-input');
        if (input) {
            input.select();
            input.focus();
        }

        // Track modal opening
        this.trackInviteMethod('modal_opened');
        
        return modal;
    },

    // SMS Invite Functions
    inviteViaSMS(eventCode, eventName) {
        const message = document.getElementById('invite-message')?.value || this.getDefaultInviteMessage(eventName, eventCode);
        const smsUrl = `sms:?body=${encodeURIComponent(message)}`;
        
        try {
            // Detect mobile vs desktop
            const isMobile = /Android|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
            
            if (isMobile) {
                // Mobile: use location.href for better app integration
                window.location.href = smsUrl;
            } else {
                // Desktop: try window.open first
                const smsWindow = window.open(smsUrl, '_self');
                if (!smsWindow) {
                    // Fallback if popup blocked
                    window.location.href = smsUrl;
                }
            }
            
            this.showSuccessMessage('Opening SMS app with invite message...');
            this.trackInviteMethod('sms');
            
            // Close modal after a short delay
            setTimeout(() => {
                const modal = document.querySelector('.invite-modal');
                if (modal) modal.remove();
            }, 1500);
            
        } catch (error) {
            console.error('SMS not supported:', error);
            this.fallbackToCopy(message, 'SMS not available on this device. Message copied to clipboard!');
        }
    },

    inviteViaWhatsApp(eventCode, eventName) {
        const message = document.getElementById('invite-message')?.value || this.getDefaultInviteMessage(eventName, eventCode);
        const whatsappUrl = `https://wa.me/?text=${encodeURIComponent(message)}`;
        
        // Try WhatsApp app first, then web
        const isMobile = /Android|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
        
        if (isMobile) {
            // Mobile: try app first
            window.location.href = `whatsapp://send?text=${encodeURIComponent(message)}`;
            
            // Fallback to web after short delay
            setTimeout(() => {
                window.open(whatsappUrl, '_blank');
            }, 1000);
        } else {
            // Desktop: open WhatsApp Web
            window.open(whatsappUrl, '_blank');
        }
        
        this.showSuccessMessage('Opening WhatsApp with invite message...');
        this.trackInviteMethod('whatsapp');
    },

    inviteViaEmail(eventCode, eventName) {
        const message = document.getElementById('invite-message')?.value || this.getDefaultInviteMessage(eventName, eventCode);
        const subject = `Join "${eventName}" on ExpenseTracker`;
        const emailUrl = `mailto:?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(message)}`;
        
        try {
            window.location.href = emailUrl;
            this.showSuccessMessage('Opening email app with invite message...');
            this.trackInviteMethod('email');
        } catch (error) {
            console.error('Email not supported:', error);
            this.fallbackToCopy(message, 'Email not available. Message copied to clipboard!');
        }
    },

    inviteViaTelegram(eventCode, eventName) {
        const message = document.getElementById('invite-message')?.value || this.getDefaultInviteMessage(eventName, eventCode);
        const telegramUrl = `https://t.me/share/url?url=${encodeURIComponent(window.location.origin)}&text=${encodeURIComponent(message)}`;
        
        window.open(telegramUrl, '_blank');
        this.showSuccessMessage('Opening Telegram with invite message...');
        this.trackInviteMethod('telegram');
    },

    shareNativeAPI(eventCode, eventName) {
        const message = document.getElementById('invite-message')?.value || this.getDefaultInviteMessage(eventName, eventCode);
        
        if (navigator.share) {
            navigator.share({
                title: `Join "${eventName}" on ExpenseTracker`,
                text: message,
                url: window.location.origin
            }).then(() => {
                this.showSuccessMessage('Invite shared successfully!');
                this.trackInviteMethod('native_share');
            }).catch((error) => {
                if (error.name !== 'AbortError') {
                    console.log('Share failed:', error);
                    this.fallbackToCopy(message, 'Sharing cancelled. Message copied to clipboard!');
                }
            });
        } else {
            this.fallbackToCopy(message, 'Native sharing not available. Message copied to clipboard!');
        }
    },

    copyInviteLink(eventCode) {
        const link = `${window.location.origin}?invite=${eventCode}`;
        this.copyToClipboard(link, 'Invite link copied to clipboard! Share this link with others.');
        this.trackInviteMethod('link_copy');
    },

    copyCustomMessage() {
        const message = document.getElementById('invite-message').value;
        this.copyToClipboard(message, 'Custom message copied to clipboard!');
        this.trackInviteMethod('message_copy');
    },

    sendCustomSMS() {
        const message = document.getElementById('invite-message').value;
        const smsUrl = `sms:?body=${encodeURIComponent(message)}`;
        
        try {
            const isMobile = /Android|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
            
            if (isMobile) {
                window.location.href = smsUrl;
            } else {
                window.open(smsUrl, '_self');
            }
            
            this.showSuccessMessage('Opening SMS app with your custom message...');
            this.trackInviteMethod('custom_sms');
            
            // Close modal after delay
            setTimeout(() => {
                const modal = document.querySelector('.invite-modal');
                if (modal) modal.remove();
            }, 1500);
            
        } catch (error) {
            console.error('SMS not supported:', error);
            this.fallbackToCopy(message, 'SMS not available. Custom message copied to clipboard!');
        }
    },

    copyInviteCode() {
        const code = document.getElementById('invite-code-input')?.value;
        if (code) {
            this.copyToClipboard(code, `Event code ${code} copied to clipboard!`);
            this.trackInviteMethod('code_copy');
        }
    },

    resetToDefault(eventName, eventCode) {
        const textarea = document.getElementById('invite-message');
        if (textarea) {
            textarea.value = this.getDefaultInviteMessage(eventName, eventCode);
            this.showSuccessMessage('Message reset to default template');
        }
    },

    // Utility Functions
    copyToClipboard(text, successMessage) {
        if (navigator.clipboard) {
            navigator.clipboard.writeText(text).then(() => {
                this.showSuccessMessage(successMessage);
            }).catch(() => {
                this.fallbackCopy(text, successMessage);
            });
        } else {
            this.fallbackCopy(text, successMessage);
        }
    },

    fallbackCopy(text, successMessage) {
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        textArea.style.top = '-999999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        
        try {
            const successful = document.execCommand('copy');
            document.body.removeChild(textArea);
            
            if (successful) {
                this.showSuccessMessage(successMessage);
            } else {
                this.showErrorMessage('Unable to copy automatically. Please copy manually.');
            }
        } catch (err) {
            document.body.removeChild(textArea);
            this.showErrorMessage('Copy operation not supported. Please copy manually.');
        }
    },

    fallbackToCopy(message, errorMsg) {
        this.copyToClipboard(message, errorMsg);
    },

    trackInviteMethod(method) {
        // Analytics tracking for invite method usage
        console.log(`Invite method used: ${method}`);
        
        // Send analytics to backend if available
        try {
            if (typeof apiCall !== 'undefined' && window.currentEventId) {
                apiCall('/api/v1/analytics/invite', {
                    method: 'POST',
                    body: JSON.stringify({ 
                        method, 
                        eventId: window.currentEventId,
                        timestamp: new Date().toISOString(),
                        userAgent: navigator.userAgent,
                        platform: this.getPlatform()
                    })
                }).catch(err => console.log('Analytics tracking failed:', err));
            }
        } catch (error) {
            console.log('Analytics not available:', error);
        }
    },

    getPlatform() {
        const userAgent = navigator.userAgent;
        if (/Android/i.test(userAgent)) return 'Android';
        if (/iPhone|iPad|iPod/i.test(userAgent)) return 'iOS';
        if (/Windows/i.test(userAgent)) return 'Windows';
        if (/Mac/i.test(userAgent)) return 'macOS';
        if (/Linux/i.test(userAgent)) return 'Linux';
        return 'Unknown';
    },

    // Message functions that integrate with existing toast system
    showSuccessMessage(message) {
        if (typeof showToast !== 'undefined') {
            showToast(message, 'success');
        } else if (typeof utils !== 'undefined' && utils.showToast) {
            utils.showToast(message, 'success');
        } else {
            console.log('SUCCESS:', message);
            this.showSimpleNotification(message, 'success');
        }
    },

    showErrorMessage(message) {
        if (typeof showToast !== 'undefined') {
            showToast(message, 'error');
        } else if (typeof utils !== 'undefined' && utils.showToast) {
            utils.showToast(message, 'error');
        } else {
            console.log('ERROR:', message);
            this.showSimpleNotification(message, 'error');
        }
    },

    showSimpleNotification(message, type) {
        // Create a simple notification if no toast system is available
        const notification = document.createElement('div');
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: ${type === 'success' ? '#10b981' : '#ef4444'};
            color: white;
            padding: 12px 20px;
            border-radius: 8px;
            z-index: 10000;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
            font-size: 14px;
            max-width: 300px;
            line-height: 1.4;
        `;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        // Auto remove after 4 seconds
        setTimeout(() => {
            if (notification.parentNode) {
                notification.style.transition = 'opacity 0.3s ease-out';
                notification.style.opacity = '0';
                setTimeout(() => {
                    if (notification.parentNode) {
                        notification.parentNode.removeChild(notification);
                    }
                }, 300);
            }
        }, 4000);
    }
};

// Backward compatibility - replace existing showInviteModal function
window.showInviteModal = function(eventCode, eventName) {
    return window.InviteSystem.showEnhancedInviteModal(eventCode, eventName);
};

// Enhanced copyEventCode function for integration
window.copyEventCodeWithSMS = function(currentEvent) {
    if (!currentEvent) {
        if (typeof showToast !== 'undefined') {
            showToast('Event details not loaded yet. Please wait a moment and try again.', 'error');
        }
        return;
    }
    
    if (!currentEvent.code) {
        if (typeof showToast !== 'undefined') {
            showToast('Event code not available', 'error');
        }
        return;
    }
    
    // Show enhanced invite modal with SMS options
    return window.InviteSystem.showEnhancedInviteModal(currentEvent.code, currentEvent.name);
};

// Auto-handle invite links in URL
document.addEventListener('DOMContentLoaded', function() {
    console.log('Enhanced SMS Invite System loaded!');
    
    // Check for invite code in URL parameters
    const urlParams = new URLSearchParams(window.location.search);
    const inviteCode = urlParams.get('invite');
    
    if (inviteCode) {
        console.log('Invite code detected in URL:', inviteCode);
        
        // Show a notification about the invite
        setTimeout(() => {
            if (typeof showToast !== 'undefined') {
                showToast(`Invite code detected: ${inviteCode}. Click "Join Event" to join!`, 'info', 5000);
            }
            
            // Auto-focus join event button if available
            const joinButton = document.querySelector('[onclick*="showJoinEventModal"], [onclick*="Join Event"]');
            if (joinButton) {
                joinButton.style.animation = 'pulse 2s infinite';
                joinButton.style.transform = 'scale(1.05)';
            }
        }, 1000);
    }
});

// Add CSS for enhanced modal styling
if (!document.getElementById('sms-invite-styles')) {
    const style = document.createElement('style');
    style.id = 'sms-invite-styles';
    style.textContent = `
        .invite-modal .modal-content {
            border-radius: var(--border-radius-xl);
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
        }
        
        .invite-modal .btn-outline {
            transition: all 0.2s ease;
        }
        
        .invite-modal .btn-outline:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        }
        
        .invite-modal #invite-message {
            resize: vertical;
            min-height: 120px;
            background: var(--color-gray-50);
            border: 1px solid var(--color-gray-300);
        }
        
        .invite-modal #invite-message:focus {
            border-color: var(--color-primary);
            box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
        }
        
        @media (max-width: 640px) {
            .invite-modal .modal-content {
                margin: 1rem;
                max-width: calc(100vw - 2rem);
            }
            
            .invite-modal [style*="grid-template-columns"] {
                grid-template-columns: 1fr !important;
            }
        }
        
        @keyframes pulse {
            0%, 100% { transform: scale(1); }
            50% { transform: scale(1.05); }
        }
    `;
    document.head.appendChild(style);
}
