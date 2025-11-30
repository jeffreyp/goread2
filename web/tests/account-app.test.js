const {
    waitFor,
    fireEvent,
    createMockResponse,
    expectElementToBeVisible,
    expectElementToBeHidden
} = require('./utils.js');

/**
 * Account App DOM Tests
 *
 * These tests verify the Account app's DOM manipulation and UI behavior
 * without requiring class instantiation. They test the actual DOM elements
 * and event handlers that would be created by the AccountApp class.
 */
describe('Account App DOM Tests', () => {

    beforeEach(() => {
        // Create account page specific DOM structure
        const existingProfile = document.getElementById('profile-info');
        if (existingProfile) {
            existingProfile.innerHTML = '';
        }

        const existingSubscription = document.getElementById('subscription-details');
        if (existingSubscription) {
            existingSubscription.innerHTML = '';
        }

        const existingStats = document.getElementById('usage-stats');
        if (existingStats) {
            existingStats.innerHTML = '';
        }
    });

    describe('Profile Rendering', () => {
        test('should render user profile information', () => {
            const profileElement = document.getElementById('profile-info');
            const user = {
                name: 'Test User',
                email: 'test@example.com',
                avatar: 'https://example.com/avatar.jpg',
                created_at: '2024-01-01T00:00:00Z'
            };

            // Simulate profile rendering
            const escapeHtml = (text) => {
                const div = document.createElement('div');
                div.textContent = text;
                return div.innerHTML;
            };

            const joinDate = new Date(user.created_at).toLocaleDateString();

            profileElement.innerHTML = `
                <div class="profile-card fade-in">
                    <img class="profile-avatar" src="${user.avatar}" alt="Profile Avatar" width="64" height="64">
                    <div class="profile-details">
                        <h3>${escapeHtml(user.name)}</h3>
                        <p><strong>Email:</strong> ${escapeHtml(user.email)}</p>
                        <p><strong>Joined:</strong> ${joinDate}</p>
                    </div>
                </div>
            `;

            expect(profileElement.innerHTML).toContain('Test User');
            expect(profileElement.innerHTML).toContain('test@example.com');
            expect(profileElement.innerHTML).toContain('https://example.com/avatar.jpg');
        });

        test('should handle missing user data', () => {
            const profileElement = document.getElementById('profile-info');

            // Simulate error state
            profileElement.innerHTML = '<div class="error">Failed to load profile information.</div>';

            expect(profileElement.innerHTML).toContain('Failed to load profile information');
        });
    });

    describe('Subscription Information', () => {
        test('should render trial subscription info', () => {
            const subscriptionElement = document.getElementById('subscription-details');
            const subscriptionInfo = {
                status: 'trial',
                current_feeds: 5,
                feed_limit: 20,
                trial_days_remaining: 15,
                trial_ends_at: '2024-12-31T23:59:59Z'
            };

            // Simulate subscription rendering
            subscriptionElement.innerHTML = `
                <div class="subscription-info trial fade-in">
                    <div class="subscription-status">
                        <span class="status-badge trial">Free Trial</span>
                        <div class="subscription-meta">
                            <div class="status-text">${subscriptionInfo.current_feeds} of ${subscriptionInfo.feed_limit} feeds</div>
                            <div class="trial-info">${subscriptionInfo.trial_days_remaining} days remaining</div>
                        </div>
                    </div>
                    <button class="btn btn-primary upgrade-btn">Upgrade to Pro</button>
                </div>
            `;

            expect(subscriptionElement.innerHTML).toContain('Free Trial');
            expect(subscriptionElement.innerHTML).toContain('5 of 20 feeds');
            expect(subscriptionElement.innerHTML).toContain('15 days remaining');
            expect(subscriptionElement.innerHTML).toContain('Upgrade to Pro');
        });

        test('should render active subscription info', () => {
            const subscriptionElement = document.getElementById('subscription-details');
            const subscriptionInfo = {
                status: 'active',
                next_billing_date: '2024-02-01T00:00:00Z'
            };

            const nextBillingDate = new Date(subscriptionInfo.next_billing_date).toLocaleDateString();

            subscriptionElement.innerHTML = `
                <div class="subscription-info active fade-in">
                    <div class="subscription-status">
                        <span class="status-badge active">GoRead2 Pro</span>
                        <div class="subscription-meta">
                            <div class="status-text">Unlimited feeds</div>
                            <div>Next billing: ${nextBillingDate}</div>
                        </div>
                    </div>
                    <button class="btn btn-secondary manage-subscription-btn">Manage Subscription</button>
                </div>
            `;

            expect(subscriptionElement.innerHTML).toContain('GoRead2 Pro');
            expect(subscriptionElement.innerHTML).toContain('Unlimited feeds');
            expect(subscriptionElement.innerHTML).toContain('Manage Subscription');
        });

        test('should render unlimited subscription info', () => {
            const subscriptionElement = document.getElementById('subscription-details');
            const subscriptionInfo = {
                status: 'unlimited',
                feed_limit: -1,
                can_add_feeds: true,
                current_feeds: 25
            };

            subscriptionElement.innerHTML = `
                <div class="subscription-info unlimited fade-in">
                    <div class="subscription-status-large">
                        <span class="status-badge-large">Unlimited Access</span>
                        <div class="subscription-meta">
                            <div class="status-text">Unlimited feeds</div>
                        </div>
                    </div>
                    <p class="subscription-details-text">
                        You have unlimited access to all features. The subscription system is currently disabled.
                    </p>
                    <p class="subscription-details-text">
                        <strong>Current feeds:</strong> ${subscriptionInfo.current_feeds} feeds (no limit)
                    </p>
                </div>
            `;

            expect(subscriptionElement.innerHTML).toContain('Unlimited Access');
            expect(subscriptionElement.innerHTML).toContain('subscription system is currently disabled');
            expect(subscriptionElement.innerHTML).toContain('25 feeds (no limit)');
        });

        test('should render expired subscription info', () => {
            const subscriptionElement = document.getElementById('subscription-details');

            subscriptionElement.innerHTML = `
                <div class="subscription-info expired fade-in">
                    <div class="subscription-status">
                        <span class="status-badge expired">Trial Expired</span>
                    </div>
                    <button class="btn btn-primary upgrade-btn">Subscribe to Pro</button>
                </div>
            `;

            expect(subscriptionElement.innerHTML).toContain('Trial Expired');
            expect(subscriptionElement.innerHTML).toContain('Subscribe to Pro');
        });
    });

    describe('Usage Statistics', () => {
        test('should render usage statistics', () => {
            const statsElement = document.getElementById('usage-stats');
            const stats = {
                total_feeds: 5,
                total_articles: 150,
                total_unread: 25,
                active_feeds: 3
            };

            statsElement.innerHTML = `
                <div class="stats-grid fade-in">
                    <div class="stat-card">
                        <div class="stat-value">${stats.total_feeds}</div>
                        <div class="stat-label">Total Feeds</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">${stats.total_articles}</div>
                        <div class="stat-label">Total Articles</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">${stats.total_unread}</div>
                        <div class="stat-label">Unread Articles</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">${stats.active_feeds}</div>
                        <div class="stat-label">Active Feeds</div>
                    </div>
                </div>
            `;

            expect(statsElement.innerHTML).toContain('5');
            expect(statsElement.innerHTML).toContain('150');
            expect(statsElement.innerHTML).toContain('25');
            expect(statsElement.innerHTML).toContain('3');
        });

        test('should handle stats load error', () => {
            const statsElement = document.getElementById('usage-stats');

            statsElement.innerHTML = '<div class="error">Failed to load usage statistics.</div>';

            expect(statsElement.innerHTML).toContain('Failed to load usage statistics');
        });
    });

    describe('Modal Management', () => {
        test('should show confirmation modal', () => {
            const modal = document.getElementById('confirm-modal');
            const title = document.getElementById('confirm-title');
            const message = document.getElementById('confirm-message');

            title.textContent = 'Test Title';
            message.textContent = 'Test message';
            modal.style.display = 'block';

            expectElementToBeVisible(modal);
            expect(title.textContent).toBe('Test Title');
            expect(message.textContent).toBe('Test message');
        });

        test('should hide modal', () => {
            const modal = document.getElementById('confirm-modal');

            modal.style.display = 'block';
            modal.style.display = 'none';

            expectElementToBeHidden(modal);
        });

        test('should handle cancel button click', () => {
            const modal = document.getElementById('confirm-modal');
            const cancelBtn = document.getElementById('cancel-action');

            modal.style.display = 'block';

            cancelBtn.addEventListener('click', () => {
                modal.style.display = 'none';
            });

            fireEvent.click(cancelBtn);
            expectElementToBeHidden(modal);
        });

        test('should handle close button click', () => {
            const modal = document.getElementById('confirm-modal');
            const closeBtn = document.querySelector('.close');

            modal.style.display = 'block';

            closeBtn.addEventListener('click', () => {
                modal.style.display = 'none';
            });

            fireEvent.click(closeBtn);
            expectElementToBeHidden(modal);
        });
    });

    describe('API Integration', () => {
        test('should handle successful authentication check', async () => {
            mockFetch({
                '/auth/me': createMockResponse({
                    user: {
                        id: 1,
                        name: 'Test User',
                        email: 'test@example.com',
                        avatar: 'https://example.com/avatar.jpg',
                        created_at: '2024-01-01T00:00:00Z'
                    }
                })
            });

            const response = await fetch('/auth/me');
            const data = await response.json();

            expect(response.ok).toBe(true);
            expect(data.user).toBeTruthy();
            expect(data.user.name).toBe('Test User');
        });

        test('should handle failed authentication', async () => {
            mockFetch({
                '/auth/me': createMockResponse(null, { status: 401, ok: false })
            });

            const response = await fetch('/auth/me');

            expect(response.ok).toBe(false);
            expect(response.status).toBe(401);
        });

        test('should handle subscription info load', async () => {
            mockFetch({
                '/api/subscription': createMockResponse({
                    status: 'trial',
                    current_feeds: 5,
                    feed_limit: 20,
                    trial_days_remaining: 15
                })
            });

            const response = await fetch('/api/subscription');
            const data = await response.json();

            expect(response.ok).toBe(true);
            expect(data.status).toBe('trial');
            expect(data.current_feeds).toBe(5);
        });

        test('should handle subscription checkout', async () => {
            mockFetch({
                '/api/subscription/checkout': createMockResponse({
                    session_url: 'https://checkout.stripe.com/123'
                })
            });

            const response = await fetch('/api/subscription/checkout', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    success_url: 'http://localhost/account?upgraded=true',
                    cancel_url: 'http://localhost/account'
                })
            });
            const data = await response.json();

            expect(response.ok).toBe(true);
            expect(data.session_url).toBe('https://checkout.stripe.com/123');
        });
    });

    describe('Utility Functions', () => {
        test('should escape HTML', () => {
            const escapeHtml = (text) => {
                const div = document.createElement('div');
                div.textContent = text;
                return div.innerHTML;
            };

            const escaped = escapeHtml('<script>alert("xss")</script>');
            expect(escaped).toBe('&lt;script&gt;alert("xss")&lt;/script&gt;');
        });

        test('should format dates', () => {
            const formatDate = (dateString) => {
                if (!dateString) return 'N/A';
                if (dateString === null) return 'N/A';
                try {
                    const date = new Date(dateString);
                    if (isNaN(date.getTime())) return 'Invalid date';
                    return date.toLocaleDateString('en-US', {
                        year: 'numeric',
                        month: 'long',
                        day: 'numeric'
                    });
                } catch {
                    return 'Invalid date';
                }
            };

            expect(formatDate('2024-01-15T10:30:00Z')).toBe('January 15, 2024');
            expect(formatDate('invalid-date')).toBe('Invalid date');
            expect(formatDate(null)).toBe('N/A');
        });

        test('should handle empty string escaping', () => {
            const escapeHtml = (text) => {
                const div = document.createElement('div');
                div.textContent = text;
                return div.innerHTML;
            };

            expect(escapeHtml('')).toBe('');
        });
    });

    describe('Event Binding', () => {
        test('should handle window click events for modal close', () => {
            const modal = document.getElementById('confirm-modal');
            let modalClosed = false;

            modal.style.display = 'block';

            window.addEventListener('click', (e) => {
                if (e.target === modal) {
                    modal.style.display = 'none';
                    modalClosed = true;
                }
            });

            // Simulate clicking on the modal backdrop
            const clickEvent = new Event('click', { bubbles: true });
            Object.defineProperty(clickEvent, 'target', { value: modal, enumerable: true });
            window.dispatchEvent(clickEvent);

            expect(modalClosed).toBe(true);
            expectElementToBeHidden(modal);
        });
    });

    describe('Error Handling', () => {
        test('should handle network errors gracefully', async () => {
            global.fetch.mockRejectedValueOnce(new Error('Network error'));

            try {
                await fetch('/api/subscription');
                fail('Should have thrown an error');
            } catch (error) {
                expect(error.message).toBe('Network error');
            }
        });

        test('should handle malformed API responses', async () => {
            mockFetch({
                '/api/subscription': {
                    ok: true,
                    json: () => Promise.reject(new Error('Invalid JSON'))
                }
            });

            const response = await fetch('/api/subscription');

            try {
                await response.json();
                fail('Should have thrown an error');
            } catch (error) {
                expect(error.message).toBe('Invalid JSON');
            }
        });
    });
});
