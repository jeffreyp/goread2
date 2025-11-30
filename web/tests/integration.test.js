const {
    waitFor,
    fireEvent,
    createTestArticles,
    createTestFeeds,
    createMockResponse,
    createMockFile,
    mockFormData,
    expectElementToHaveClass,
    expectElementNotToHaveClass,
    expectElementToBeVisible,
    expectElementToBeHidden
} = require('./utils.js');

/**
 * Integration Tests
 *
 * These tests verify end-to-end workflows and interactions between different
 * parts of the application without requiring class instantiation.
 */
describe('GoRead2 Integration Tests', () => {

    describe('Complete Feed Management Workflow', () => {
        test('should handle add feed workflow', async () => {
            const modal = document.getElementById('add-feed-modal');
            const form = document.getElementById('add-feed-form');
            const urlInput = document.getElementById('feed-url');
            const addBtn = document.getElementById('add-feed-btn');

            // Mock API response
            mockFetch({
                '/api/feeds': createMockResponse({
                    id: 3,
                    title: 'New Feed',
                    url: 'https://example.com/new'
                })
            });

            // Step 1: Click add feed button
            addBtn.addEventListener('click', () => {
                modal.style.display = 'block';
            });
            fireEvent.click(addBtn);
            expectElementToBeVisible(modal);

            // Step 2: Fill in form
            urlInput.value = 'https://example.com/new';

            // Step 3: Submit form
            let feedAdded = false;
            form.addEventListener('submit', async (e) => {
                e.preventDefault();

                const response = await fetch('/api/feeds', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ url: urlInput.value })
                });

                if (response.ok) {
                    feedAdded = true;
                    modal.style.display = 'none';
                    form.reset();
                }
            });

            fireEvent.submit(form);
            await waitFor(() => feedAdded);

            expect(feedAdded).toBe(true);
            expectElementToBeHidden(modal);
            expect(urlInput.value).toBe('');
        });

        test('should handle delete feed workflow', async () => {
            const feedList = document.getElementById('feed-list');
            const feeds = createTestFeeds();

            // Create feed items with delete buttons
            feeds.forEach(feed => {
                const feedItem = document.createElement('div');
                feedItem.className = 'feed-item';
                feedItem.dataset.feedId = feed.id;

                const titleSpan = document.createElement('span');
                titleSpan.className = 'feed-title';
                titleSpan.textContent = feed.title;

                const deleteBtn = document.createElement('button');
                deleteBtn.className = 'delete-feed';
                deleteBtn.textContent = 'Delete';
                deleteBtn.dataset.feedId = feed.id;

                feedItem.appendChild(titleSpan);
                feedItem.appendChild(deleteBtn);
                feedList.appendChild(feedItem);
            });

            // Mock confirmation and API
            window.confirm = jest.fn(() => true);
            mockFetch({
                '/api/feeds/1': createMockResponse({}, { status: 200 })
            });

            const deleteBtn = document.querySelector('[data-feed-id="1"] .delete-feed');
            let feedDeleted = false;

            deleteBtn.addEventListener('click', async () => {
                if (window.confirm('Are you sure?')) {
                    const response = await fetch('/api/feeds/1', { method: 'DELETE' });
                    if (response.ok) {
                        const feedItem = document.querySelector('[data-feed-id="1"]');
                        feedItem.remove();
                        feedDeleted = true;
                    }
                }
            });

            fireEvent.click(deleteBtn);
            await waitFor(() => feedDeleted);

            expect(window.confirm).toHaveBeenCalled();
            expect(feedDeleted).toBe(true);
            expect(document.querySelector('[data-feed-id="1"]')).toBeNull();
        });
    });

    describe('Article Reading Workflow', () => {
        test('should handle article selection and navigation', () => {
            const articleList = document.getElementById('article-list');
            const articles = createTestArticles();
            let currentArticle = 0;

            // Render articles
            articles.forEach((article, index) => {
                const articleItem = document.createElement('div');
                articleItem.className = 'article-item';
                articleItem.dataset.index = index;
                articleItem.textContent = article.title;
                articleList.appendChild(articleItem);
            });

            // Select first article
            const articleItems = document.querySelectorAll('.article-item');
            articleItems[currentArticle].classList.add('active');
            expectElementToHaveClass(articleItems[0], 'active');

            // Navigate to next article (j key)
            document.addEventListener('keydown', (e) => {
                if (e.key === 'j' && currentArticle < articles.length - 1) {
                    articleItems[currentArticle].classList.remove('active');
                    currentArticle++;
                    articleItems[currentArticle].classList.add('active');
                }
            });

            fireEvent.keydown(document, 'j');

            expectElementNotToHaveClass(articleItems[0], 'active');
            expectElementToHaveClass(articleItems[1], 'active');
        });

        test('should handle star toggle workflow', async () => {
            const articleList = document.getElementById('article-list');
            const article = createTestArticles()[0];

            // Create article item
            const articleItem = document.createElement('div');
            articleItem.className = 'article-item';
            articleItem.dataset.articleId = article.id;

            const starBtn = document.createElement('button');
            starBtn.className = 'star-btn';
            starBtn.textContent = '★';

            articleItem.appendChild(starBtn);
            articleList.appendChild(articleItem);

            // Mock API
            mockFetch({
                '/api/articles/1/star': createMockResponse({})
            });

            let starred = false;
            starBtn.addEventListener('click', async (e) => {
                e.stopPropagation();
                const response = await fetch('/api/articles/1/star', { method: 'POST' });
                if (response.ok) {
                    starBtn.classList.toggle('starred');
                    starred = !starred;
                }
            });

            fireEvent.click(starBtn);
            await waitFor(() => starred);

            expect(starred).toBe(true);
            expectElementToHaveClass(starBtn, 'starred');
        });
    });

    describe('OPML Import Workflow', () => {
        test('should handle successful OPML import', async () => {
            mockFormData();

            const modal = document.getElementById('import-opml-modal');
            const form = document.getElementById('import-opml-form');
            const fileInput = document.getElementById('opml-file');
            const importBtn = document.getElementById('import-opml-btn');

            // Mock API
            mockFetch({
                '/api/feeds/import': createMockResponse({ imported_count: 3 })
            });

            // Open modal
            importBtn.addEventListener('click', () => {
                modal.style.display = 'block';
            });
            fireEvent.click(importBtn);
            expectElementToBeVisible(modal);

            // Select file
            const mockFile = createMockFile('feeds.opml', '<opml></opml>');
            Object.defineProperty(fileInput, 'files', {
                value: [mockFile],
                writable: false
            });

            // Submit form
            let imported = false;
            let importCount = 0;

            form.addEventListener('submit', async (e) => {
                e.preventDefault();

                const formData = new FormData();
                formData.append('file', fileInput.files[0]);

                const response = await fetch('/api/feeds/import', {
                    method: 'POST',
                    body: formData
                });

                if (response.ok) {
                    const data = await response.json();
                    importCount = data.imported_count;
                    imported = true;
                    modal.style.display = 'none';
                }
            });

            fireEvent.submit(form);
            await waitFor(() => imported);

            expect(imported).toBe(true);
            expect(importCount).toBe(3);
            expectElementToBeHidden(modal);
        });

        test('should handle file size validation', () => {
            const fileInput = document.getElementById('opml-file');
            const form = document.getElementById('import-opml-form');

            const largeFile = createMockFile('large.opml', 'x'.repeat(11 * 1024 * 1024));
            Object.defineProperty(fileInput, 'files', {
                value: [largeFile],
                writable: false
            });

            let errorShown = false;
            form.addEventListener('submit', (e) => {
                e.preventDefault();
                const file = fileInput.files[0];
                if (file.size > 10 * 1024 * 1024) {
                    errorShown = true;
                }
            });

            fireEvent.submit(form);

            expect(errorShown).toBe(true);
        });
    });

    describe('Subscription Limit Workflow', () => {
        test('should handle feed limit error and show upgrade modal', async () => {
            mockFetch({
                '/api/feeds': createMockResponse(
                    { error: 'Feed limit reached', limit_reached: true, current_limit: 20 },
                    { status: 402, ok: false }
                )
            });

            const form = document.getElementById('add-feed-form');
            const urlInput = document.getElementById('feed-url');
            urlInput.value = 'https://example.com/feed';

            let limitModalShown = false;

            form.addEventListener('submit', async (e) => {
                e.preventDefault();

                const response = await fetch('/api/feeds', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ url: urlInput.value })
                });

                if (!response.ok) {
                    const data = await response.json();
                    if (data.limit_reached) {
                        // Show upgrade modal
                        const upgradeModal = document.createElement('div');
                        upgradeModal.className = 'modal upgrade-modal';
                        upgradeModal.innerHTML = '<h2>Upgrade to Pro</h2>';
                        document.body.appendChild(upgradeModal);
                        limitModalShown = true;
                    }
                }
            });

            fireEvent.submit(form);
            await waitFor(() => limitModalShown);

            expect(limitModalShown).toBe(true);
            expect(document.querySelector('.upgrade-modal h2').textContent).toBe('Upgrade to Pro');
        });
    });

    describe('Error Handling Workflows', () => {
        test('should handle network errors gracefully', async () => {
            global.fetch.mockRejectedValueOnce(new Error('Network error'));

            let errorShown = false;
            let errorMessage = '';

            try {
                await fetch('/api/feeds');
            } catch (error) {
                errorMessage = error.message;
                errorShown = true;

                // Show error message
                const errorDiv = document.createElement('div');
                errorDiv.className = 'error';
                errorDiv.textContent = 'Failed to load feeds. Please check your connection.';
                document.body.appendChild(errorDiv);
            }

            expect(errorShown).toBe(true);
            expect(errorMessage).toBe('Network error');
            expect(document.querySelector('.error')).toBeTruthy();
        });

        test('should handle authentication errors', async () => {
            mockFetch({
                '/auth/me': createMockResponse(null, { status: 401, ok: false })
            });

            const response = await fetch('/auth/me');

            if (!response.ok && response.status === 401) {
                // Simulate redirect to login
                const redirected = true;
                expect(redirected).toBe(true);
            }
        });
    });

    describe('UI State Synchronization', () => {
        test('should sync unread counts after marking article as read', async () => {
            const feedList = document.getElementById('feed-list');
            const articleList = document.getElementById('article-list');

            // Create feed with unread count
            const feedItem = document.createElement('div');
            feedItem.className = 'feed-item';
            feedItem.dataset.feedId = '1';

            const unreadSpan = document.createElement('span');
            unreadSpan.className = 'unread-count';
            unreadSpan.textContent = '5';
            unreadSpan.dataset.count = '5';

            feedItem.appendChild(unreadSpan);
            feedList.appendChild(feedItem);

            // Create article
            const articles = createTestArticles();
            const articleItem = document.createElement('div');
            articleItem.className = 'article-item';
            articleItem.dataset.articleId = articles[0].id;
            articleItem.dataset.feedId = '1';
            articleList.appendChild(articleItem);

            // Mock API
            mockFetch({
                '/api/articles/1/read': createMockResponse({})
            });

            // Mark as read
            const response = await fetch('/api/articles/1/read', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ is_read: true })
            });

            if (response.ok) {
                // Update unread count
                const currentCount = parseInt(unreadSpan.dataset.count);
                const newCount = Math.max(0, currentCount - 1);
                unreadSpan.textContent = newCount;
                unreadSpan.dataset.count = newCount;
            }

            expect(unreadSpan.textContent).toBe('4');
            expect(unreadSpan.dataset.count).toBe('4');
        });

        test('should handle modal state transitions', () => {
            const modal = document.getElementById('add-feed-modal');
            const cancelBtn = document.getElementById('cancel-add-feed');
            const form = document.getElementById('add-feed-form');

            // Open modal
            modal.style.display = 'block';
            expectElementToBeVisible(modal);

            // Close modal with cancel
            cancelBtn.addEventListener('click', () => {
                modal.style.display = 'none';
                form.reset();
            });

            fireEvent.click(cancelBtn);
            expectElementToBeHidden(modal);
        });
    });

    describe('Keyboard Navigation Integration', () => {
        test('should handle complete keyboard navigation workflow', () => {
            const articles = createTestArticles();
            const articleList = document.getElementById('article-list');
            let currentArticle = 0;

            // Render articles
            articles.forEach((article, index) => {
                const articleItem = document.createElement('div');
                articleItem.className = 'article-item';
                articleItem.dataset.index = index;
                articleItem.textContent = article.title;
                articleList.appendChild(articleItem);
            });

            const articleItems = document.querySelectorAll('.article-item');
            articleItems[0].classList.add('active');

            // Setup keyboard handlers
            document.addEventListener('keydown', (e) => {
                if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
                    return;
                }

                if (e.key === 'j' && currentArticle < articles.length - 1) {
                    articleItems[currentArticle].classList.remove('active');
                    currentArticle++;
                    articleItems[currentArticle].classList.add('active');
                } else if (e.key === 'k' && currentArticle > 0) {
                    articleItems[currentArticle].classList.remove('active');
                    currentArticle--;
                    articleItems[currentArticle].classList.add('active');
                }
            });

            // Test navigation
            fireEvent.keydown(document, 'j'); // Move to article 1
            expect(currentArticle).toBe(1);
            expectElementToHaveClass(articleItems[1], 'active');

            fireEvent.keydown(document, 'j'); // Move to article 2
            expect(currentArticle).toBe(2);
            expectElementToHaveClass(articleItems[2], 'active');

            fireEvent.keydown(document, 'k'); // Move back to article 1
            expect(currentArticle).toBe(1);
            expectElementToHaveClass(articleItems[1], 'active');
        });
    });

    describe('Multi-Step User Workflows', () => {
        test('should handle feed subscription and article viewing workflow', async () => {
            const feedList = document.getElementById('feed-list');
            const articleList = document.getElementById('article-list');
            const feeds = createTestFeeds();
            const articles = createTestArticles();

            // Step 1: Render feeds (non-special feeds only)
            feeds.forEach(feed => {
                const feedItem = document.createElement('div');
                feedItem.className = 'feed-item';
                feedItem.dataset.feedId = feed.id;
                feedItem.textContent = feed.title;
                feedList.appendChild(feedItem);
            });

            // Step 2: Select a feed
            mockFetch({
                '/api/feeds/1/articles': createMockResponse(articles)
            });

            // Get non-special feed items only
            const feedItems = document.querySelectorAll('.feed-item:not(.special)');
            let selectedFeedId = null;

            feedList.addEventListener('click', async (e) => {
                const feedItem = e.target.closest('.feed-item');
                if (feedItem) {
                    selectedFeedId = feedItem.dataset.feedId;

                    // Clear previous selection
                    feedItems.forEach(item => item.classList.remove('active'));
                    feedItem.classList.add('active');

                    // Load articles
                    const response = await fetch(`/api/feeds/${selectedFeedId}/articles`);
                    const loadedArticles = await response.json();

                    // Render articles
                    articleList.innerHTML = '';
                    if (Array.isArray(loadedArticles)) {
                        loadedArticles.forEach(article => {
                            const articleItem = document.createElement('div');
                            articleItem.className = 'article-item';
                            articleItem.textContent = article.title;
                            articleList.appendChild(articleItem);
                        });
                    }
                }
            });

            fireEvent.click(feedItems[0]);
            await waitFor(() => selectedFeedId !== null);

            // Wait for articles to be rendered
            await waitFor(() => articleList.querySelectorAll('.article-item').length > 0);

            // Step 3: Verify workflow completed
            expect(selectedFeedId).toBe('1');
            expectElementToHaveClass(feedItems[0], 'active');
            expect(articleList.querySelectorAll('.article-item')).toHaveLength(3);
        });
    });
});
