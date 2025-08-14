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
});