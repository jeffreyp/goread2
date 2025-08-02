class GoReadApp {
    constructor() {
        this.currentFeed = null;
        this.currentArticle = null;
        this.feeds = [];
        this.articles = [];
        this.user = null;
        
        this.init();
    }

    async init() {
        await this.checkAuth();
        if (this.user) {
            this.bindEvents();
            this.loadFeeds();
            this.setupKeyboardShortcuts();
            this.showApp();
        } else {
            this.showLogin();
        }
    }

    async checkAuth() {
        try {
            const response = await fetch('/auth/me');
            if (response.ok) {
                const data = await response.json();
                this.user = data.user;
                return true;
            }
        } catch (error) {
            console.log('Not authenticated:', error);
        }
        return false;
    }

    bindEvents() {
        document.getElementById('add-feed-btn').addEventListener('click', () => {
            this.showAddFeedModal();
        });

        document.getElementById('refresh-btn').addEventListener('click', () => {
            this.refreshFeeds();
        });

        document.querySelector('.close').addEventListener('click', () => {
            this.hideAddFeedModal();
        });

        document.getElementById('cancel-add-feed').addEventListener('click', () => {
            this.hideAddFeedModal();
        });

        document.getElementById('add-feed-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.addFeed();
        });

        window.addEventListener('click', (e) => {
            const modal = document.getElementById('add-feed-modal');
            if (e.target === modal) {
                this.hideAddFeedModal();
            }
        });
    }

    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) return;
            
            // Don't handle shortcuts when typing in input fields
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
            
            switch(e.key) {
                case 'j':
                    e.preventDefault();
                    this.selectNextArticle();
                    break;
                case 'k':
                    e.preventDefault();
                    this.selectPreviousArticle();
                    break;
                case 'o':
                case 'Enter':
                    e.preventDefault();
                    this.openCurrentArticle();
                    break;
                case 'm':
                    e.preventDefault();
                    this.toggleCurrentArticleRead();
                    break;
                case 's':
                    e.preventDefault();
                    this.toggleCurrentArticleStar();
                    break;
                case 'r':
                    e.preventDefault();
                    this.refreshFeeds();
                    break;
            }
        });
    }

    async loadFeeds() {
        try {
            const response = await fetch('/api/feeds');
            this.feeds = await response.json();
            this.renderFeeds();
            
            if (this.feeds.length > 0) {
                this.selectFeed('all');
            }
        } catch (error) {
            this.showError('Failed to load feeds: ' + error.message);
        }
    }

    renderFeeds() {
        const feedList = document.getElementById('feed-list');
        const allItem = feedList.querySelector('[data-feed-id="all"]');
        
        feedList.querySelectorAll('.feed-item:not(.special)').forEach(item => item.remove());
        
        this.feeds.forEach(feed => {
            const feedItem = document.createElement('div');
            feedItem.className = 'feed-item';
            feedItem.dataset.feedId = feed.id;
            feedItem.innerHTML = `
                <span class="feed-title">${this.escapeHtml(feed.title)}</span>
                <div style="display: flex; align-items: center;">
                    <span class="unread-count" data-count="0">0</span>
                    <div class="feed-actions">
                        <button class="delete-feed" data-feed-id="${feed.id}" title="Delete feed">×</button>
                    </div>
                </div>
            `;
            
            feedItem.addEventListener('click', (e) => {
                if (!e.target.classList.contains('delete-feed')) {
                    this.selectFeed(feed.id);
                }
            });
            
            feedItem.querySelector('.delete-feed').addEventListener('click', (e) => {
                e.stopPropagation();
                this.deleteFeed(feed.id);
            });
            
            feedList.appendChild(feedItem);
        });

        allItem.addEventListener('click', () => {
            this.selectFeed('all');
        });
        
        this.updateUnreadCounts();
    }

    async selectFeed(feedId) {
        this.currentFeed = feedId;
        
        document.querySelectorAll('.feed-item').forEach(item => {
            item.classList.remove('active');
        });
        
        document.querySelector(`[data-feed-id="${feedId}"]`).classList.add('active');
        
        await this.loadArticles(feedId);
        
        const feedTitle = feedId === 'all' ? 'All Articles' : 
            this.feeds.find(f => f.id == feedId)?.title || 'Unknown Feed';
        document.getElementById('article-pane-title').textContent = feedTitle;
    }

    async loadArticles(feedId) {
        try {
            document.getElementById('article-list').innerHTML = '<div class="loading">Loading articles...</div>';
            
            const url = feedId === 'all' ? '/api/feeds/all/articles' : `/api/feeds/${feedId}/articles`;
            const response = await fetch(url);
            this.articles = await response.json();
            
            this.renderArticles();
        } catch (error) {
            this.showError('Failed to load articles: ' + error.message);
        }
    }

    renderArticles() {
        const articleList = document.getElementById('article-list');
        
        if (this.articles.length === 0) {
            articleList.innerHTML = '<div class="placeholder">No articles found</div>';
            return;
        }
        
        articleList.innerHTML = '';
        
        this.articles.forEach((article, index) => {
            const articleItem = document.createElement('div');
            articleItem.className = `article-item ${article.is_read ? 'read' : ''}`;
            articleItem.dataset.articleId = article.id;
            articleItem.dataset.index = index;
            
            const publishedDate = new Date(article.published_at).toLocaleDateString();
            
            articleItem.innerHTML = `
                <div class="article-header">
                    <div style="flex: 1;">
                        <div class="article-title">${this.escapeHtml(article.title)}</div>
                        <div class="article-meta">
                            <span>${publishedDate}</span>
                            ${article.author ? `<span>by ${this.escapeHtml(article.author)}</span>` : ''}
                        </div>
                    </div>
                    <div class="article-actions">
                        <button class="action-btn star-btn ${article.is_starred ? 'starred' : ''}" 
                                data-article-id="${article.id}" title="Star article">★</button>
                    </div>
                </div>
                ${article.description ? `<div class="article-description">${this.escapeHtml(article.description)}</div>` : ''}
            `;
            
            articleItem.addEventListener('click', () => {
                this.selectArticle(index);
            });
            
            articleItem.querySelector('.star-btn').addEventListener('click', (e) => {
                e.stopPropagation();
                this.toggleStar(article.id);
            });
            
            articleList.appendChild(articleItem);
        });
    }

    selectArticle(index) {
        this.currentArticle = index;
        
        document.querySelectorAll('.article-item').forEach(item => {
            item.classList.remove('active');
        });
        
        const articleItem = document.querySelector(`[data-index="${index}"]`);
        articleItem.classList.add('active');
        
        const article = this.articles[index];
        this.displayArticle(article);
        
        if (!article.is_read) {
            this.markAsRead(article.id, true);
            articleItem.classList.add('read');
            article.is_read = true;
            this.updateUnreadCounts();
        }
        
        articleItem.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }

    displayArticle(article) {
        const contentPane = document.getElementById('article-content');
        const publishedDate = new Date(article.published_at).toLocaleString();
        
        contentPane.innerHTML = `
            <h1>${this.escapeHtml(article.title)}</h1>
            <div class="meta">
                <span>${publishedDate}</span>
                ${article.author ? `<span>by ${this.escapeHtml(article.author)}</span>` : ''}
                <a href="${article.url}" target="_blank" rel="noopener">View Original</a>
            </div>
            <div class="content">
                ${article.content || article.description || '<p>No content available.</p>'}
            </div>
        `;
        
        document.getElementById('content-pane-title').textContent = this.escapeHtml(article.title);
    }

    selectNextArticle() {
        if (this.currentArticle === null || this.articles.length === 0) return;
        
        const nextIndex = Math.min(this.currentArticle + 1, this.articles.length - 1);
        if (nextIndex !== this.currentArticle) {
            this.selectArticle(nextIndex);
        }
    }

    selectPreviousArticle() {
        if (this.currentArticle === null || this.articles.length === 0) return;
        
        const prevIndex = Math.max(this.currentArticle - 1, 0);
        if (prevIndex !== this.currentArticle) {
            this.selectArticle(prevIndex);
        }
    }

    openCurrentArticle() {
        if (this.currentArticle === null) return;
        
        const article = this.articles[this.currentArticle];
        window.open(article.url, '_blank');
    }

    toggleCurrentArticleRead() {
        if (this.currentArticle === null) return;
        
        const article = this.articles[this.currentArticle];
        this.markAsRead(article.id, !article.is_read);
        
        article.is_read = !article.is_read;
        const articleItem = document.querySelector(`[data-index="${this.currentArticle}"]`);
        articleItem.classList.toggle('read', article.is_read);
        
        this.updateUnreadCounts();
    }

    toggleCurrentArticleStar() {
        if (this.currentArticle === null) return;
        
        const article = this.articles[this.currentArticle];
        this.toggleStar(article.id);
    }

    async markAsRead(articleId, isRead) {
        try {
            await fetch(`/api/articles/${articleId}/read`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ is_read: isRead })
            });
        } catch (error) {
            console.error('Failed to mark article as read:', error);
        }
    }

    async toggleStar(articleId) {
        try {
            await fetch(`/api/articles/${articleId}/star`, {
                method: 'POST'
            });
            
            const article = this.articles.find(a => a.id == articleId);
            if (article) {
                article.is_starred = !article.is_starred;
                const starBtn = document.querySelector(`[data-article-id="${articleId}"]`);
                starBtn.classList.toggle('starred', article.is_starred);
            }
        } catch (error) {
            console.error('Failed to toggle star:', error);
        }
    }

    updateUnreadCounts() {
        const allUnreadCount = this.articles.filter(a => !a.is_read).length;
        document.getElementById('all-unread-count').textContent = allUnreadCount;
        document.getElementById('all-unread-count').dataset.count = allUnreadCount;
        
        this.feeds.forEach(feed => {
            const unreadCount = this.articles.filter(a => a.feed_id === feed.id && !a.is_read).length;
            const countElement = document.querySelector(`[data-feed-id="${feed.id}"] .unread-count`);
            if (countElement) {
                countElement.textContent = unreadCount;
                countElement.dataset.count = unreadCount;
            }
        });
    }

    showAddFeedModal() {
        document.getElementById('add-feed-modal').style.display = 'block';
        document.getElementById('feed-url').focus();
    }

    hideAddFeedModal() {
        document.getElementById('add-feed-modal').style.display = 'none';
        document.getElementById('add-feed-form').reset();
    }

    async addFeed() {
        const url = document.getElementById('feed-url').value;
        console.log('Adding feed with URL:', url);
        
        try {
            console.log('Sending request to /api/feeds');
            const response = await fetch('/api/feeds', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url })
            });
            
            console.log('Response status:', response.status);
            console.log('Response ok:', response.ok);
            
            if (response.ok) {
                console.log('Feed added successfully');
                this.hideAddFeedModal();
                await this.loadFeeds();
            } else {
                let errorMessage = `HTTP ${response.status}`;
                try {
                    const error = await response.json();
                    console.log('Server error:', error);
                    errorMessage = error.error || errorMessage;
                } catch (e) {
                    console.log('Could not parse error response');
                }
                this.showError('Failed to add feed: ' + errorMessage);
            }
        } catch (error) {
            console.log('Network error:', error);
            this.showError('Failed to add feed: ' + error.message);
        }
    }

    async deleteFeed(feedId) {
        if (!confirm('Are you sure you want to delete this feed?')) return;
        
        try {
            await fetch(`/api/feeds/${feedId}`, {
                method: 'DELETE'
            });
            
            await this.loadFeeds();
            
            if (this.currentFeed == feedId) {
                this.selectFeed('all');
            }
        } catch (error) {
            this.showError('Failed to delete feed: ' + error.message);
        }
    }

    async refreshFeeds() {
        try {
            await fetch('/api/feeds/refresh', { method: 'POST' });
            
            if (this.currentFeed) {
                await this.loadArticles(this.currentFeed);
            }
            
            this.updateUnreadCounts();
        } catch (error) {
            console.error('Failed to refresh feeds:', error);
        }
    }

    showError(message) {
        const existingError = document.querySelector('.error');
        if (existingError) {
            existingError.remove();
        }
        
        const errorDiv = document.createElement('div');
        errorDiv.className = 'error';
        errorDiv.textContent = message;
        
        document.body.appendChild(errorDiv);
        
        setTimeout(() => {
            errorDiv.remove();
        }, 5000);
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Authentication methods
    showLogin() {
        document.getElementById('app').style.display = 'none';
        this.showLoginScreen();
    }

    showApp() {
        document.getElementById('login-screen')?.remove();
        document.getElementById('app').style.display = 'block';
        this.updateUserInfo();
    }

    showLoginScreen() {
        const loginScreen = document.createElement('div');
        loginScreen.id = 'login-screen';
        loginScreen.innerHTML = `
            <div class="login-container">
                <h1>GoRead2</h1>
                <p>Sign in with Google to access your RSS feeds</p>
                <button id="google-login-btn" class="btn btn-primary">Sign in with Google</button>
            </div>
        `;
        document.body.appendChild(loginScreen);

        document.getElementById('google-login-btn').addEventListener('click', () => {
            this.login();
        });
    }

    async login() {
        try {
            const response = await fetch('/auth/login');
            if (response.ok) {
                const data = await response.json();
                window.location.href = data.auth_url;
            } else {
                this.showError('Failed to start login process');
            }
        } catch (error) {
            this.showError('Login failed: ' + error.message);
        }
    }

    async logout() {
        try {
            await fetch('/auth/logout', { method: 'POST' });
            this.user = null;
            this.showLogin();
        } catch (error) {
            this.showError('Logout failed: ' + error.message);
        }
    }

    updateUserInfo() {
        if (this.user) {
            // Update header with user info
            const headerActions = document.querySelector('.header-actions');
            const userInfo = document.createElement('div');
            userInfo.className = 'user-info';
            userInfo.innerHTML = `
                <span class="user-name">${this.escapeHtml(this.user.name)}</span>
                <img class="user-avatar" src="${this.user.avatar}" alt="User Avatar" width="32" height="32">
                <button id="logout-btn" class="btn btn-secondary">Logout</button>
            `;
            headerActions.appendChild(userInfo);

            document.getElementById('logout-btn').addEventListener('click', () => {
                this.logout();
            });
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    new GoReadApp();
});