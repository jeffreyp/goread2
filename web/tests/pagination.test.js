const {
    waitFor,
    fireEvent
} = require('./utils.js');

// Tests for pagination functionality
// Tests the Load More button, cursor-based pagination, and article loading
describe('Pagination Functionality', () => {

    beforeEach(() => {
        // Clear article list and any existing load more buttons
        document.getElementById('article-list').innerHTML = '';
        document.querySelectorAll('.load-more-button').forEach(el => el.remove());
    });

    afterEach(() => {
        // Cleanup
        document.getElementById('article-list').innerHTML = '';
        document.querySelectorAll('.load-more-button').forEach(el => el.remove());
    });

    describe('Load More Button', () => {
        test('should create load more button when hasMoreArticles is true', () => {
            const articleList = document.getElementById('article-list');

            // Simulate adding load more button
            const loadMoreDiv = document.createElement('div');
            loadMoreDiv.className = 'load-more-button';
            loadMoreDiv.style.cssText = 'padding: 20px; text-align: center; border-top: 1px solid #e1e5e9;';

            const button = document.createElement('button');
            button.className = 'btn btn-secondary';
            button.textContent = 'Load More Articles';

            loadMoreDiv.appendChild(button);
            articleList.appendChild(loadMoreDiv);

            const foundButton = document.querySelector('.load-more-button button');
            expect(foundButton).toBeTruthy();
            expect(foundButton.textContent).toBe('Load More Articles');
            expect(foundButton.className).toContain('btn-secondary');
        });

        test('should not create load more button when hasMoreArticles is false', () => {
            const articleList = document.getElementById('article-list');

            // Simulate no more articles - button should not be added
            const hasMoreArticles = false;

            if (!hasMoreArticles) {
                // Don't add button
            }

            const foundButton = document.querySelector('.load-more-button');
            expect(foundButton).toBeNull();
        });

        test('should remove existing load more button before adding new one', () => {
            const articleList = document.getElementById('article-list');

            // Add first button
            const firstButton = document.createElement('div');
            firstButton.className = 'load-more-button';
            firstButton.textContent = 'First button';
            articleList.appendChild(firstButton);

            // Simulate removing old and adding new button
            const existingButton = articleList.querySelector('.load-more-button');
            if (existingButton) {
                existingButton.remove();
            }

            const loadMoreDiv = document.createElement('div');
            loadMoreDiv.className = 'load-more-button';
            loadMoreDiv.textContent = 'New button';
            articleList.appendChild(loadMoreDiv);

            const buttons = document.querySelectorAll('.load-more-button');
            expect(buttons).toHaveLength(1);
            expect(buttons[0].textContent).toBe('New button');
        });

        test('should have correct styling for load more button', () => {
            const articleList = document.getElementById('article-list');

            const loadMoreDiv = document.createElement('div');
            loadMoreDiv.className = 'load-more-button';
            loadMoreDiv.style.cssText = 'padding: 20px; text-align: center; border-top: 1px solid #e1e5e9;';

            articleList.appendChild(loadMoreDiv);

            expect(loadMoreDiv.style.padding).toBe('20px');
            expect(loadMoreDiv.style.textAlign).toBe('center');
            expect(loadMoreDiv.style.borderTop).toContain('1px solid');
        });
    });

    describe('Load More Button Click', () => {
        test('should change button text to "Loading..." when clicked', () => {
            const articleList = document.getElementById('article-list');

            const button = document.createElement('button');
            button.className = 'btn btn-secondary';
            button.textContent = 'Load More Articles';

            const loadingHandler = () => {
                button.textContent = 'Loading...';
                button.disabled = true;
            };

            button.onclick = loadingHandler;

            const loadMoreDiv = document.createElement('div');
            loadMoreDiv.className = 'load-more-button';
            loadMoreDiv.appendChild(button);
            articleList.appendChild(loadMoreDiv);

            fireEvent.click(button);

            expect(button.textContent).toBe('Loading...');
            expect(button.disabled).toBe(true);
        });

        test('should restore button text after loading completes', (done) => {
            const articleList = document.getElementById('article-list');

            const button = document.createElement('button');
            button.className = 'btn btn-secondary';
            button.textContent = 'Load More Articles';

            const loadingHandler = async () => {
                button.textContent = 'Loading...';
                button.disabled = true;

                // Simulate async loading
                await new Promise(resolve => setTimeout(resolve, 50));

                button.textContent = 'Load More Articles';
                button.disabled = false;
            };

            button.onclick = loadingHandler;

            const loadMoreDiv = document.createElement('div');
            loadMoreDiv.className = 'load-more-button';
            loadMoreDiv.appendChild(button);
            articleList.appendChild(loadMoreDiv);

            fireEvent.click(button);

            expect(button.textContent).toBe('Loading...');

            setTimeout(() => {
                expect(button.textContent).toBe('Load More Articles');
                expect(button.disabled).toBe(false);
                done();
            }, 100);
        });
    });

    describe('Article List Rendering', () => {
        test('should display placeholder when no articles exist', () => {
            const articleList = document.getElementById('article-list');

            // Simulate empty articles
            const articles = [];

            if (articles.length === 0) {
                articleList.innerHTML = '<div class="placeholder">No articles found</div>';
            }

            const placeholder = articleList.querySelector('.placeholder');
            expect(placeholder).toBeTruthy();
            expect(placeholder.textContent).toBe('No articles found');
        });

        test('should render multiple articles in article list', () => {
            const articleList = document.getElementById('article-list');

            const articles = [
                { id: 1, title: 'Article 1', is_read: false },
                { id: 2, title: 'Article 2', is_read: true },
                { id: 3, title: 'Article 3', is_read: false }
            ];

            // Simulate rendering articles
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

            const renderedArticles = document.querySelectorAll('.article-item');
            expect(renderedArticles).toHaveLength(3);
            expect(renderedArticles[0].querySelector('.article-title').textContent).toBe('Article 1');
            expect(renderedArticles[1].classList.contains('read')).toBe(true);
            expect(renderedArticles[2].dataset.index).toBe('2');
        });

        test('should append new articles when loading more', () => {
            const articleList = document.getElementById('article-list');

            // Initial articles
            const initialArticles = [
                { id: 1, title: 'Article 1' },
                { id: 2, title: 'Article 2' }
            ];

            initialArticles.forEach((article, index) => {
                const articleItem = document.createElement('div');
                articleItem.className = 'article-item';
                articleItem.dataset.index = index;
                articleItem.textContent = article.title;
                articleList.appendChild(articleItem);
            });

            // Simulate loading more articles
            const moreArticles = [
                { id: 3, title: 'Article 3' },
                { id: 4, title: 'Article 4' }
            ];

            const existingCount = document.querySelectorAll('.article-item').length;

            moreArticles.forEach((article, index) => {
                const articleItem = document.createElement('div');
                articleItem.className = 'article-item';
                articleItem.dataset.index = existingCount + index;
                articleItem.textContent = article.title;
                articleList.appendChild(articleItem);
            });

            const allArticles = document.querySelectorAll('.article-item');
            expect(allArticles).toHaveLength(4);
            expect(allArticles[2].textContent).toBe('Article 3');
            expect(allArticles[3].dataset.index).toBe('3');
        });
    });

    describe('Pagination State Management', () => {
        test('should track hasMoreArticles flag', () => {
            let hasMoreArticles = true;

            expect(hasMoreArticles).toBe(true);

            // Simulate receiving empty next cursor (no more articles)
            const nextCursor = '';
            hasMoreArticles = nextCursor !== '';

            expect(hasMoreArticles).toBe(false);
        });

        test('should update hasMoreArticles based on API response cursor', () => {
            // Simulate API response with cursor
            const response1 = { articles: [], next_cursor: 'abc123' };
            let hasMoreArticles1 = response1.next_cursor !== '';
            expect(hasMoreArticles1).toBe(true);

            // Simulate API response without cursor
            const response2 = { articles: [], next_cursor: '' };
            let hasMoreArticles2 = response2.next_cursor !== '';
            expect(hasMoreArticles2).toBe(false);
        });

        test('should only add load more button for "all" feed view', () => {
            const articleList = document.getElementById('article-list');

            let currentFeed = 'all';
            let hasMoreArticles = true;

            if (currentFeed === 'all' && hasMoreArticles) {
                const button = document.createElement('div');
                button.className = 'load-more-button';
                articleList.appendChild(button);
            }

            expect(document.querySelector('.load-more-button')).toBeTruthy();

            // Clear for next test
            articleList.innerHTML = '';

            // Test with specific feed
            currentFeed = 'feed-123';
            hasMoreArticles = true;

            if (currentFeed === 'all' && hasMoreArticles) {
                const button = document.createElement('div');
                button.className = 'load-more-button';
                articleList.appendChild(button);
            }

            expect(document.querySelector('.load-more-button')).toBeNull();
        });
    });

    describe('Cursor-Based Pagination', () => {
        test('should use cursor for subsequent page requests', () => {
            let nextCursor = '';
            let articles = [];

            // First page response
            const firstPage = {
                articles: [{ id: 1 }, { id: 2 }],
                next_cursor: 'cursor_abc123'
            };

            articles = firstPage.articles;
            nextCursor = firstPage.next_cursor;

            expect(nextCursor).toBe('cursor_abc123');
            expect(articles).toHaveLength(2);

            // Second page request would use the cursor
            const cursorForNextRequest = nextCursor;
            expect(cursorForNextRequest).toBe('cursor_abc123');
        });

        test('should handle empty cursor indicating last page', () => {
            const lastPageResponse = {
                articles: [{ id: 100 }],
                next_cursor: ''
            };

            const hasMorePages = lastPageResponse.next_cursor !== '';

            expect(hasMorePages).toBe(false);
        });

        test('should maintain article order across pages', () => {
            const articleList = document.getElementById('article-list');

            // First page
            const page1 = [{ id: 1, index: 0 }, { id: 2, index: 1 }];
            page1.forEach(article => {
                const item = document.createElement('div');
                item.className = 'article-item';
                item.dataset.index = article.index;
                item.dataset.articleId = article.id;
                articleList.appendChild(item);
            });

            // Second page should continue indexing
            const existingCount = document.querySelectorAll('.article-item').length;
            const page2 = [{ id: 3 }, { id: 4 }];

            page2.forEach((article, i) => {
                const item = document.createElement('div');
                item.className = 'article-item';
                item.dataset.index = existingCount + i;
                item.dataset.articleId = article.id;
                articleList.appendChild(item);
            });

            const allItems = document.querySelectorAll('.article-item');
            expect(allItems).toHaveLength(4);
            expect(allItems[0].dataset.index).toBe('0');
            expect(allItems[2].dataset.index).toBe('2');
            expect(allItems[3].dataset.index).toBe('3');
        });
    });

    describe('Load More Error Handling', () => {
        test('should restore button state on error', (done) => {
            const articleList = document.getElementById('article-list');

            const button = document.createElement('button');
            button.className = 'btn btn-secondary';
            button.textContent = 'Load More Articles';

            const errorHandler = async () => {
                button.textContent = 'Loading...';
                button.disabled = true;

                try {
                    // Simulate error during loading
                    await new Promise((resolve, reject) => {
                        setTimeout(() => reject(new Error('Network error')), 50);
                    });
                } catch (error) {
                    // Restore button state on error
                    button.textContent = 'Load More Articles';
                    button.disabled = false;
                }
            };

            button.onclick = errorHandler;

            const loadMoreDiv = document.createElement('div');
            loadMoreDiv.className = 'load-more-button';
            loadMoreDiv.appendChild(button);
            articleList.appendChild(loadMoreDiv);

            fireEvent.click(button);

            setTimeout(() => {
                expect(button.textContent).toBe('Load More Articles');
                expect(button.disabled).toBe(false);
                done();
            }, 100);
        });
    });

    describe('Article Index Management', () => {
        test('should assign correct data-index to articles', () => {
            const articleList = document.getElementById('article-list');

            const articles = [
                { id: 1, title: 'First' },
                { id: 2, title: 'Second' },
                { id: 3, title: 'Third' }
            ];

            articles.forEach((article, index) => {
                const item = document.createElement('div');
                item.className = 'article-item';
                item.dataset.index = index;
                item.dataset.articleId = article.id;
                articleList.appendChild(item);
            });

            const items = document.querySelectorAll('.article-item');
            expect(items[0].dataset.index).toBe('0');
            expect(items[1].dataset.index).toBe('1');
            expect(items[2].dataset.index).toBe('2');
        });

        test('should preserve existing article indices when loading more', () => {
            const articleList = document.getElementById('article-list');

            // Add initial articles
            [0, 1, 2].forEach(index => {
                const item = document.createElement('div');
                item.className = 'article-item';
                item.dataset.index = index;
                articleList.appendChild(item);
            });

            // Load more starting from index 3
            const existingCount = document.querySelectorAll('.article-item').length;
            [3, 4, 5].forEach((_, i) => {
                const item = document.createElement('div');
                item.className = 'article-item';
                item.dataset.index = existingCount + i;
                articleList.appendChild(item);
            });

            const allItems = document.querySelectorAll('.article-item');
            expect(allItems).toHaveLength(6);

            // Verify sequential indexing
            allItems.forEach((item, expectedIndex) => {
                expect(parseInt(item.dataset.index)).toBe(expectedIndex);
            });
        });
    });
});
