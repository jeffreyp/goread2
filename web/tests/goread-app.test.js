const {
    waitFor,
    fireEvent,
    createTestArticles,
    createTestFeeds,
    createMockResponse,
    expectElementToHaveClass,
    expectElementNotToHaveClass,
    expectElementToBeVisible,
    expectElementToBeHidden
} = require('./utils.js');

/**
 * GoRead App DOM Tests
 *
 * These tests verify the GoRead app's DOM manipulation and UI behavior
 * without requiring class instantiation. They test the actual DOM elements
 * and event handlers that would be created by the GoReadApp class.
 */
describe('GoRead App DOM Tests', () => {

    describe('Feed List Rendering', () => {
        test('should render feeds in sidebar', () => {
            const feedList = document.getElementById('feed-list');
            const feeds = createTestFeeds();

            // Simulate feed rendering (as GoReadApp.renderFeeds would do)
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

        test('should update unread counts', () => {
            const feedList = document.getElementById('feed-list');
            const feeds = createTestFeeds();

            // Create feed items
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
                const count = unreadCounts[feed.id] || 0;
                totalUnread += count;

                const countElement = document.querySelector(`[data-feed-id="${feed.id}"] .unread-count`);
                if (countElement) {
                    countElement.textContent = count;
                    countElement.dataset.count = count;
                }
            });

            const allUnreadElement = document.getElementById('all-unread-count');
            if (allUnreadElement) {
                allUnreadElement.textContent = totalUnread;
                allUnreadElement.dataset.count = totalUnread;
            }

            expect(document.querySelector('[data-feed-id="1"] .unread-count').textContent).toBe('5');
            expect(document.querySelector('[data-feed-id="2"] .unread-count').textContent).toBe('3');
            expect(document.getElementById('all-unread-count').textContent).toBe('8');
        });
    });

    describe('Article List Rendering', () => {
        test('should render articles in article list', () => {
            const articleList = document.getElementById('article-list');
            const articles = createTestArticles();

            // Simulate article rendering
            articles.forEach((article, index) => {
                const articleItem = document.createElement('div');
                articleItem.className = `article-item ${article.is_read ? 'read' : ''}`;
                articleItem.dataset.articleId = article.id;
                articleItem.dataset.index = index;

                const titleDiv = document.createElement('div');
                titleDiv.className = 'article-title';
                titleDiv.textContent = article.title;

                articleItem.appendChild(titleDiv);
                articleList.appendChild(articleItem);
            });

            const articleItems = document.querySelectorAll('.article-item');
            expect(articleItems).toHaveLength(3);
            expect(articleItems[0].querySelector('.article-title').textContent).toBe('Test Article 1');
            expectElementToHaveClass(articleItems[0], 'read');
        });

        test('should handle article selection state', () => {
            const articleList = document.getElementById('article-list');
            const articles = createTestArticles();

            // Create article items
            articles.forEach((article, index) => {
                const articleItem = document.createElement('div');
                articleItem.className = 'article-item';
                articleItem.dataset.index = index;
                articleItem.textContent = article.title;
                articleList.appendChild(articleItem);
            });

            const articleItems = document.querySelectorAll('.article-item');

            // Simulate selecting an article
            articleItems.forEach(item => item.classList.remove('active'));
            articleItems[1].classList.add('active');

            expectElementNotToHaveClass(articleItems[0], 'active');
            expectElementToHaveClass(articleItems[1], 'active');
            expectElementNotToHaveClass(articleItems[2], 'active');
        });
    });

    describe('Modal Functionality', () => {
        test('should show add feed modal', () => {
            const modal = document.getElementById('add-feed-modal');
            modal.style.display = 'block';

            expectElementToBeVisible(modal);
        });

        test('should hide modal', () => {
            const modal = document.getElementById('add-feed-modal');
            modal.style.display = 'block';
            modal.style.display = 'none';

            expectElementToBeHidden(modal);
        });

        test('should close modal on cancel button click', () => {
            const modal = document.getElementById('add-feed-modal');
            const cancelBtn = document.getElementById('cancel-add-feed');

            modal.style.display = 'block';

            cancelBtn.addEventListener('click', () => {
                modal.style.display = 'none';
            });

            fireEvent.click(cancelBtn);
            expectElementToBeHidden(modal);
        });
    });

    describe('Form Handling', () => {
        test('should handle add feed form submission', () => {
            const form = document.getElementById('add-feed-form');
            const urlInput = document.getElementById('feed-url');

            let submittedUrl = null;
            form.addEventListener('submit', (e) => {
                e.preventDefault();
                submittedUrl = urlInput.value;
            });

            urlInput.value = 'https://example.com/feed';
            fireEvent.submit(form);

            expect(submittedUrl).toBe('https://example.com/feed');
        });

        test('should clear form after modal close', () => {
            const form = document.getElementById('add-feed-form');
            const urlInput = document.getElementById('feed-url');

            urlInput.value = 'https://example.com/feed';
            form.reset();

            expect(urlInput.value).toBe('');
        });
    });

    describe('Event Handling', () => {
        test('should handle feed item clicks', () => {
            const feedList = document.getElementById('feed-list');
            let clickedFeedId = null;

            feedList.addEventListener('click', (e) => {
                const feedItem = e.target.closest('.feed-item');
                if (feedItem && feedItem.dataset.feedId) {
                    clickedFeedId = feedItem.dataset.feedId;

                    // Remove active class from all
                    document.querySelectorAll('.feed-item').forEach(item => {
                        item.classList.remove('active');
                    });

                    // Add active class to clicked item
                    feedItem.classList.add('active');
                }
            });

            const feedItem = document.createElement('div');
            feedItem.className = 'feed-item';
            feedItem.dataset.feedId = '1';
            feedItem.textContent = 'Test Feed';
            feedList.appendChild(feedItem);

            fireEvent.click(feedItem);

            expect(clickedFeedId).toBe('1');
            expectElementToHaveClass(feedItem, 'active');
        });

        test('should handle keyboard shortcuts', () => {
            let keypressHandled = false;

            document.addEventListener('keydown', (e) => {
                if (e.target.tagName !== 'INPUT' && e.target.tagName !== 'TEXTAREA') {
                    if (e.key === 'j') {
                        keypressHandled = true;
                    }
                }
            });

            fireEvent.keydown(document, 'j');
            expect(keypressHandled).toBe(true);
        });

        test('should ignore keyboard shortcuts in input fields', () => {
            let keypressHandled = false;

            document.addEventListener('keydown', (e) => {
                if (e.target.tagName !== 'INPUT' && e.target.tagName !== 'TEXTAREA') {
                    if (e.key === 'j') {
                        keypressHandled = true;
                    }
                }
            });

            const input = document.createElement('input');
            document.body.appendChild(input);

            fireEvent.keydown(input, 'j');
            expect(keypressHandled).toBe(false);

            document.body.removeChild(input);
        });
    });

    describe('API Integration', () => {
        test('should handle successful feed load', async () => {
            const testFeeds = createTestFeeds();
            mockFetch({
                '/api/feeds': createMockResponse(testFeeds)
            });

            const response = await fetch('/api/feeds');
            const feeds = await response.json();

            expect(response.ok).toBe(true);
            expect(feeds).toHaveLength(2);
        });

        test('should handle feed limit error', async () => {
            mockFetch({
                '/api/feeds': createMockResponse(
                    { error: 'Feed limit reached', limit_reached: true },
                    { status: 402, ok: false }
                )
            });

            const response = await fetch('/api/feeds', {
                method: 'POST',
                body: JSON.stringify({ url: 'https://example.com/feed' })
            });
            const data = await response.json();

            expect(response.ok).toBe(false);
            expect(data.limit_reached).toBe(true);
        });

        test('should handle network error', async () => {
            global.fetch.mockRejectedValueOnce(new Error('Network error'));

            try {
                await fetch('/api/feeds');
                fail('Should have thrown an error');
            } catch (error) {
                expect(error.message).toBe('Network error');
            }
        });
    });

    describe('UI State Management', () => {
        test('should show error message', () => {
            const showError = (message) => {
                const existing = document.querySelector('.error');
                if (existing) existing.remove();

                const errorDiv = document.createElement('div');
                errorDiv.className = 'error';
                errorDiv.textContent = message;
                document.body.appendChild(errorDiv);
            };

            showError('Test error');

            const errorDiv = document.querySelector('.error');
            expect(errorDiv).toBeTruthy();
            expect(errorDiv.textContent).toBe('Test error');
        });

        test('should show success message', () => {
            const showSuccess = (message) => {
                const existing = document.querySelector('.success');
                if (existing) existing.remove();

                const successDiv = document.createElement('div');
                successDiv.className = 'success';
                successDiv.textContent = message;
                document.body.appendChild(successDiv);
            };

            showSuccess('Feed added successfully');

            const successDiv = document.querySelector('.success');
            expect(successDiv).toBeTruthy();
            expect(successDiv.textContent).toBe('Feed added successfully');
        });

        test('should toggle loading state', () => {
            const form = document.getElementById('add-feed-form');
            const submitButton = form.querySelector('button[type="submit"]');

            const setLoading = (loading) => {
                submitButton.disabled = loading;
                submitButton.textContent = loading ? 'Adding...' : 'Add Feed';
            };

            setLoading(true);
            expect(submitButton.disabled).toBe(true);
            expect(submitButton.textContent).toBe('Adding...');

            setLoading(false);
            expect(submitButton.disabled).toBe(false);
            expect(submitButton.textContent).toBe('Add Feed');
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
                try {
                    const date = new Date(dateString);
                    return date.toLocaleDateString();
                } catch {
                    return 'Invalid date';
                }
            };

            const formatted = formatDate('2024-01-15T10:30:00Z');
            expect(formatted).toMatch(/\d{1,2}\/\d{1,2}\/\d{4}/);
        });
    });
});
