const { 
    waitFor, 
    waitForElement, 
    fireEvent, 
    createTestArticles, 
    createTestFeeds,
    createMockResponse,
    mockFormData,
    createMockFile,
    expectElementToHaveClass,
    expectElementNotToHaveClass,
    expectElementToBeVisible,
    expectElementToBeHidden 
} = require('./utils.js');

// Load both apps
const fs = require('fs');
const path = require('path');

const appJsPath = path.join(__dirname, '../static/js/app.js');
const appJsContent = fs.readFileSync(appJsPath, 'utf8');

describe('GoRead2 Frontend Integration Tests', () => {
    let app;

    beforeEach(async () => {
        // Execute the app.js content to define GoReadApp class
        eval(appJsContent);
        
        // Mock successful authentication and comprehensive API responses
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
                current_feeds: 2, 
                feed_limit: 20, 
                trial_days_remaining: 15,
                trial_ends_at: '2024-12-31T23:59:59Z'
            }),
            '/api/feeds': createMockResponse(createTestFeeds()),
            '/api/feeds/unread-counts': createMockResponse({ '1': 5, '2': 3 }),
            '/api/feeds/all/articles': createMockResponse(createTestArticles()),
            '/api/feeds/1/articles': createMockResponse(createTestArticles().filter(a => a.feed_id === 1)),
            '/api/feeds/2/articles': createMockResponse(createTestArticles().filter(a => a.feed_id === 2))
        });
    });

    afterEach(() => {
        if (app) {
            app = null;
        }
    });

    describe('Complete User Workflow', () => {
        test('should complete full RSS reader workflow', async () => {
            // Initialize app
            app = new GoReadApp();
            await waitFor(() => app.user !== null && app.feeds.length > 0);
            
            // Verify initial state
            expect(app.feeds).toHaveLength(2);
            expect(document.querySelectorAll('.feed-item:not(.special)')).toHaveLength(2);
            
            // Select "All Articles" and verify articles load
            const allFeedItem = document.querySelector('[data-feed-id="all"]');
            fireEvent.click(allFeedItem);
            
            await waitFor(() => app.articles.length > 0);
            expect(app.articles).toHaveLength(3);
            expect(document.querySelectorAll('.article-item')).toHaveLength(3);
            
            // Select specific feed
            const feed1Item = document.querySelector('[data-feed-id="1"]');
            fireEvent.click(feed1Item);
            
            await waitFor(() => app.currentFeed === 1);
            expectElementToHaveClass(feed1Item, 'active');
            
            // Select first article
            const firstArticle = document.querySelector('[data-index="0"]');
            fireEvent.click(firstArticle);
            
            await waitFor(() => app.currentArticle === 0);
            expectElementToHaveClass(firstArticle, 'active');
            
            // Verify article content is displayed
            const articleContent = document.getElementById('article-content');
            expect(articleContent.innerHTML).toContain('Test Article 1');
            
            // Test keyboard navigation
            fireEvent.keydown(document, 'j'); // Next article
            await waitFor(() => app.currentArticle === 1);
            
            fireEvent.keydown(document, 'k'); // Previous article
            await waitFor(() => app.currentArticle === 0);
            
            // Test star toggle
            mockFetch({
                '/api/articles/1/star': createMockResponse({})
            });
            
            fireEvent.keydown(document, 's');
            await waitFor(() => fetch.mock.calls.some(call => call[0] === '/api/articles/1/star'));
            
            // Test mark as read
            mockFetch({
                '/api/articles/1/read': createMockResponse({})
            });
            
            fireEvent.keydown(document, 'm');
            await waitFor(() => fetch.mock.calls.some(call => call[0] === '/api/articles/1/read'));
        });

        test('should handle feed management workflow', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null && app.feeds.length > 0);
            
            // Add new feed
            mockFetch({
                '/api/feeds': createMockResponse({ 
                    id: 3, 
                    title: 'New Feed', 
                    url: 'https://example.com/new' 
                })
            });
            
            // Open add feed modal
            const addBtn = document.getElementById('add-feed-btn');
            fireEvent.click(addBtn);
            
            const modal = document.getElementById('add-feed-modal');
            expectElementToBeVisible(modal);
            
            // Fill and submit form
            const urlInput = document.getElementById('feed-url');
            const form = document.getElementById('add-feed-form');
            
            urlInput.value = 'https://example.com/new';
            fireEvent.submit(form);
            
            await waitFor(() => modal.style.display === 'none');
            
            expect(fetch).toHaveBeenCalledWith('/api/feeds', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url: 'https://example.com/new' })
            });
            
            // Delete feed
            window.confirm = jest.fn(() => true);
            mockFetch({
                '/api/feeds/1': createMockResponse({})
            });
            
            const deleteBtn = document.querySelector('[data-feed-id="1"] .delete-feed');
            fireEvent.click(deleteBtn);
            
            await waitFor(() => fetch.mock.calls.some(call => 
                call[0] === '/api/feeds/1' && call[1]?.method === 'DELETE'
            ));
            
            expect(window.confirm).toHaveBeenCalledWith(
                'Are you sure you want to remove this feed from your subscriptions?'
            );
        });

        test('should handle OPML import workflow', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
            
            mockFormData();
            mockFetch({
                '/api/feeds/import': createMockResponse({ imported_count: 3 })
            });
            
            // Open import modal
            const importBtn = document.getElementById('import-opml-btn');
            fireEvent.click(importBtn);
            
            const modal = document.getElementById('import-opml-modal');
            expectElementToBeVisible(modal);
            
            // Select file and submit
            const fileInput = document.getElementById('opml-file');
            const form = document.getElementById('import-opml-form');
            const mockFile = createMockFile('feeds.opml', '<?xml version="1.0"?><opml><body><outline text="Tech News" xmlUrl="https://example.com/tech"/></body></opml>');
            
            Object.defineProperty(fileInput, 'files', {
                value: [mockFile],
                writable: false,
            });
            
            fireEvent.submit(form);
            
            await waitFor(() => modal.style.display === 'none');
            
            expect(fetch).toHaveBeenCalledWith('/api/feeds/import', {
                method: 'POST',
                body: expect.any(Object)
            });
            
            // Verify success message
            await waitFor(() => document.querySelector('.success'));
            const successMsg = document.querySelector('.success');
            expect(successMsg.textContent).toContain('Successfully imported 3 feed(s)');
        });
    });

    describe('Error Handling Integration', () => {
        test('should handle subscription limit during feed addition', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
            
            // Mock feed limit error
            mockFetch({
                '/api/feeds': createMockResponse(
                    { 
                        error: 'Feed limit reached', 
                        limit_reached: true, 
                        current_limit: 20 
                    },
                    { status: 402, ok: false }
                )
            });
            
            // Try to add feed
            app.showAddFeedModal();
            const urlInput = document.getElementById('feed-url');
            const form = document.getElementById('add-feed-form');
            
            urlInput.value = 'https://example.com/new';
            fireEvent.submit(form);
            
            // Wait for upgrade modal to appear
            await waitFor(() => document.querySelector('.modal h2'));
            
            const upgradeModal = document.querySelector('.modal h2');
            expect(upgradeModal.textContent).toBe('Upgrade to Pro');
            
            // Test upgrade process
            mockFetch({
                '/api/stripe/config': createMockResponse({ publishable_key: 'pk_test_123' }),
                '/api/subscription/checkout': createMockResponse({ 
                    session_url: 'https://checkout.stripe.com/123' 
                })
            });
            
            const upgradeBtn = document.getElementById('upgrade-btn');
            fireEvent.click(upgradeBtn);
            
            await waitFor(() => window.location.href === 'https://checkout.stripe.com/123');
        });

        test('should handle trial expired during OPML import', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
            
            mockFormData();
            mockFetch({
                '/api/feeds/import': createMockResponse(
                    { 
                        error: 'Trial expired', 
                        trial_expired: true 
                    },
                    { status: 402, ok: false }
                )
            });
            
            // Try to import OPML
            app.showImportOpmlModal();
            const fileInput = document.getElementById('opml-file');
            const form = document.getElementById('import-opml-form');
            const mockFile = createMockFile();
            
            Object.defineProperty(fileInput, 'files', {
                value: [mockFile],
                writable: false,
            });
            
            fireEvent.submit(form);
            
            // Wait for trial expired modal
            await waitFor(() => document.querySelector('.modal h2'));
            
            const expiredModal = document.querySelector('.modal h2');
            expect(expiredModal.textContent).toBe('Free Trial Expired');
        });

        test('should handle network errors gracefully', async () => {
            // Mock network failure
            global.fetch.mockRejectedValueOnce(new Error('Network error'));
            
            app = new GoReadApp();
            
            // Should show error message
            await waitFor(() => document.querySelector('.error'));
            
            const errorMsg = document.querySelector('.error');
            expect(errorMsg.textContent).toContain('Failed to load feeds');
        });
    });

    describe('User Interface Interactions', () => {
        test('should handle modal interactions correctly', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
            
            // Test help modal
            const helpBtn = document.getElementById('help-btn');
            fireEvent.click(helpBtn);
            
            let modal = document.getElementById('help-modal');
            expectElementToBeVisible(modal);
            
            // Close with X button
            const closeBtn = modal.querySelector('.close');
            fireEvent.click(closeBtn);
            expectElementToBeHidden(modal);
            
            // Test add feed modal
            const addBtn = document.getElementById('add-feed-btn');
            fireEvent.click(addBtn);
            
            modal = document.getElementById('add-feed-modal');
            expectElementToBeVisible(modal);
            
            // Close by clicking outside
            fireEvent.click(modal);
            expectElementToBeHidden(modal);
            
            // Test import modal
            const importBtn = document.getElementById('import-opml-btn');
            fireEvent.click(importBtn);
            
            modal = document.getElementById('import-opml-modal');
            expectElementToBeVisible(modal);
            
            // Close with cancel button
            const cancelBtn = document.getElementById('cancel-import-opml');
            fireEvent.click(cancelBtn);
            expectElementToBeHidden(modal);
        });

        test('should update UI state correctly during article navigation', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null && app.feeds.length > 0);
            
            // Load articles
            const allFeedItem = document.querySelector('[data-feed-id="all"]');
            fireEvent.click(allFeedItem);
            
            await waitFor(() => app.articles.length > 0);
            
            // Test article selection state management
            const articles = document.querySelectorAll('.article-item');
            expect(articles).toHaveLength(3);
            
            // First article should be auto-selected
            expectElementToHaveClass(articles[0], 'active');
            
            // Select second article
            fireEvent.click(articles[1]);
            await waitFor(() => app.currentArticle === 1);
            
            expectElementNotToHaveClass(articles[0], 'active');
            expectElementToHaveClass(articles[1], 'active');
            
            // Test star button UI updates
            const starBtn = articles[1].querySelector('.star-btn');
            const isStarred = starBtn.classList.contains('starred');
            
            mockFetch({
                '/api/articles/2/star': createMockResponse({})
            });
            
            fireEvent.click(starBtn);
            
            await waitFor(() => starBtn.classList.contains('starred') !== isStarred);
        });

        test('should handle unread count updates', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null && app.feeds.length > 0);
            
            // Verify initial unread counts
            const feed1Count = document.querySelector('[data-feed-id="1"] .unread-count');
            const feed2Count = document.querySelector('[data-feed-id="2"] .unread-count');
            const allCount = document.getElementById('all-unread-count');
            
            expect(feed1Count.textContent).toBe('5');
            expect(feed2Count.textContent).toBe('3');
            expect(allCount.textContent).toBe('8');
            
            // Mock updated counts
            mockFetch({
                '/api/feeds/unread-counts': createMockResponse({ '1': 3, '2': 2 })
            });
            
            // Trigger update
            await app.updateUnreadCounts();
            
            expect(feed1Count.textContent).toBe('3');
            expect(feed2Count.textContent).toBe('2');
            expect(allCount.textContent).toBe('5');
        });
    });

    describe('Accessibility and Usability', () => {
        test('should handle keyboard navigation properly', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null && app.feeds.length > 0);
            
            // Load articles
            const allFeedItem = document.querySelector('[data-feed-id="all"]');
            fireEvent.click(allFeedItem);
            await waitFor(() => app.articles.length > 0);
            
            // Test that shortcuts don't interfere with form inputs
            const urlInput = document.getElementById('feed-url');
            urlInput.focus();
            
            const selectSpy = jest.spyOn(app, 'selectNextArticle');
            fireEvent.keydown(urlInput, 'j');
            
            expect(selectSpy).not.toHaveBeenCalled();
            
            // Test shortcuts work on document
            fireEvent.keydown(document, 'j');
            expect(selectSpy).toHaveBeenCalled();
        });

        test('should provide proper focus management', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
            
            // Test modal focus
            app.showAddFeedModal();
            
            const urlInput = document.getElementById('feed-url');
            expect(document.activeElement).toBe(urlInput);
        });

        test('should handle error states gracefully', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
            
            // Test error display and removal
            app.showError('Test error message');
            
            const errorDiv = document.querySelector('.error');
            expect(errorDiv).toBeTruthy();
            expect(errorDiv.textContent).toBe('Test error message');
            
            // Test that new errors replace old ones
            app.showError('New error message');
            
            const errorDivs = document.querySelectorAll('.error');
            expect(errorDivs).toHaveLength(1);
            expect(errorDivs[0].textContent).toBe('New error message');
        });
    });

    describe('Performance and Loading States', () => {
        test('should show loading states during async operations', async () => {
            app = new GoReadApp();
            await waitFor(() => app.user !== null);
            
            // Test article loading state
            app.loadArticles('all');
            
            const articleList = document.getElementById('article-list');
            expect(articleList.innerHTML).toContain('Loading articles...');
            
            await waitFor(() => app.articles.length > 0);
            expect(articleList.innerHTML).not.toContain('Loading articles...');
        });

        test('should handle concurrent API calls correctly', async () => {
            app = new GoReadApp();
            
            // Make multiple concurrent calls
            const promises = [
                app.checkAuth(),
                app.loadSubscriptionInfo(),
                app.loadFeeds()
            ];
            
            await Promise.all(promises);
            
            expect(app.user).toBeTruthy();
            expect(app.subscriptionInfo).toBeTruthy();
            expect(app.feeds.length).toBeGreaterThan(0);
        });
    });
});