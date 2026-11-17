const fs = require('fs');
const path = require('path');

// Loads the real GoReadApp class and runs its Load More methods against a
// stand-in for the app instance, so no page has to be booted.
const source = fs.readFileSync(path.join(__dirname, '../static/js/app.js'), 'utf8');
const GoReadApp = new Function(`${source}\nreturn GoReadApp;`)();

function createApp(overrides = {}) {
    const app = Object.create(GoReadApp.prototype);
    return Object.assign(app, {
        currentFeed: 'all',
        articleFilter: 'unread',
        articles: [],
        nextCursor: '',
        lastPageWasShort: false,
        unreadCountsLoaded: true,
        newArticlesBaselineCount: 0,
        hasMoreArticles: false
    }, overrides);
}

function articles(count) {
    return Array.from({ length: count }, (_, i) => ({ id: i + 1, is_read: false }));
}

describe('Load More decision', () => {
    beforeEach(() => {
        document.getElementById('article-list').innerHTML = '';
    });

    describe('moreArticlesExpected', () => {
        test('is false without a cursor', () => {
            const app = createApp({ articles: articles(50), newArticlesBaselineCount: 80 });
            expect(app.moreArticlesExpected()).toBe(false);
        });

        test('trusts the cursor after a full page', () => {
            const app = createApp({ nextCursor: 'c', articles: articles(50), newArticlesBaselineCount: 50 });
            expect(app.moreArticlesExpected()).toBe(true);
        });

        test('is false after a short unread page once every counted unread article is loaded', () => {
            const app = createApp({
                nextCursor: 'c', lastPageWasShort: true,
                articles: articles(12), newArticlesBaselineCount: 12
            });
            expect(app.moreArticlesExpected()).toBe(false);
        });

        test('is true after a short unread page while the unread count exceeds what is loaded', () => {
            const app = createApp({
                nextCursor: 'c', lastPageWasShort: true,
                articles: articles(12), newArticlesBaselineCount: 20
            });
            expect(app.moreArticlesExpected()).toBe(true);
        });

        test('trusts the cursor in the all-articles filter', () => {
            const app = createApp({
                nextCursor: 'c', lastPageWasShort: true, articleFilter: 'all',
                articles: articles(12), newArticlesBaselineCount: 12
            });
            expect(app.moreArticlesExpected()).toBe(true);
        });
    });

    test('the button is removed when unread counts arrive after the article list', () => {
        // Startup order seen in production: the list loads first, so the
        // decision is made before the counts exist.
        const app = createApp({
            nextCursor: 'c', lastPageWasShort: true,
            articles: articles(12), unreadCountsLoaded: false
        });
        app.refreshLoadMoreButton();
        expect(document.querySelector('.load-more-button')).not.toBeNull();

        app.unreadCountsLoaded = true;
        app.newArticlesBaselineCount = 12;
        app.refreshLoadMoreButton();

        expect(app.hasMoreArticles).toBe(false);
        expect(document.querySelector('.load-more-button')).toBeNull();
    });

    describe('loadMoreArticles', () => {
        function appWithPages(pages) {
            const app = createApp({
                nextCursor: 'c', hasMoreArticles: true,
                articles: articles(12), newArticlesBaselineCount: 20
            });
            app.addLoadMoreButton();
            app.loadArticles = jest.fn(async () => {
                const page = pages.shift();
                if (page === 'error') {
                    return false;
                }
                app.articles.push(...page.articles);
                app.nextCursor = page.cursor;
                app.hasMoreArticles = page.cursor !== '';
                app.addLoadMoreButton();
                return true;
            });
            return app;
        }

        test('withdraws the button when every followed page comes back empty', async () => {
            const empty = () => ({ articles: [], cursor: 'c' });
            const app = appWithPages(Array.from({ length: 10 }, empty));

            await app.loadMoreArticles();

            expect(app.loadArticles).toHaveBeenCalledTimes(GoReadApp.maxEmptyLoadMoreRounds);
            expect(app.hasMoreArticles).toBe(false);
            expect(document.querySelector('.load-more-button')).toBeNull();
        });

        test('keeps following empty pages until one has articles', async () => {
            const app = appWithPages([
                { articles: [], cursor: 'c' },
                { articles: [], cursor: 'c' },
                { articles: [{ id: 100, is_read: false }], cursor: 'd' }
            ]);

            await app.loadMoreArticles();

            expect(app.loadArticles).toHaveBeenCalledTimes(3);
            expect(app.articles).toHaveLength(13);
            expect(app.hasMoreArticles).toBe(true);
            expect(document.querySelector('.load-more-button')).not.toBeNull();
        });

        test('stops at the end of the list', async () => {
            const app = appWithPages([{ articles: [], cursor: '' }]);

            await app.loadMoreArticles();

            expect(app.loadArticles).toHaveBeenCalledTimes(1);
            expect(document.querySelector('.load-more-button')).toBeNull();
        });

        test('keeps the button when the request fails', async () => {
            const app = appWithPages(['error']);

            await app.loadMoreArticles();

            expect(app.loadArticles).toHaveBeenCalledTimes(1);
            expect(app.hasMoreArticles).toBe(true);
            const button = document.querySelector('.load-more-button button');
            expect(button).not.toBeNull();
            expect(button.disabled).toBe(false);
        });
    });
});
