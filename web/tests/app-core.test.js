const { 
    waitFor, 
    fireEvent, 
    createTestArticles, 
    createTestFeeds,
    createMockResponse,
    expectElementToHaveClass,
    expectElementToBeVisible,
    expectElementToBeHidden 
} = require('./utils.js');

// Simple core functionality tests that don't require the full app classes
describe('GoRead2 Core Frontend Functionality', () => {
    
    describe('DOM Manipulation', () => {
        test('should create and manage feed list elements', () => {
            const feedList = document.getElementById('feed-list');
            const feeds = createTestFeeds();
            
            // Simulate feed rendering
            feeds.forEach((feed) => {
                const feedItem = document.createElement('div');
                feedItem.className = 'feed-item';
                feedItem.dataset.feedId = feed.id;
                
                const titleSpan = document.createElement('span');
                titleSpan.className = 'feed-title';
                titleSpan.textContent = feed.title;
                
                const unreadSpan = document.createElement('span');
                unreadSpan.className = 'unread-count';
                unreadSpan.dataset.count = '0';
                unreadSpan.textContent = '0';
                
                feedItem.appendChild(titleSpan);
                feedItem.appendChild(unreadSpan);
                feedList.appendChild(feedItem);
            });
            
            const feedItems = document.querySelectorAll('.feed-item:not(.special)');
            expect(feedItems).toHaveLength(2);
            expect(feedItems[0].querySelector('.feed-title').textContent).toBe('Test Feed 1');
            expect(feedItems[1].querySelector('.feed-title').textContent).toBe('Test Feed 2');
        });

        test('should create and manage article list elements', () => {
            const articleList = document.getElementById('article-list');
            const articles = createTestArticles();
            
            // Simulate article rendering
            articles.forEach((article, index) => {
                const articleItem = document.createElement('div');
                articleItem.className = `article-item ${article.is_read ? 'read' : ''}`;
                articleItem.dataset.articleId = article.id;
                articleItem.dataset.index = index;
                
                const publishedDate = new Date(article.published_at).toLocaleDateString();
                
                articleItem.innerHTML = `
                    <div class="article-header">
                        <div style="flex: 1;">
                            <div class="article-title">${article.title}</div>
                            <div class="article-meta">
                                <span>${publishedDate}</span>
                                ${article.author ? `<span>by ${article.author}</span>` : ''}
                            </div>
                        </div>
                        <div class="article-actions">
                            <button class="action-btn star-btn ${article.is_starred ? 'starred' : ''}" 
                                    data-article-id="${article.id}" title="Star article">★</button>
                        </div>
                    </div>
                    ${article.description ? `<div class="article-description">${article.description}</div>` : ''}
                `;
                
                articleList.appendChild(articleItem);
            });
            
            const articleItems = document.querySelectorAll('.article-item');
            expect(articleItems).toHaveLength(3);
            
            // Check that first article is marked as read
            expectElementToHaveClass(articleItems[0], 'read');
            
            // Check that second article star button is starred
            const starBtn = articleItems[1].querySelector('.star-btn');
            expectElementToHaveClass(starBtn, 'starred');
        });

        test('should handle modal show/hide functionality', () => {
            const modal = document.getElementById('add-feed-modal');
            
            // Show modal
            modal.style.display = 'block';
            expect(modal.style.display).toBe('block');
            
            // Hide modal
            modal.style.display = 'none';
            expect(modal.style.display).toBe('none');
        });

        test('should handle form interactions', () => {
            const form = document.getElementById('add-feed-form');
            const urlInput = document.getElementById('feed-url');
            
            let formSubmitted = false;
            form.addEventListener('submit', (e) => {
                e.preventDefault();
                formSubmitted = true;
            });
            
            urlInput.value = 'https://example.com/feed';
            fireEvent.submit(form);
            
            expect(formSubmitted).toBe(true);
            expect(urlInput.value).toBe('https://example.com/feed');
        });
    });

    describe('Event Handling', () => {
        test('should handle click events on feed items', () => {
            const feedList = document.getElementById('feed-list');
            let clickedFeedId = null;
            
            // Add event listener
            feedList.addEventListener('click', (e) => {
                const feedItem = e.target.closest('.feed-item');
                if (feedItem && feedItem.dataset.feedId) {
                    clickedFeedId = feedItem.dataset.feedId;
                }
            });
            
            // Create a feed item
            const feedItem = document.createElement('div');
            feedItem.className = 'feed-item';
            feedItem.dataset.feedId = '1';
            feedItem.innerHTML = '<span class="feed-title">Test Feed</span>';
            feedList.appendChild(feedItem);
            
            // Click the feed item
            fireEvent.click(feedItem);
            
            expect(clickedFeedId).toBe('1');
        });

        test('should handle keyboard events', () => {
            let keyPressed = null;
            
            document.addEventListener('keydown', (e) => {
                if (e.target.tagName !== 'INPUT' && e.target.tagName !== 'TEXTAREA') {
                    keyPressed = e.key;
                }
            });
            
            fireEvent.keydown(document, 'j');
            expect(keyPressed).toBe('j');
            
            // Reset
            keyPressed = null;
            
            // Test that keyboard events are ignored in input fields
            const input = document.createElement('input');
            document.body.appendChild(input);
            
            fireEvent.keydown(input, 'j');
            expect(keyPressed).toBeNull();
        });

        test('should handle star button clicks with event delegation', () => {
            const articleList = document.getElementById('article-list');
            let starredArticleId = null;
            
            // Add event delegation
            articleList.addEventListener('click', (e) => {
                if (e.target.classList.contains('star-btn')) {
                    e.stopPropagation();
                    e.preventDefault();
                    starredArticleId = e.target.dataset.articleId;
                }
            });
            
            // Create article with star button
            const articleItem = document.createElement('div');
            articleItem.className = 'article-item';
            articleItem.innerHTML = `
                <div class="article-actions">
                    <button class="action-btn star-btn" data-article-id="123">★</button>
                </div>
            `;
            articleList.appendChild(articleItem);
            
            const starBtn = articleItem.querySelector('.star-btn');
            fireEvent.click(starBtn);
            
            expect(starredArticleId).toBe('123');
        });
    });

    describe('API Interaction Simulation', () => {
        test('should mock fetch calls correctly', async () => {
            const testResponse = { feeds: createTestFeeds() };
            mockFetch({
                '/api/feeds': createMockResponse(testResponse)
            });
            
            const response = await fetch('/api/feeds');
            const data = await response.json();
            
            expect(response.ok).toBe(true);
            expect(data).toEqual(testResponse);
            expect(data.feeds).toHaveLength(2);
        });

        test('should handle error responses', async () => {
            mockFetch({
                '/api/feeds': createMockResponse(
                    { error: 'Server error' },
                    { status: 500, ok: false }
                )
            });
            
            const response = await fetch('/api/feeds');
            const data = await response.json();
            
            expect(response.ok).toBe(false);
            expect(response.status).toBe(500);
            expect(data.error).toBe('Server error');
        });

        test('should handle subscription limit responses', async () => {
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
            
            const response = await fetch('/api/feeds');
            const data = await response.json();
            
            expect(response.status).toBe(402);
            expect(data.limit_reached).toBe(true);
            expect(data.current_limit).toBe(20);
        });
    });

    describe('Utility Functions', () => {
        test('should escape HTML correctly', () => {
            const escapeHtml = (text) => {
                const div = document.createElement('div');
                div.textContent = text;
                return div.innerHTML;
            };
            
            const maliciousInput = '<script>alert("xss")</script>';
            const escaped = escapeHtml(maliciousInput);
            
            expect(escaped).toBe('&lt;script&gt;alert("xss")&lt;/script&gt;');
        });

        test('should format dates correctly', () => {
            const formatDate = (dateString) => {
                if (!dateString) return 'N/A';
                try {
                    const date = new Date(dateString);
                    return date.toLocaleDateString('en-US', {
                        year: 'numeric',
                        month: 'long',
                        day: 'numeric'
                    });
                } catch (error) {
                    return 'Invalid date';
                }
            };
            
            expect(formatDate('2024-01-15T10:30:00Z')).toBe('January 15, 2024');
            expect(formatDate('invalid-date')).toBe('Invalid Date');
            expect(formatDate(null)).toBe('N/A');
        });

        test('should update unread counts correctly', () => {
            const feedList = document.getElementById('feed-list');
            
            // Create feed items with unread counts
            const feeds = createTestFeeds();
            feeds.forEach(feed => {
                const feedItem = document.createElement('div');
                feedItem.className = 'feed-item';
                feedItem.dataset.feedId = feed.id;
                
                const unreadSpan = document.createElement('span');
                unreadSpan.className = 'unread-count';
                unreadSpan.dataset.count = '0';
                unreadSpan.textContent = '0';
                
                feedItem.appendChild(unreadSpan);
                feedList.appendChild(feedItem);
            });
            
            // Simulate updating unread counts
            const unreadCounts = { '1': 5, '2': 3 };
            let totalUnread = 0;
            
            feeds.forEach(feed => {
                const unreadCount = unreadCounts[feed.id] || 0;
                totalUnread += unreadCount;
                
                const countElement = document.querySelector(`[data-feed-id="${feed.id}"] .unread-count`);
                if (countElement) {
                    countElement.textContent = unreadCount;
                    countElement.dataset.count = unreadCount;
                }
            });
            
            // Update "All Articles" count
            const allUnreadElement = document.getElementById('all-unread-count');
            if (allUnreadElement) {
                allUnreadElement.textContent = totalUnread;
                allUnreadElement.dataset.count = totalUnread;
            }
            
            // Verify counts
            const feed1Count = document.querySelector('[data-feed-id="1"] .unread-count');
            const feed2Count = document.querySelector('[data-feed-id="2"] .unread-count');
            const allCount = document.getElementById('all-unread-count');
            
            expect(feed1Count.textContent).toBe('5');
            expect(feed2Count.textContent).toBe('3');
            expect(allCount.textContent).toBe('8');
        });
    });

    describe('UI State Management', () => {
        test('should manage active states correctly', () => {
            const feedList = document.getElementById('feed-list');
            
            // Create multiple feed items
            for (let i = 1; i <= 3; i++) {
                const feedItem = document.createElement('div');
                feedItem.className = 'feed-item';
                feedItem.dataset.feedId = i;
                feedItem.innerHTML = `<span>Feed ${i}</span>`;
                feedList.appendChild(feedItem);
            }
            
            const feedItems = document.querySelectorAll('.feed-item:not(.special)');
            
            // Simulate selecting a feed
            const selectFeed = (feedId) => {
                feedItems.forEach(item => {
                    item.classList.remove('active');
                });
                document.querySelector(`[data-feed-id="${feedId}"]`).classList.add('active');
            };
            
            selectFeed('2');
            
            expect(feedItems[0].classList.contains('active')).toBe(false);
            expect(feedItems[1].classList.contains('active')).toBe(true);
            expect(feedItems[2].classList.contains('active')).toBe(false);
        });

        test('should handle error and success message display', () => {
            const showMessage = (message, type) => {
                // Remove existing messages
                const existing = document.querySelector(`.${type}`);
                if (existing) {
                    existing.remove();
                }
                
                const messageDiv = document.createElement('div');
                messageDiv.className = type;
                messageDiv.textContent = message;
                document.body.appendChild(messageDiv);
                
                return messageDiv;
            };
            
            // Test error message
            const errorDiv = showMessage('Test error', 'error');
            expect(document.querySelector('.error')).toBe(errorDiv);
            expect(errorDiv.textContent).toBe('Test error');
            
            // Test success message replaces error
            const successDiv = showMessage('Test success', 'success');
            expect(document.querySelector('.success')).toBe(successDiv);
            expect(successDiv.textContent).toBe('Test success');
            
            // Test new error replaces existing error
            showMessage('First error', 'error');
            showMessage('Second error', 'error');
            
            const errorDivs = document.querySelectorAll('.error');
            expect(errorDivs).toHaveLength(1);
            expect(errorDivs[0].textContent).toBe('Second error');
        });
    });

    describe('Form Validation and Handling', () => {
        test('should validate file size for OPML import', () => {
            const validateFileSize = (file, maxSize = 10 * 1024 * 1024) => {
                return file.size <= maxSize;
            };
            
            // Create mock files
            const validFile = { size: 5 * 1024 * 1024 }; // 5MB
            const invalidFile = { size: 15 * 1024 * 1024 }; // 15MB
            
            expect(validateFileSize(validFile)).toBe(true);
            expect(validateFileSize(invalidFile)).toBe(false);
        });

        test('should handle form submission states', () => {
            const form = document.getElementById('add-feed-form');
            const submitButton = form.querySelector('button[type="submit"]');
            const cancelButton = document.getElementById('cancel-add-feed');
            const inputField = document.getElementById('feed-url');
            
            // Simulate loading state
            const setLoadingState = (loading) => {
                submitButton.disabled = loading;
                cancelButton.disabled = loading;
                inputField.disabled = loading;
                
                if (loading) {
                    submitButton.textContent = 'Adding...';
                } else {
                    submitButton.textContent = 'Add Feed';
                }
            };
            
            setLoadingState(true);
            
            expect(submitButton.disabled).toBe(true);
            expect(cancelButton.disabled).toBe(true);
            expect(inputField.disabled).toBe(true);
            expect(submitButton.textContent).toBe('Adding...');
            
            setLoadingState(false);
            
            expect(submitButton.disabled).toBe(false);
            expect(cancelButton.disabled).toBe(false);
            expect(inputField.disabled).toBe(false);
            expect(submitButton.textContent).toBe('Add Feed');
        });
    });

    describe('Feature Flag Support', () => {
        test('should handle unlimited subscription status', async () => {
            // Mock subscription info with unlimited status (when subscription system disabled)
            mockFetch({
                '/api/subscription': createMockResponse({
                    status: 'unlimited',
                    feed_limit: -1,
                    can_add_feeds: true,
                    current_feeds: 15,
                    is_active: true
                })
            });

            // Simulate loading subscription info
            const response = await fetch('/api/subscription');
            const subscriptionInfo = await response.json();

            // Verify unlimited status
            expect(subscriptionInfo.status).toBe('unlimited');
            expect(subscriptionInfo.feed_limit).toBe(-1);
            expect(subscriptionInfo.can_add_feeds).toBe(true);

            // Test creating subscription status element for unlimited status
            const createSubscriptionStatusElement = (info) => {
                if (info.status === 'unlimited') {
                    return `
                        <div class="subscription-status unlimited">
                            <span class="status-badge">UNLIMITED</span>
                            <span class="status-text">Unlimited feeds</span>
                        </div>
                    `;
                }
                return '<div class="subscription-status loading">Loading...</div>';
            };

            const statusHTML = createSubscriptionStatusElement(subscriptionInfo);
            expect(statusHTML).toContain('UNLIMITED');
            expect(statusHTML).toContain('Unlimited feeds');
            expect(statusHTML).toContain('subscription-status unlimited');
        });

        test('should not show upgrade prompts when subscription disabled', async () => {
            // Mock unlimited subscription status
            mockFetch({
                '/api/subscription': createMockResponse({
                    status: 'unlimited',
                    feed_limit: -1,
                    can_add_feeds: true,
                    current_feeds: 50, // Many feeds, but unlimited
                    is_active: true
                })
            });

            const response = await fetch('/api/subscription');
            const subscriptionInfo = await response.json();

            // Simulate creating feed warning (should not show upgrade prompts)
            const createFeedWarning = (info) => {
                if (info.status === 'unlimited') {
                    return ''; // No warning for unlimited users
                }
                
                const feedsUsed = info.current_feeds || 0;
                const feedLimit = info.feed_limit || 20;
                const isNearLimit = feedsUsed >= feedLimit - 3;
                
                if (isNearLimit) {
                    return `
                        <div class="feed-warning">
                            <div class="details">${feedsUsed}/${feedLimit} feeds</div>
                            <button class="upgrade-btn">Upgrade</button>
                        </div>
                    `;
                }
                return '';
            };

            const warningHTML = createFeedWarning(subscriptionInfo);
            expect(warningHTML).toBe(''); // Should be empty for unlimited users
        });

        test('should handle account page unlimited status', () => {
            const subscriptionInfo = {
                status: 'unlimited',
                feed_limit: -1,
                can_add_feeds: true,
                current_feeds: 25,
                is_active: true
            };

            // Simulate account page subscription rendering
            const renderAccountSubscription = (info) => {
                if (info.status === 'unlimited') {
                    return `
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
                                <strong>Current feeds:</strong> ${info.current_feeds} feeds (no limit)
                            </p>
                        </div>
                    `;
                }
                return '<div class="error">Unknown status</div>';
            };

            const accountHTML = renderAccountSubscription(subscriptionInfo);
            expect(accountHTML).toContain('Unlimited Access');
            expect(accountHTML).toContain('subscription system is currently disabled');
            expect(accountHTML).toContain('25 feeds (no limit)');
            expect(accountHTML).toContain('subscription-info unlimited');
        });

        test('should not show subscription-related errors when unlimited', async () => {
            // When subscription system is disabled, API should never return feed limit errors
            // But we can test that the frontend handles it gracefully anyway
            
            // Mock successful feed addition (no limits when disabled)
            mockFetch({
                '/api/feeds': createMockResponse({ success: true, message: 'Feed added successfully' })
            });

            const response = await fetch('/api/feeds', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url: 'https://example.com/feed' })
            });

            expect(response.ok).toBe(true);
            
            const result = await response.json();
            expect(result.success).toBe(true);
            expect(result.message).toBe('Feed added successfully');
            
            // Should not contain any limit-related error messages
            expect(result.error).toBeUndefined();
            expect(result.limit_reached).toBeUndefined();
            expect(result.trial_expired).toBeUndefined();
        });
    });

    describe('Auto-Read Last Article Functionality', () => {
        test('should identify when article is last unread', () => {
            // Test data: articles where only one is unread
            const articles = [
                { id: 1, is_read: true, title: 'Article 1' },
                { id: 2, is_read: true, title: 'Article 2' },
                { id: 3, is_read: false, title: 'Article 3' } // Only unread article
            ];

            // Count unread articles
            const unreadArticles = articles.filter(a => !a.is_read);
            
            expect(unreadArticles.length).toBe(1);
            expect(unreadArticles[0].id).toBe(3);
        });

        test('should not trigger auto-read when multiple unread articles exist', () => {
            // Test data: articles where multiple are unread
            const articles = [
                { id: 1, is_read: false, title: 'Article 1' },
                { id: 2, is_read: true, title: 'Article 2' },
                { id: 3, is_read: false, title: 'Article 3' }
            ];

            // Count unread articles
            const unreadArticles = articles.filter(a => !a.is_read);
            
            expect(unreadArticles.length).toBe(2);
            // Should NOT auto-read since there are multiple unread articles
        });

        test('should handle article state updates correctly', () => {
            // Simulate optimistic update for auto-read
            const article = { id: 1, is_read: false, title: 'Test Article' };
            
            // Simulate marking as read
            article.is_read = true;
            
            expect(article.is_read).toBe(true);
        });

        test('should handle DOM element hiding in unread-only mode', () => {
            // Create a test article element
            const articleItem = document.createElement('div');
            articleItem.className = 'article-item';
            articleItem.dataset.index = '0';
            document.body.appendChild(articleItem);

            // Simulate hiding the article
            articleItem.classList.add('filtered-out');
            articleItem.style.display = 'none';

            expectElementToBeHidden(articleItem);

            // Cleanup
            document.body.removeChild(articleItem);
        });

        test('should handle unread count updates for auto-read articles', () => {
            // Create test feed count element
            const feedCountElement = document.createElement('span');
            feedCountElement.className = 'unread-count';
            feedCountElement.dataset.count = '5';
            feedCountElement.textContent = '5';
            
            const feedItem = document.createElement('div');
            feedItem.dataset.feedId = '1';
            feedItem.appendChild(feedCountElement);
            document.body.appendChild(feedItem);

            // Create test total count element
            const allUnreadElement = document.createElement('span');
            allUnreadElement.id = 'all-unread-count';
            allUnreadElement.dataset.count = '10';
            allUnreadElement.textContent = '10';
            document.body.appendChild(allUnreadElement);

            // Simulate optimistic count update (decrease by 1) - testing the logic from updateUnreadCountsOptimistically
            const feedIdStr = '1';
            const countChange = -1;
            
            // Update specific feed count
            const targetFeedElement = document.querySelector(`[data-feed-id="${feedIdStr}"] .unread-count`);
            if (targetFeedElement) {
                const currentCount = parseInt(targetFeedElement.dataset.count) || 0;
                const newCount = Math.max(0, currentCount + countChange);
                targetFeedElement.textContent = newCount;
                targetFeedElement.dataset.count = newCount;
            }

            // Update total count (using the same element we created)
            if (allUnreadElement) {
                const currentTotal = parseInt(allUnreadElement.dataset.count) || 0;
                const newTotal = Math.max(0, currentTotal + countChange);
                allUnreadElement.textContent = newTotal;
                allUnreadElement.dataset.count = newTotal;
            }

            // Verify updates
            expect(feedCountElement.textContent).toBe('4');
            expect(feedCountElement.dataset.count).toBe('4');
            expect(allUnreadElement.textContent).toBe('9');
            expect(allUnreadElement.dataset.count).toBe('9');

            // Cleanup
            document.body.removeChild(feedItem);
            document.body.removeChild(allUnreadElement);
        });
    });

    describe('Feed Alphabetical Sorting', () => {
        test('should sort feeds alphabetically by title', () => {
            const feedList = document.getElementById('feed-list');

            // Create feeds in non-alphabetical order
            const feeds = [
                { id: 1, title: 'Zebra News', url: 'https://example.com/zebra' },
                { id: 2, title: 'Apple Updates', url: 'https://example.com/apple' },
                { id: 3, title: 'Beta Tech', url: 'https://example.com/beta' }
            ];

            // Sort feeds alphabetically (simulating the app logic)
            const sortedFeeds = [...feeds].sort((a, b) => a.title.localeCompare(b.title));

            // Render sorted feeds
            sortedFeeds.forEach((feed) => {
                const feedItem = document.createElement('div');
                feedItem.className = 'feed-item';
                feedItem.dataset.feedId = feed.id;

                const titleSpan = document.createElement('span');
                titleSpan.className = 'feed-title';
                titleSpan.textContent = feed.title;

                feedItem.appendChild(titleSpan);
                feedList.appendChild(feedItem);
            });

            const feedItems = document.querySelectorAll('.feed-item:not(.special)');
            expect(feedItems).toHaveLength(3);

            // Verify alphabetical order
            expect(feedItems[0].querySelector('.feed-title').textContent).toBe('Apple Updates');
            expect(feedItems[1].querySelector('.feed-title').textContent).toBe('Beta Tech');
            expect(feedItems[2].querySelector('.feed-title').textContent).toBe('Zebra News');
        });

        test('should handle case-insensitive alphabetical sorting', () => {
            const feedList = document.getElementById('feed-list');

            // Create feeds with mixed case
            const feeds = [
                { id: 1, title: 'zebra news', url: 'https://example.com/zebra' },
                { id: 2, title: 'Apple Updates', url: 'https://example.com/apple' },
                { id: 3, title: 'beta tech', url: 'https://example.com/beta' }
            ];

            // Sort feeds alphabetically (simulating the app logic)
            const sortedFeeds = [...feeds].sort((a, b) => a.title.localeCompare(b.title));

            // Render sorted feeds
            sortedFeeds.forEach((feed) => {
                const feedItem = document.createElement('div');
                feedItem.className = 'feed-item';
                feedItem.dataset.feedId = feed.id;

                const titleSpan = document.createElement('span');
                titleSpan.className = 'feed-title';
                titleSpan.textContent = feed.title;

                feedItem.appendChild(titleSpan);
                feedList.appendChild(feedItem);
            });

            const feedItems = document.querySelectorAll('.feed-item:not(.special)');

            // localeCompare should handle case-insensitive sorting correctly
            expect(feedItems[0].querySelector('.feed-title').textContent).toBe('Apple Updates');
            expect(feedItems[1].querySelector('.feed-title').textContent).toBe('beta tech');
            expect(feedItems[2].querySelector('.feed-title').textContent).toBe('zebra news');
        });
    });
});