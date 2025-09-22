class AccountApp {
    constructor() {
        this.user = null;
        this.subscriptionInfo = null;
        this.feeds = [];
        this.init();
    }

    async init() {
        await this.checkAuth();
        if (this.user) {
            this.bindEvents();
            await this.loadData();
        } else {
            // Redirect to home page if not authenticated
            window.location.href = '/';
        }
    }

    async checkAuth() {
        try {
            const response = await fetch('/auth/me');
            if (response.ok) {
                const data = await response.json();
                this.user = data.user;
                return true;
            }
        } catch (error) {
            console.error('Auth check failed:', error);
        }
        return false;
    }

    bindEvents() {
        // Modal close handlers
        const closeBtn = document.querySelector('.close');
        const cancelBtn = document.getElementById('cancel-action');
        const modal = document.getElementById('confirm-modal');

        if (closeBtn) {
            closeBtn.addEventListener('click', () => {
                this.hideModal();
            });
        }

        if (cancelBtn) {
            cancelBtn.addEventListener('click', () => {
                this.hideModal();
            });
        }

        // Click outside modal to close
        window.addEventListener('click', (e) => {
            if (e.target === modal) {
                this.hideModal();
            }
        });
    }

    async loadData() {
        await Promise.all([
            this.loadProfile(),
            this.loadSubscriptionInfo(),
            this.loadSettings(),
            this.loadUsageStats()
        ]);
    }

    async loadProfile() {
        const profileElement = document.getElementById('profile-info');
        
        if (this.user) {
            const joinDate = new Date(this.user.created_at || Date.now()).toLocaleDateString();
            
            profileElement.innerHTML = `
                <div class="profile-card fade-in">
                    <img class="profile-avatar" src="${this.user.avatar}" alt="Profile Avatar" width="64" height="64">
                    <div class="profile-details">
                        <h3>${this.escapeHtml(this.user.name)}</h3>
                        <p><strong>Email:</strong> ${this.escapeHtml(this.user.email)}</p>
                        <p><strong>Joined:</strong> ${joinDate}</p>
                    </div>
                </div>
            `;
        } else {
            profileElement.innerHTML = '<div class="error">Failed to load profile information.</div>';
        }
    }

    async loadSubscriptionInfo() {
        const subscriptionElement = document.getElementById('subscription-details');
        
        try {
            const response = await fetch('/api/subscription');
            if (response.ok) {
                this.subscriptionInfo = await response.json();
                this.renderSubscriptionInfo();
            } else {
                throw new Error('Failed to load subscription info');
            }
        } catch (error) {
            console.error('Error loading subscription info:', error);
            subscriptionElement.innerHTML = '<div class="error">Failed to load subscription information.</div>';
        }
    }

    renderSubscriptionInfo() {
        const subscriptionElement = document.getElementById('subscription-details');
        const info = this.subscriptionInfo;
        
        if (!info) {
            subscriptionElement.innerHTML = '<div class="error">No subscription information available.</div>';
            return;
        }

        let statusClass = '';
        let statusText = '';
        let detailsHTML = '';
        let actionsHTML = '';

        if (info.status === 'unlimited') {
            // When subscription system is disabled
            statusClass = 'unlimited';
            statusText = 'Unlimited Access';
            detailsHTML = `
                <p class="subscription-details-text">
                    You have unlimited access to all features. The subscription system is currently disabled.
                </p>
                <p class="subscription-details-text">
                    <strong>Current feeds:</strong> ${info.current_feeds} feeds (no limit)
                </p>
            `;
            actionsHTML = '';
        } else if (info.status === 'admin') {
            // Admin users with subscription
            statusClass = 'admin';
            statusText = 'Admin + GoRead2 Pro';
            detailsHTML = `
                <p class="subscription-details-text">
                    You have admin privileges and an active Pro subscription with unlimited feeds.
                </p>
                <p class="subscription-details-text">
                    <strong>Last payment:</strong> ${this.formatDate(info.last_payment_date)}
                </p>
                <p class="subscription-details-text">
                    <strong>Current feeds:</strong> ${info.current_feeds} feeds (unlimited)
                </p>
            `;
            actionsHTML = `
                <button class="btn btn-primary" onclick="accountApp.manageSubscription()">
                    Manage Subscription
                </button>
                <button class="btn btn-secondary" onclick="accountApp.downloadInvoices()">
                    View Billing History
                </button>
            `;
        } else if (info.status === 'admin_trial') {
            // Admin users without subscription
            statusClass = 'admin';
            statusText = 'Admin (Unlimited)';
            detailsHTML = `
                <p class="subscription-details-text">
                    You have admin privileges with unlimited access to all features.
                </p>
                <p class="subscription-details-text">
                    You can optionally subscribe to GoRead2 Pro to support the service.
                </p>
                <p class="subscription-details-text">
                    <strong>Current feeds:</strong> ${info.current_feeds} feeds (unlimited)
                </p>
            `;
            actionsHTML = `
                <button class="btn btn-primary" onclick="accountApp.startSubscription()">
                    Subscribe to GoRead2 Pro
                </button>
            `;
        } else if (info.status === 'active') {
            statusClass = 'pro';
            statusText = 'GoRead2 Pro';
            detailsHTML = `
                <p class="subscription-details-text">
                    You have an active Pro subscription with unlimited feeds.
                </p>
                <p class="subscription-details-text">
                    <strong>Next billing date:</strong> ${this.formatDate(info.next_billing_date)}
                </p>
            `;
            actionsHTML = `
                <button class="btn btn-primary" onclick="accountApp.manageSubscription()">
                    Manage Subscription
                </button>
                <button class="btn btn-secondary" onclick="accountApp.downloadInvoices()">
                    View Billing History
                </button>
            `;
        } else if (info.status === 'trial') {
            // Check if trial is expired
            if (new Date(info.trial_ends_at) < new Date()) {
                statusClass = 'expired';
                statusText = 'Trial Expired';
                detailsHTML = `
                    <p class="subscription-details-text">
                        Your free trial has expired. Subscribe to GoRead2 Pro to continue using the service.
                    </p>
                    <p class="subscription-details-text">
                        <strong>Trial ended:</strong> ${this.formatDate(info.trial_ends_at)}
                    </p>
                `;
                actionsHTML = `
                    <button class="btn btn-primary" onclick="accountApp.upgradeSubscription()">
                        Subscribe to Pro ($2.99/month)
                    </button>
                `;
            } else {
                statusClass = 'trial';
                statusText = 'Free Trial';
                const daysLeft = info.trial_days_remaining || 0;
                detailsHTML = `
                    <p class="subscription-details-text">
                        You're currently on a free trial with access to up to ${info.feed_limit} feeds.
                    </p>
                    <p class="subscription-details-text">
                        <strong>Trial ends:</strong> ${this.formatDate(info.trial_ends_at)} (${daysLeft} days remaining)
                    </p>
                    <p class="subscription-details-text">
                        <strong>Current usage:</strong> ${info.current_feeds} of ${info.feed_limit} feeds
                    </p>
                `;
                actionsHTML = `
                    <button class="btn btn-primary" onclick="accountApp.upgradeSubscription()">
                        Upgrade to Pro ($2.99/month)
                    </button>
                `;
            }
        } else {
            // Handle unknown status - don't assume expired
            statusClass = 'unknown';
            statusText = 'Status: ' + (info.status || 'Unknown');
            detailsHTML = `
                <p class="subscription-details-text">
                    <strong>Account Status:</strong> ${info.status || 'Unknown'}
                </p>
                <p class="subscription-details-text">
                    <strong>Current feeds:</strong> ${info.current_feeds} feeds
                    ${info.feed_limit > 0 ? ` (limit: ${info.feed_limit})` : ' (unlimited)'}
                </p>
            `;
            actionsHTML = info.feed_limit > 0 ? `
                <button class="btn btn-primary" onclick="accountApp.upgradeSubscription()">
                    Subscribe to Pro ($2.99/month)
                </button>
            ` : '';
        }

        subscriptionElement.innerHTML = `
            <div class="subscription-info ${statusClass} fade-in">
                <div class="subscription-status-large">
                    <span class="status-badge-large">${statusText}</span>
                    <div class="subscription-meta">
                        <div class="status-text">
                            ${info.status === 'active' || info.status === 'admin' || info.status === 'admin_trial' ? 'Unlimited feeds' : 
                              info.status === 'unlimited' ? 'Unlimited feeds' :
                              info.status === 'trial' ? `${info.current_feeds}/${info.feed_limit} feeds` : 
                              'Subscribe to continue'}
                        </div>
                    </div>
                </div>
                ${detailsHTML}
                <div class="subscription-actions">
                    ${actionsHTML}
                </div>
            </div>
        `;
    }

    async loadSettings() {
        const settingsElement = document.getElementById('settings-container');

        if (this.user) {
            const maxArticles = this.user.max_articles_on_feed_add || 100;

            settingsElement.innerHTML = `
                <div class="settings-card fade-in">
                    <div class="setting-item">
                        <label for="max-articles-input">
                            <strong>Maximum articles to import when adding a new feed:</strong>
                            <div class="setting-description">
                                Limit the number of articles imported when subscribing to a new feed.
                                Set to 0 for unlimited. Higher numbers may slow down feed addition.
                            </div>
                        </label>
                        <div class="setting-control">
                            <input type="number"
                                   id="max-articles-input"
                                   value="${maxArticles}"
                                   min="0"
                                   max="10000"
                                   step="1">
                            <button id="save-max-articles" class="btn btn-primary">Save</button>
                        </div>
                    </div>
                </div>
            `;

            // Bind save button event
            const saveButton = document.getElementById('save-max-articles');
            if (saveButton) {
                saveButton.addEventListener('click', () => this.saveMaxArticlesSetting());
            }
        } else {
            settingsElement.innerHTML = '<div class="error">Failed to load settings.</div>';
        }
    }

    async saveMaxArticlesSetting() {
        const input = document.getElementById('max-articles-input');
        const saveButton = document.getElementById('save-max-articles');

        if (!input || !saveButton) return;

        const maxArticles = parseInt(input.value, 10);

        if (isNaN(maxArticles) || maxArticles < 0 || maxArticles > 10000) {
            alert('Please enter a valid number between 0 and 10000');
            return;
        }

        try {
            saveButton.disabled = true;
            saveButton.textContent = 'Saving...';

            const response = await fetch('/api/account/max-articles', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ max_articles: maxArticles })
            });

            if (response.ok) {
                const result = await response.json();
                // Update the user object
                this.user.max_articles_on_feed_add = maxArticles;

                // Show success message
                saveButton.textContent = 'Saved!';
                setTimeout(() => {
                    saveButton.textContent = 'Save';
                    saveButton.disabled = false;
                }, 2000);
            } else {
                const error = await response.json();
                throw new Error(error.error || 'Failed to save setting');
            }
        } catch (error) {
            console.error('Error saving max articles setting:', error);
            alert('Failed to save setting. Please try again.');
            saveButton.textContent = 'Save';
            saveButton.disabled = false;
        }
    }

    async loadUsageStats() {
        const statsElement = document.getElementById('usage-stats');
        
        try {
            // Load comprehensive account stats
            const response = await fetch('/api/account/stats');
            if (response.ok) {
                const stats = await response.json();
                this.renderUsageStats(stats);
            } else {
                throw new Error('Failed to load account stats');
            }
        } catch (error) {
            console.error('Error loading usage stats:', error);
            statsElement.innerHTML = '<div class="error">Failed to load usage statistics.</div>';
        }
    }

    renderUsageStats(stats) {
        const statsElement = document.getElementById('usage-stats');
        
        const totalFeeds = stats.total_feeds || 0;
        const totalUnread = stats.total_unread || 0;
        const activeFeedsCount = stats.active_feeds || 0;
        const totalArticles = stats.total_articles || 0;

        const statsHTML = `
            <div class="stats-grid fade-in">
                <div class="stat-card">
                    <div class="stat-number">${totalFeeds}</div>
                    <div class="stat-label">Total Feeds</div>
                    <div class="stat-description">RSS feeds subscribed</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">${totalUnread}</div>
                    <div class="stat-label">Unread Articles</div>
                    <div class="stat-description">Across all feeds</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">${activeFeedsCount}</div>
                    <div class="stat-label">Active Feeds</div>
                    <div class="stat-description">With unread content</div>
                </div>
                <div class="stat-card">
                    <div class="stat-number">${totalArticles}</div>
                    <div class="stat-label">Total Articles</div>
                    <div class="stat-description">Across all feeds</div>
                </div>
            </div>
        `;

        let feedListHTML = '';
        const feeds = stats.feeds || [];
        if (feeds.length > 0) {
            const feedsToShow = feeds.slice(0, 10); // Show first 10 feeds
            feedListHTML = `
                <div class="feed-summary">
                    <div class="feed-summary-header">
                        <h3>Your Feeds</h3>
                        <span class="feed-count">${feeds.length} total</span>
                    </div>
                    ${feedsToShow.map(feed => `
                        <div class="feed-item-account">
                            <div class="feed-info">
                                <div class="feed-name">${this.escapeHtml(feed.title)}</div>
                                <div class="feed-url">${this.escapeHtml(feed.url)}</div>
                            </div>
                            <div class="feed-articles">
                                Added ${this.formatDate(feed.created_at)}
                            </div>
                        </div>
                    `).join('')}
                    ${feeds.length > 10 ? `
                        <div class="feed-item-account">
                            <div class="feed-info">
                                <div class="feed-name">... and ${feeds.length - 10} more feeds</div>
                            </div>
                        </div>
                    ` : ''}
                </div>
            `;
        }

        statsElement.innerHTML = statsHTML + feedListHTML;
    }

    async startSubscription() {
        // Alias for upgradeSubscription - used for admin accounts who want to subscribe
        return await this.upgradeSubscription();
    }

    async upgradeSubscription() {
        try {
            const response = await fetch('/api/subscription/checkout', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    success_url: `${window.location.origin}/account?upgraded=true`,
                    cancel_url: `${window.location.origin}/account`
                })
            });

            if (response.ok) {
                const data = await response.json();
                window.location.href = data.session_url;
            } else {
                const error = await response.json();
                throw new Error(error.error || 'Failed to create checkout session');
            }
        } catch (error) {
            console.error('Error starting upgrade process:', error);
            alert('Failed to start upgrade process. Please try again.');
        }
    }

    async manageSubscription() {
        try {
            const response = await fetch('/api/subscription/portal', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    return_url: `${window.location.origin}/account`
                })
            });

            if (response.ok) {
                const data = await response.json();
                window.location.href = data.portal_url;
            } else {
                const error = await response.json();
                throw new Error(error.error || 'Failed to create customer portal session');
            }
        } catch (error) {
            console.error('Error opening customer portal:', error);
            alert('Failed to open subscription management. Please try again.');
        }
    }

    async downloadInvoices() {
        // This opens the customer portal which includes billing history
        await this.manageSubscription();
    }

    showModal(title, message, confirmAction) {
        const modal = document.getElementById('confirm-modal');
        const titleElement = document.getElementById('confirm-title');
        const messageElement = document.getElementById('confirm-message');
        const confirmButton = document.getElementById('confirm-action');

        titleElement.textContent = title;
        messageElement.textContent = message;
        
        // Remove previous event listeners and add new one
        const newConfirmButton = confirmButton.cloneNode(true);
        confirmButton.parentNode.replaceChild(newConfirmButton, confirmButton);
        
        newConfirmButton.addEventListener('click', () => {
            this.hideModal();
            confirmAction();
        });

        modal.style.display = 'block';
    }

    hideModal() {
        const modal = document.getElementById('confirm-modal');
        modal.style.display = 'none';
    }

    formatDate(dateString) {
        if (!dateString) return 'N/A';

        try {
            const date = new Date(dateString);
            // Check if the date is invalid or a zero/very old date (before year 1900)
            if (isNaN(date.getTime()) || date.getFullYear() < 1900) {
                return 'N/A';
            }
            return date.toLocaleDateString('en-US', {
                year: 'numeric',
                month: 'long',
                day: 'numeric'
            });
        } catch (error) {
            return 'N/A';
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize the app when DOM is loaded
let accountApp;
document.addEventListener('DOMContentLoaded', () => {
    accountApp = new AccountApp();
});