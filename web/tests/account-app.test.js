const { 
    waitFor, 
    waitForElement, 
    fireEvent, 
    createMockResponse,
    triggerDOMContentLoaded,
    expectElementToHaveClass,
    expectElementToBeVisible,
    expectElementToBeHidden 
} = require('./utils.js');

// Load the AccountApp class
const fs = require('fs');
const path = require('path');
const accountJsPath = path.join(__dirname, '../static/js/account.js');
const accountJsContent = fs.readFileSync(accountJsPath, 'utf8');

describe('AccountApp', () => {
    let app;

    beforeEach(async () => {
        // Create account page specific DOM structure
        document.body.innerHTML = `
            <div class="container">
                <div id="profile-info"></div>
                <div id="subscription-details"></div>
                <div id="usage-stats"></div>
                
                <!-- Confirmation Modal -->
                <div id="confirm-modal" class="modal" style="display: none;">
                    <div class="modal-content">
                        <span class="close">&times;</span>
                        <h2 id="confirm-title">Confirm Action</h2>
                        <p id="confirm-message">Are you sure?</p>
                        <div class="form-actions">
                            <button id="confirm-action" class="btn btn-primary">Confirm</button>
                            <button id="cancel-action" class="btn btn-secondary">Cancel</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        // Execute the account.js content to define AccountApp class
        eval(accountJsContent);
        
        // Mock successful authentication and API responses
        mockFetch({
            '/auth/me': createMockResponse({ 
                user: { 
                    id: 1, 
                    name: 'Test User', 
                    email: 'test@example.com', 
                    avatar: 'https://example.com/avatar.jpg',
                    created_at: '2024-01-01T00:00:00Z'
                } 
            }),
            '/api/subscription': createMockResponse({ 
                status: 'trial', 
                current_feeds: 5, 
                feed_limit: 20, 
                trial_days_remaining: 15,
                trial_ends_at: '2024-12-31T23:59:59Z'
            }),
            '/api/account/stats': createMockResponse({
                total_feeds: 5,
                total_articles: 150,
                total_unread: 25,
                active_feeds: 3,
                subscription_info: {
                    status: 'trial',
                    current_feeds: 5,
                    feed_limit: 20
                },
                feeds: [
                    {
                        id: 1,
                        title: 'Tech News',
                        url: 'https://example.com/tech',
                        created_at: '2024-01-01T00:00:00Z'
                    },
                    {
                        id: 2,
                        title: 'Science Blog',
                        url: 'https://example.com/science',
                        created_at: '2024-01-02T00:00:00Z'
                    }
                ]
            })
        });
    });

    afterEach(() => {
        if (app) {
            app = null;
        }
    });

    describe('Initialization', () => {
        test('should initialize with authenticated user', async () => {
            app = new AccountApp();
            
            await waitFor(() => app.user !== null);
            
            expect(app.user).toEqual({
                id: 1,
                name: 'Test User',
                email: 'test@example.com',
                avatar: 'https://example.com/avatar.jpg',
                created_at: '2024-01-01T00:00:00Z'
            });
        });

        test('should redirect unauthenticated user to home', async () => {
            mockFetch({
                '/auth/me': createMockResponse(null, { status: 401, ok: false })
            });
            
            const originalLocation = window.location.href;
            
            app = new AccountApp();
            
            await waitFor(() => window.location.href !== originalLocation);
            
            expect(window.location.href).toBe('/');
        });

        test('should load all data on init', async () => {
            app = new AccountApp();
            
            await waitFor(() => 
                app.user !== null && 
                app.subscriptionInfo !== null
            );
            
            expect(app.subscriptionInfo).toBeTruthy();
            expect(document.getElementById('profile-info').innerHTML).toContain('Test User');
            expect(document.getElementById('subscription-details').innerHTML).toContain('Free Trial');
            expect(document.getElementById('usage-stats').innerHTML).toContain('5');
        });
    });

    describe('Profile Management', () => {
        beforeEach(async () => {
            app = new AccountApp();
            await waitFor(() => app.user !== null);
        });

        test('should render profile information', () => {
            const profileElement = document.getElementById('profile-info');
            
            expect(profileElement.innerHTML).toContain('Test User');
            expect(profileElement.innerHTML).toContain('test@example.com');
            expect(profileElement.innerHTML).toContain('https://example.com/avatar.jpg');
            expect(profileElement.innerHTML).toContain('January 1, 2024');
        });

        test('should handle missing user data gracefully', async () => {
            app.user = null;
            
            await app.loadProfile();
            
            const profileElement = document.getElementById('profile-info');
            expect(profileElement.innerHTML).toContain('Failed to load profile information');
        });
    });

    describe('Subscription Information', () => {
        beforeEach(async () => {
            app = new AccountApp();
            await waitFor(() => app.user !== null);
        });

        test('should render trial subscription info', () => {
            const subscriptionElement = document.getElementById('subscription-details');
            
            expect(subscriptionElement.innerHTML).toContain('Free Trial');
            expect(subscriptionElement.innerHTML).toContain('5 of 20 feeds');
            expect(subscriptionElement.innerHTML).toContain('15 days remaining');
            expect(subscriptionElement.innerHTML).toContain('Upgrade to Pro');
        });

        test('should render active subscription info', async () => {
            mockFetch({
                '/api/subscription': createMockResponse({ 
                    status: 'active',
                    next_billing_date: '2024-02-01T00:00:00Z'
                })
            });
            
            await app.loadSubscriptionInfo();
            
            const subscriptionElement = document.getElementById('subscription-details');
            expect(subscriptionElement.innerHTML).toContain('GoRead2 Pro');
            expect(subscriptionElement.innerHTML).toContain('Unlimited feeds');
            expect(subscriptionElement.innerHTML).toContain('Manage Subscription');
        });

        test('should render expired subscription info', async () => {
            mockFetch({
                '/api/subscription': createMockResponse({ 
                    status: 'expired',
                    trial_ends_at: '2024-01-01T00:00:00Z'
                })
            });
            
            await app.loadSubscriptionInfo();
            
            const subscriptionElement = document.getElementById('subscription-details');
            expect(subscriptionElement.innerHTML).toContain('Trial Expired');
            expect(subscriptionElement.innerHTML).toContain('Subscribe to Pro');
        });

        test('should handle subscription info load error', async () => {
            mockFetch({
                '/api/subscription': createMockResponse(
                    { error: 'Server error' },
                    { status: 500, ok: false }
                )
            });
            
            await app.loadSubscriptionInfo();
            
            const subscriptionElement = document.getElementById('subscription-details');
            expect(subscriptionElement.innerHTML).toContain('Failed to load subscription information');
        });
    });

    describe('Usage Statistics', () => {
        beforeEach(async () => {
            app = new AccountApp();
            await waitFor(() => app.user !== null);
        });

        test('should render usage statistics', () => {
            const statsElement = document.getElementById('usage-stats');
            
            expect(statsElement.innerHTML).toContain('5'); // total feeds
            expect(statsElement.innerHTML).toContain('150'); // total articles
            expect(statsElement.innerHTML).toContain('25'); // unread articles
            expect(statsElement.innerHTML).toContain('3'); // active feeds
        });

        test('should render feed list', () => {
            const statsElement = document.getElementById('usage-stats');
            
            expect(statsElement.innerHTML).toContain('Tech News');
            expect(statsElement.innerHTML).toContain('Science Blog');
            expect(statsElement.innerHTML).toContain('https://example.com/tech');
            expect(statsElement.innerHTML).toContain('2 total');
        });

        test('should limit displayed feeds to 10', async () => {
            const manyFeeds = Array.from({ length: 15 }, (_, i) => ({
                id: i + 1,
                title: `Feed ${i + 1}`,
                url: `https://example.com/feed${i + 1}`,
                created_at: '2024-01-01T00:00:00Z'
            }));

            mockFetch({
                '/api/account/stats': createMockResponse({
                    total_feeds: 15,
                    feeds: manyFeeds
                })
            });
            
            await app.loadUsageStats();
            
            const statsElement = document.getElementById('usage-stats');
            expect(statsElement.innerHTML).toContain('15 total');
            expect(statsElement.innerHTML).toContain('... and 5 more feeds');
        });

        test('should handle stats load error', async () => {
            mockFetch({
                '/api/account/stats': createMockResponse(
                    { error: 'Server error' },
                    { status: 500, ok: false }
                )
            });
            
            await app.loadUsageStats();
            
            const statsElement = document.getElementById('usage-stats');
            expect(statsElement.innerHTML).toContain('Failed to load usage statistics');
        });
    });

    describe('Subscription Actions', () => {
        beforeEach(async () => {
            app = new AccountApp();
            await waitFor(() => app.user !== null);
        });

        test('should start upgrade process', async () => {
            mockFetch({
                '/api/subscription/checkout': createMockResponse({
                    session_url: 'https://checkout.stripe.com/123'
                })
            });
            
            await app.upgradeSubscription();
            
            expect(fetch).toHaveBeenCalledWith('/api/subscription/checkout', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    success_url: 'http://localhost:3000/account?upgraded=true',
                    cancel_url: 'http://localhost:3000/account'
                })
            });
            
            expect(window.location.href).toBe('https://checkout.stripe.com/123');
        });

        test('should handle upgrade process error', async () => {
            mockFetch({
                '/api/subscription/checkout': createMockResponse(
                    { error: 'Payment error' },
                    { status: 400, ok: false }
                )
            });
            
            window.alert = jest.fn();
            
            await app.upgradeSubscription();
            
            expect(window.alert).toHaveBeenCalledWith('Failed to start upgrade process. Please try again.');
        });

        test('should open customer portal', async () => {
            mockFetch({
                '/api/subscription/portal': createMockResponse({
                    portal_url: 'https://billing.stripe.com/p/session_123'
                })
            });
            
            await app.manageSubscription();
            
            expect(fetch).toHaveBeenCalledWith('/api/subscription/portal', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    return_url: 'http://localhost:3000/account'
                })
            });
            
            expect(window.location.href).toBe('https://billing.stripe.com/p/session_123');
        });

        test('should handle portal error', async () => {
            mockFetch({
                '/api/subscription/portal': createMockResponse(
                    { error: 'Portal error' },
                    { status: 400, ok: false }
                )
            });
            
            window.alert = jest.fn();
            
            await app.manageSubscription();
            
            expect(window.alert).toHaveBeenCalledWith('Failed to open subscription management. Please try again.');
        });
    });

    describe('Modal Management', () => {
        beforeEach(async () => {
            app = new AccountApp();
            await waitFor(() => app.user !== null);
        });

        test('should show confirmation modal', () => {
            const confirmAction = jest.fn();
            
            app.showModal('Test Title', 'Test message', confirmAction);
            
            const modal = document.getElementById('confirm-modal');
            const title = document.getElementById('confirm-title');
            const message = document.getElementById('confirm-message');
            
            expectElementToBeVisible(modal);
            expect(title.textContent).toBe('Test Title');
            expect(message.textContent).toBe('Test message');
        });

        test('should hide modal', () => {
            app.showModal('Test', 'Test', jest.fn());
            
            app.hideModal();
            
            const modal = document.getElementById('confirm-modal');
            expectElementToBeHidden(modal);
        });

        test('should execute confirm action and hide modal', () => {
            const confirmAction = jest.fn();
            
            app.showModal('Test', 'Test', confirmAction);
            
            const confirmBtn = document.getElementById('confirm-action');
            fireEvent.click(confirmBtn);
            
            expect(confirmAction).toHaveBeenCalled();
            
            const modal = document.getElementById('confirm-modal');
            expectElementToBeHidden(modal);
        });

        test('should cancel modal with cancel button', () => {
            const confirmAction = jest.fn();
            
            app.showModal('Test', 'Test', confirmAction);
            
            const cancelBtn = document.getElementById('cancel-action');
            fireEvent.click(cancelBtn);
            
            expect(confirmAction).not.toHaveBeenCalled();
            
            const modal = document.getElementById('confirm-modal');
            expectElementToBeHidden(modal);
        });

        test('should close modal when clicking outside', () => {
            const confirmAction = jest.fn();
            
            app.showModal('Test', 'Test', confirmAction);
            
            const modal = document.getElementById('confirm-modal');
            fireEvent.click(modal);
            
            expect(confirmAction).not.toHaveBeenCalled();
            expectElementToBeHidden(modal);
        });
    });

    describe('Utility Functions', () => {
        beforeEach(async () => {
            app = new AccountApp();
            await waitFor(() => app.user !== null);
        });

        test('should format dates correctly', () => {
            const formatted = app.formatDate('2024-01-15T10:30:00Z');
            expect(formatted).toBe('January 15, 2024');
        });

        test('should handle invalid dates', () => {
            const formatted = app.formatDate('invalid-date');
            expect(formatted).toBe('Invalid date');
        });

        test('should handle null dates', () => {
            const formatted = app.formatDate(null);
            expect(formatted).toBe('N/A');
        });

        test('should escape HTML properly', () => {
            const escaped = app.escapeHtml('<script>alert("xss")</script>');
            expect(escaped).toBe('&lt;script&gt;alert("xss")&lt;/script&gt;');
        });

        test('should handle empty string escaping', () => {
            const escaped = app.escapeHtml('');
            expect(escaped).toBe('');
        });
    });

    describe('Event Binding', () => {
        beforeEach(async () => {
            app = new AccountApp();
            await waitFor(() => app.user !== null);
        });

        test('should bind close button event', () => {
            const closeBtn = document.querySelector('.close');
            const hideModalSpy = jest.spyOn(app, 'hideModal');
            
            fireEvent.click(closeBtn);
            
            expect(hideModalSpy).toHaveBeenCalled();
        });

        test('should bind cancel button event', () => {
            const cancelBtn = document.getElementById('cancel-action');
            const hideModalSpy = jest.spyOn(app, 'hideModal');
            
            fireEvent.click(cancelBtn);
            
            expect(hideModalSpy).toHaveBeenCalled();
        });
    });

    describe('Error Handling', () => {
        beforeEach(async () => {
            app = new AccountApp();
            await waitFor(() => app.user !== null);
        });

        test('should handle network errors gracefully', async () => {
            global.fetch.mockRejectedValueOnce(new Error('Network error'));
            
            await app.loadSubscriptionInfo();
            
            const subscriptionElement = document.getElementById('subscription-details');
            expect(subscriptionElement.innerHTML).toContain('Failed to load subscription information');
        });

        test('should handle malformed API responses', async () => {
            mockFetch({
                '/api/subscription': {
                    ok: true,
                    json: () => Promise.reject(new Error('Invalid JSON'))
                }
            });
            
            await app.loadSubscriptionInfo();
            
            const subscriptionElement = document.getElementById('subscription-details');
            expect(subscriptionElement.innerHTML).toContain('Failed to load subscription information');
        });
    });
});