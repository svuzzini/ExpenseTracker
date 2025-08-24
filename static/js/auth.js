// Authentication utilities for ExpenseTracker

class AuthManager {
    constructor() {
        this.token = null;
        this.user = null;
        this.refreshTimer = null;
        this.init();
    }

    init() {
        this.loadStoredAuth();
        this.setupAutoRefresh();
        this.handleRouteProtection();
    }

    loadStoredAuth() {
        this.token = localStorage.getItem('token');
        const userString = localStorage.getItem('user');
        
        if (userString) {
            try {
                this.user = JSON.parse(userString);
            } catch (error) {
                console.error('Error parsing stored user data:', error);
                this.clearAuth();
            }
        }
    }

    isAuthenticated() {
        return this.token !== null && this.user !== null;
    }

    async login(email, password) {
        try {
            const response = await fetch('/api/v1/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ email, password })
            });

            const data = await response.json();

            if (response.ok) {
                this.setAuth(data.token, data.user);
                this.setupAutoRefresh();
                return { success: true, user: data.user };
            } else {
                return { success: false, error: data.error || 'Login failed' };
            }
        } catch (error) {
            return { success: false, error: 'Network error: ' + error.message };
        }
    }

    async register(userData) {
        try {
            const response = await fetch('/api/v1/auth/register', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(userData)
            });

            const data = await response.json();

            if (response.ok) {
                this.setAuth(data.token, data.user);
                this.setupAutoRefresh();
                return { success: true, user: data.user };
            } else {
                return { success: false, error: data.error || 'Registration failed' };
            }
        } catch (error) {
            return { success: false, error: 'Network error: ' + error.message };
        }
    }

    async logout() {
        this.clearAuth();
        this.clearAutoRefresh();
        
        // Redirect to login page
        window.location.href = '/';
    }

    async refreshToken() {
        if (!this.token) return false;

        try {
            const response = await fetch('/api/v1/auth/refresh', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (response.ok) {
                const data = await response.json();
                this.setAuth(data.token, this.user);
                return true;
            } else {
                this.logout();
                return false;
            }
        } catch (error) {
            console.error('Token refresh failed:', error);
            return false;
        }
    }

    async updateProfile(profileData) {
        if (!this.isAuthenticated()) {
            return { success: false, error: 'Not authenticated' };
        }

        try {
            const response = await apiCall('/api/v1/auth/profile', {
                method: 'PUT',
                body: JSON.stringify(profileData)
            });

            if (response.ok) {
                const updatedUser = await response.json();
                this.setAuth(this.token, updatedUser);
                return { success: true, user: updatedUser };
            } else {
                const error = await response.json();
                return { success: false, error: error.error || 'Profile update failed' };
            }
        } catch (error) {
            return { success: false, error: 'Network error: ' + error.message };
        }
    }

    async changePassword(currentPassword, newPassword, confirmPassword) {
        if (!this.isAuthenticated()) {
            return { success: false, error: 'Not authenticated' };
        }

        if (newPassword !== confirmPassword) {
            return { success: false, error: 'New passwords do not match' };
        }

        try {
            const response = await apiCall('/api/v1/auth/change-password', {
                method: 'POST',
                body: JSON.stringify({
                    current_password: currentPassword,
                    new_password: newPassword,
                    confirm_password: confirmPassword
                })
            });

            if (response.ok) {
                return { success: true };
            } else {
                const error = await response.json();
                return { success: false, error: error.error || 'Password change failed' };
            }
        } catch (error) {
            return { success: false, error: 'Network error: ' + error.message };
        }
    }

    setAuth(token, user) {
        this.token = token;
        this.user = user;
        
        localStorage.setItem('token', token);
        localStorage.setItem('user', JSON.stringify(user));
        
        // Dispatch auth state change event
        window.dispatchEvent(new CustomEvent('authStateChanged', {
            detail: { authenticated: true, user }
        }));
    }

    clearAuth() {
        this.token = null;
        this.user = null;
        
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        
        // Dispatch auth state change event
        window.dispatchEvent(new CustomEvent('authStateChanged', {
            detail: { authenticated: false, user: null }
        }));
    }

    setupAutoRefresh() {
        this.clearAutoRefresh();
        
        if (this.token) {
            // Refresh token every 20 minutes (assuming 24-hour token lifetime)
            this.refreshTimer = setInterval(() => {
                this.refreshToken();
            }, 20 * 60 * 1000);
        }
    }

    clearAutoRefresh() {
        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
            this.refreshTimer = null;
        }
    }

    handleRouteProtection() {
        const currentPath = window.location.pathname;
        const protectedRoutes = ['/dashboard', '/events', '/profile', '/settings'];
        const authRoutes = ['/', '/login', '/register'];

        // Check if current route requires authentication
        const isProtectedRoute = protectedRoutes.some(route => 
            currentPath.startsWith(route)
        );

        // Check if current route is auth-only (shouldn't be accessible when logged in)
        const isAuthRoute = authRoutes.includes(currentPath);

        if (isProtectedRoute && !this.isAuthenticated()) {
            // Redirect to login if accessing protected route without auth
            window.location.href = '/';
        } else if (isAuthRoute && this.isAuthenticated()) {
            // Redirect to dashboard if accessing auth route while logged in
            window.location.href = '/dashboard';
        }
    }

    getAuthHeader() {
        return this.token ? `Bearer ${this.token}` : null;
    }

    getUserInfo() {
        return this.user;
    }

    hasRole(role) {
        return this.user && this.user.role === role;
    }

    // Validate password strength
    validatePassword(password) {
        const requirements = {
            length: password.length >= 8,
            uppercase: /[A-Z]/.test(password),
            lowercase: /[a-z]/.test(password),
            number: /\d/.test(password),
            special: /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password)
        };

        const score = Object.values(requirements).filter(Boolean).length;
        const strength = score <= 2 ? 'weak' : score <= 3 ? 'fair' : score <= 4 ? 'good' : 'strong';

        return {
            score,
            strength,
            requirements,
            isValid: score >= 4
        };
    }

    // Check if username is available
    async checkUsernameAvailability(username) {
        if (username.length < 3) {
            return { available: false, error: 'Username must be at least 3 characters' };
        }

        try {
            const response = await fetch(`/api/v1/auth/check-username?username=${encodeURIComponent(username)}`);
            
            if (response.ok) {
                const data = await response.json();
                return { available: data.available };
            } else {
                return { available: false, error: 'Unable to check username availability' };
            }
        } catch (error) {
            return { available: false, error: 'Network error' };
        }
    }

    // Request password reset
    async requestPasswordReset(email) {
        try {
            const response = await fetch('/api/v1/auth/forgot-password', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ email })
            });

            if (response.ok) {
                return { success: true };
            } else {
                const error = await response.json();
                return { success: false, error: error.error || 'Password reset request failed' };
            }
        } catch (error) {
            return { success: false, error: 'Network error: ' + error.message };
        }
    }

    // Reset password with token
    async resetPassword(token, newPassword) {
        try {
            const response = await fetch('/api/v1/auth/reset-password', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ token, new_password: newPassword })
            });

            if (response.ok) {
                return { success: true };
            } else {
                const error = await response.json();
                return { success: false, error: error.error || 'Password reset failed' };
            }
        } catch (error) {
            return { success: false, error: 'Network error: ' + error.message };
        }
    }
}

