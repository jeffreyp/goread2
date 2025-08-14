const { 
    waitFor, 
    waitForElement, 
    fireEvent, 
    createTestArticles, 
    createTestFeeds,
    createMockResponse,
    mockFormData,
    createMockFile,
    triggerDOMContentLoaded,
    expectElementToHaveClass,
    expectElementNotToHaveClass,
    expectElementToBeVisible,
    expectElementToBeHidden 
} = require('./utils.js');

// Load the GoReadApp class
const fs = require('fs');
const path = require('path');
const appJsPath = path.join(__dirname, '../static/js/app.js');
const appJsContent = fs.readFileSync(appJsPath, 'utf8');

describe('GoReadApp', () => {
    let app;

    beforeEach(async () => {
        // Execute the app.js content to define GoReadApp class
        eval(appJsContent);
        
        // Mock successful authentication
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
            '/api/feeds': createMockResponse(createTestFeeds()),
            '/api/feeds/unread-counts': createMockResponse({ '1': 5, '2': 3 }),
            '/api/feeds/all/articles': createMockResponse(createTestArticles())
        });
    });

    afterEach(() => {
        if (app) {
            app = null;
        }
    });

    describe('Initialization', () => {
        test('should initialize with authenticated user', async () => {
            app = new GoReadApp();
            
            await waitFor(() => app.user !== null);
            
            expect(app.user).toEqual({
                id: 1,
                name: 'Test User',
                email: 'test@example.com',
                avatar: 'https://example.com/avatar.jpg',
                created_at: '2024-01-01T00:00:00Z'
            });
            
            expect(document.getElementById('app').style.display).toBe('block');
        });

        test('should show login screen for unauthenticated user', async () => {
            mockFetch({
                '/auth/me': createMockResponse(null, { status: 401, ok: false })
            });
            
            app = new GoReadApp();
            
            await waitFor(() => document.getElementById('login-screen') !== null);
            
            expect(document.getElementById('login-screen')).toBeTruthy();
            expect(document.getElementById('app').style.display).toBe('none');
        });

        test('should load subscription info on init', async () => {
            app = new GoReadApp();
            
            await waitFor(() => app.subscriptionInfo !== null);
            
            expect(app.subscriptionInfo).toEqual({
                status: 'trial',
                current_feeds: 5,
                feed_limit: 20,
                trial_days_remaining: 15,
                trial_ends_at: '2024-12-31T23:59:59Z'
            });
        });

        test('should load feeds on init', async () => {
            app = new GoReadApp();
            
            await waitFor(() => app.feeds.length > 0);
            
            expect(app.feeds).toHaveLength(2);
            expect(app.feeds[0].title).toBe('Test Feed 1');
            expect(app.feeds[1].title).toBe('Test Feed 2');
        });
    });

    describe('Feed Management', () => {
        beforeEach(async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
        });

        test('should render feeds in the sidebar', async () => {
            await waitFor(() => app.feeds.length > 0);
            
            const feedItems = document.querySelectorAll('.feed-item:not(.special)');
            expect(feedItems).toHaveLength(2);
            
            expect(feedItems[0].querySelector('.feed-title').textContent).toBe('Test Feed 1');
            expect(feedItems[1].querySelector('.feed-title').textContent).toBe('Test Feed 2');
        });

        test('should select feed when clicked', async () => {
            await waitFor(() => app.feeds.length > 0);
            
            const feedItem = document.querySelector('[data-feed-id="1"]');
            fireEvent.click(feedItem);
            
            await waitFor(() => app.currentFeed === 1);
            
            expect(app.currentFeed).toBe(1);
            expectElementToHaveClass(feedItem, 'active');
        });

        test('should show add feed modal', () => {
            const addBtn = document.getElementById('add-feed-btn');
            fireEvent.click(addBtn);
            
            const modal = document.getElementById('add-feed-modal');
            expectElementToBeVisible(modal);
        });

        test('should hide add feed modal on cancel', () => {
            app.showAddFeedModal();
            
            const cancelBtn = document.getElementById('cancel-add-feed');
            fireEvent.click(cancelBtn);
            
            const modal = document.getElementById('add-feed-modal');
            expectElementToBeHidden(modal);
        });

        test('should add new feed successfully', async () => {
            mockFetch({
                ...global.fetch.mock.defaultResponses,
                '/api/feeds': createMockResponse({ id: 3, title: 'New Feed', url: 'https://example.com/new' })
            });
            
            app.showAddFeedModal();
            
            const urlInput = document.getElementById('feed-url');
            const form = document.getElementById('add-feed-form');
            
            urlInput.value = 'https://example.com/new';
            fireEvent.submit(form);
            
            await waitFor(() => document.getElementById('add-feed-modal').style.display === 'none');
            
            expect(fetch).toHaveBeenCalledWith('/api/feeds', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url: 'https://example.com/new' })
            });
        });

        test('should handle feed limit error', async () => {
            mockFetch({
                '/api/feeds': createMockResponse(
                    { error: 'Feed limit reached', limit_reached: true, current_limit: 20 },
                    { status: 402, ok: false }
                )
            });
            
            app.showAddFeedModal();
            
            const urlInput = document.getElementById('feed-url');
            const form = document.getElementById('add-feed-form');
            
            urlInput.value = 'https://example.com/new';
            fireEvent.submit(form);
            
            await waitFor(() => document.querySelector('.modal'));
            
            const limitModal = document.querySelector('.modal .modal-content h2');
            expect(limitModal.textContent).toBe('Upgrade to Pro');
        });

        test('should delete feed with confirmation', async () => {
            await waitFor(() => app.feeds.length > 0);
            
            window.confirm = jest.fn(() => true);
            mockFetch({
                '/api/feeds/1': createMockResponse({}, { status: 200 })
            });
            
            const deleteBtn = document.querySelector('[data-feed-id="1"] .delete-feed');
            fireEvent.click(deleteBtn);
            
            await waitFor(() => fetch.mock.calls.some(call => call[0] === '/api/feeds/1'));
            
            expect(fetch).toHaveBeenCalledWith('/api/feeds/1', { method: 'DELETE' });
        });
    });

    describe('Article Management', () => {
        beforeEach(async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null && app.feeds.length > 0);
        });

        test('should load articles for selected feed', async () => {
            mockFetch({
                '/api/feeds/1/articles': createMockResponse(createTestArticles())
            });
            
            await app.selectFeed(1);
            
            await waitFor(() => app.articles.length > 0);
            
            expect(app.articles).toHaveLength(3);
            expect(app.articles[0].title).toBe('Test Article 1');
        });

        test('should render articles in article list', async () => {
            app.articles = createTestArticles();
            app.renderArticles();
            
            const articleItems = document.querySelectorAll('.article-item');
            expect(articleItems).toHaveLength(3);
            
            expect(articleItems[0].querySelector('.article-title').textContent).toBe('Test Article 1');
        });

        test('should select article when clicked', async () => {
            app.articles = createTestArticles();
            app.renderArticles();
            
            const articleItem = document.querySelector('[data-index="1"]');
            fireEvent.click(articleItem);
            
            await waitFor(() => app.currentArticle === 1);
            
            expect(app.currentArticle).toBe(1);
            expectElementToHaveClass(articleItem, 'active');
        });

        test('should toggle article star status', async () => {
            mockFetch({
                '/api/articles/2/star': createMockResponse({})
            });
            
            app.articles = createTestArticles();
            app.renderArticles();
            
            const starBtn = document.querySelector('[data-article-id="2"] .star-btn');
            fireEvent.click(starBtn);
            
            await waitFor(() => fetch.mock.calls.some(call => call[0] === '/api/articles/2/star'));
            
            expect(fetch).toHaveBeenCalledWith('/api/articles/2/star', { method: 'POST' });
        });

        test('should mark article as read', async () => {
            mockFetch({
                '/api/articles/1/read': createMockResponse({})
            });
            
            await app.markAsRead(1, true);
            
            expect(fetch).toHaveBeenCalledWith('/api/articles/1/read', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ is_read: true })
            });
        });
    });

    describe('Keyboard Shortcuts', () => {
        beforeEach(async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
            app.articles = createTestArticles();
            app.renderArticles();
            app.currentArticle = 0;
        });

        test('should navigate to next article with j key', () => {
            const selectArticleSpy = jest.spyOn(app, 'selectArticle');
            
            fireEvent.keydown(document, 'j');
            
            expect(selectArticleSpy).toHaveBeenCalledWith(1);
        });

        test('should navigate to previous article with k key', () => {
            app.currentArticle = 1;
            const selectArticleSpy = jest.spyOn(app, 'selectArticle');
            
            fireEvent.keydown(document, 'k');
            
            expect(selectArticleSpy).toHaveBeenCalledWith(0);
        });

        test('should open current article with o key', () => {
            window.open = jest.fn();
            
            fireEvent.keydown(document, 'o');
            
            expect(window.open).toHaveBeenCalledWith('https://example.com/article1', '_blank');
        });

        test('should toggle read status with m key', async () => {
            const toggleSpy = jest.spyOn(app, 'toggleCurrentArticleRead');
            
            fireEvent.keydown(document, 'm');
            
            expect(toggleSpy).toHaveBeenCalled();
        });

        test('should toggle star status with s key', () => {
            const toggleSpy = jest.spyOn(app, 'toggleCurrentArticleStar');
            
            fireEvent.keydown(document, 's');
            
            expect(toggleSpy).toHaveBeenCalled();
        });

        test('should refresh feeds with r key', () => {
            const refreshSpy = jest.spyOn(app, 'refreshFeeds');
            
            fireEvent.keydown(document, 'r');
            
            expect(refreshSpy).toHaveBeenCalled();
        });

        test('should ignore shortcuts when typing in input', () => {
            const input = document.createElement('input');
            document.body.appendChild(input);
            input.focus();
            
            const selectSpy = jest.spyOn(app, 'selectNextArticle');
            
            fireEvent.keydown(input, 'j');
            
            expect(selectSpy).not.toHaveBeenCalled();
        });
    });

    describe('OPML Import', () => {
        beforeEach(async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
        });

        test('should show import OPML modal', () => {
            const importBtn = document.getElementById('import-opml-btn');
            fireEvent.click(importBtn);
            
            const modal = document.getElementById('import-opml-modal');
            expectElementToBeVisible(modal);
        });

        test('should import OPML file successfully', async () => {
            mockFormData();
            mockFetch({
                '/api/feeds/import': createMockResponse({ imported_count: 5 })
            });
            
            app.showImportOpmlModal();
            
            const fileInput = document.getElementById('opml-file');
            const form = document.getElementById('import-opml-form');
            const mockFile = createMockFile();
            
            Object.defineProperty(fileInput, 'files', {
                value: [mockFile],
                writable: false,
            });
            
            fireEvent.submit(form);
            
            await waitFor(() => document.getElementById('import-opml-modal').style.display === 'none');
            
            expect(fetch).toHaveBeenCalledWith('/api/feeds/import', {
                method: 'POST',
                body: expect.any(Object)
            });
        });

        test('should handle file size validation', async () => {
            const showErrorSpy = jest.spyOn(app, 'showError');
            
            app.showImportOpmlModal();
            
            const fileInput = document.getElementById('opml-file');
            const form = document.getElementById('import-opml-form');
            
            // Create a large file (over 10MB)
            const largeFile = createMockFile('large.opml', 'x'.repeat(11 * 1024 * 1024));
            Object.defineProperty(fileInput, 'files', {
                value: [largeFile],
                writable: false,
            });
            
            fireEvent.submit(form);
            
            expect(showErrorSpy).toHaveBeenCalledWith('File is too large (max 10MB)');
        });
    });

    describe('Error Handling', () => {
        beforeEach(async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
        });

        test('should show error message', () => {
            app.showError('Test error message');
            
            const errorDiv = document.querySelector('.error');
            expect(errorDiv).toBeTruthy();
            expect(errorDiv.textContent).toBe('Test error message');
        });

        test('should show success message', () => {
            app.showSuccess('Test success message');
            
            const successDiv = document.querySelector('.success');
            expect(successDiv).toBeTruthy();
            expect(successDiv.textContent).toBe('Test success message');
        });

        test('should remove error message after timeout', (done) => {
            app.showError('Test error');
            
            setTimeout(() => {
                const errorDiv = document.querySelector('.error');
                expect(errorDiv).toBeNull();
                done();
            }, 5100);
        });

        test('should handle API errors gracefully', async () => {
            mockFetch({
                '/api/feeds': createMockResponse(
                    { error: 'Server error' },
                    { status: 500, ok: false }
                )
            });
            
            const showErrorSpy = jest.spyOn(app, 'showError');
            
            await app.loadFeeds();
            
            expect(showErrorSpy).toHaveBeenCalledWith('Failed to load feeds: HTTP 500');
        });
    });

    describe('Subscription Management', () => {
        beforeEach(async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
        });

        test('should show subscription limit modal', () => {
            const error = {
                limit_reached: true,
                current_limit: 20,
                error: 'Feed limit reached'
            };
            
            app.showSubscriptionLimitModal(error);
            
            const modal = document.querySelector('.modal');
            expect(modal).toBeTruthy();
            expect(modal.querySelector('h2').textContent).toBe('Upgrade to Pro');
        });

        test('should show trial expired modal', () => {
            const error = {
                trial_expired: true,
                error: 'Trial expired'
            };
            
            app.showTrialExpiredModal(error);
            
            const modal = document.querySelector('.modal');
            expect(modal).toBeTruthy();
            expect(modal.querySelector('h2').textContent).toBe('Free Trial Expired');
        });

        test('should start upgrade process', async () => {
            mockFetch({
                '/api/stripe/config': createMockResponse({ publishable_key: 'pk_test_123' }),
                '/api/subscription/checkout': createMockResponse({ session_url: 'https://checkout.stripe.com/123' })
            });
            
            await app.startUpgradeProcess();
            
            expect(window.location.href).toBe('https://checkout.stripe.com/123');
        });
    });

    describe('Utility Functions', () => {
        beforeEach(async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
        });

        test('should escape HTML properly', () => {
            const escaped = app.escapeHtml('<script>alert("xss")</script>');
            expect(escaped).toBe('&lt;script&gt;alert("xss")&lt;/script&gt;');
        });

        test('should update unread counts', async () => {
            app.feeds = createTestFeeds();
            app.renderFeeds();
            
            await app.updateUnreadCounts();
            
            const feed1Count = document.querySelector('[data-feed-id="1"] .unread-count');
            const feed2Count = document.querySelector('[data-feed-id="2"] .unread-count');
            const allCount = document.getElementById('all-unread-count');
            
            expect(feed1Count.textContent).toBe('5');
            expect(feed2Count.textContent).toBe('3');
            expect(allCount.textContent).toBe('8');
        });
    });
});