// Create global auth manager instance
const authManager = new AuthManager();

// Auth event listeners
document.addEventListener('DOMContentLoaded', function() {
    // Handle auth state changes
    window.addEventListener('authStateChanged', function(event) {
        const { authenticated, user } = event.detail;
        
        if (authenticated) {
            console.log('User authenticated:', user);
            updateUIForAuthenticatedUser(user);
        } else {
            console.log('User logged out');
            updateUIForUnauthenticatedUser();
        }
    });

    // Handle page visibility change for token refresh
    document.addEventListener('visibilitychange', function() {
        if (!document.hidden && authManager.isAuthenticated()) {
            // Page became visible, refresh token if needed
            authManager.refreshToken();
        }
    });
});

// UI update functions
function updateUIForAuthenticatedUser(user) {
    // Update user info displays
    const userNameElements = document.querySelectorAll('[data-user-name]');
    userNameElements.forEach(element => {
        element.textContent = user.display_name || user.username;
    });

    const userAvatarElements = document.querySelectorAll('[data-user-avatar]');
    userAvatarElements.forEach(element => {
        const name = user.display_name || user.username;
        element.textContent = name ? name.charAt(0).toUpperCase() : '';
    });

    // Show/hide elements based on auth state
    const authElements = document.querySelectorAll('[data-auth-show]');
    authElements.forEach(element => {
        element.classList.remove('hidden');
    });

    const guestElements = document.querySelectorAll('[data-guest-show]');
    guestElements.forEach(element => {
        element.classList.add('hidden');
    });
}

function updateUIForUnauthenticatedUser() {
    // Hide auth-required elements
    const authElements = document.querySelectorAll('[data-auth-show]');
    authElements.forEach(element => {
        element.classList.add('hidden');
    });

    // Show guest elements
    const guestElements = document.querySelectorAll('[data-guest-show]');
    guestElements.forEach(element => {
        element.classList.remove('hidden');
    });
}

// Export auth manager for global use
window.authManager = authManager;
