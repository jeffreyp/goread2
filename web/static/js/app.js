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
            // Not authenticated
        }
        return false;
    }

    bindEvents() {
        const addFeedBtn = document.getElementById('add-feed-btn');
        if (addFeedBtn) {
            addFeedBtn.addEventListener('click', () => {
                this.showAddFeedModal();
            });
        }

        const refreshBtn = document.getElementById('refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.refreshFeeds();
            });
        }

        const helpBtn = document.getElementById('help-btn');
        if (helpBtn) {
            helpBtn.addEventListener('click', () => {
                this.showHelpModal();
            });
        }

        // Handle close buttons for both modals
        document.querySelectorAll('.close').forEach(closeBtn => {
            closeBtn.addEventListener('click', (e) => {
                const modal = e.target.closest('.modal');
                if (modal.id === 'add-feed-modal') {
                    this.hideAddFeedModal();
                } else if (modal.id === 'help-modal') {
                    this.hideHelpModal();
                }
            });
        });

        const cancelBtn = document.getElementById('cancel-add-feed');
        if (cancelBtn) {
            cancelBtn.addEventListener('click', () => {
                this.hideAddFeedModal();
            });
        }

        const addFeedForm = document.getElementById('add-feed-form');
        if (addFeedForm) {
            addFeedForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.addFeed();
            });
        }

        window.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                if (e.target.id === 'add-feed-modal') {
                    this.hideAddFeedModal();
                } else if (e.target.id === 'help-modal') {
                    this.hideHelpModal();
                }
            }
        });

        // Set up "Articles" click listener
        const allItem = document.querySelector('[data-feed-id="all"]');
        if (allItem) {
            allItem.addEventListener('click', () => {
                this.selectFeed('all');
            });
        }
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
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
            
            const feedsData = await response.json();
            this.feeds = feedsData;
            
            if (Array.isArray(this.feeds)) {
                this.renderFeeds();
                await this.updateUnreadCounts();
                
                if (this.feeds.length > 0) {
                    this.selectFeed('all');
                }
            } else {
                this.showError('Invalid feed data received from server');
            }
        } catch (error) {
            console.error('Failed to load feeds:', error);
            this.showError('Failed to load feeds: ' + error.message);
        }
    }

    renderFeeds() {
        if (!Array.isArray(this.feeds)) {
            return;
        }
        
        const feedList = document.getElementById('feed-list');
        
        // Remove existing feed items (not the "All" item)
        const existingFeeds = feedList.querySelectorAll('.feed-item:not(.special)');
        existingFeeds.forEach(item => item.remove());
        
        this.feeds.forEach((feed) => {
            // Create main container
            const feedItem = document.createElement('div');
            feedItem.className = 'feed-item';
            feedItem.dataset.feedId = feed.id;
            
            // Create title
            const titleSpan = document.createElement('span');
            titleSpan.className = 'feed-title';
            titleSpan.textContent = feed.title;
            
            // Create right side container
            const rightDiv = document.createElement('div');
            rightDiv.style.display = 'flex';
            rightDiv.style.alignItems = 'center';
            
            // Create unread count
            const unreadSpan = document.createElement('span');
            unreadSpan.className = 'unread-count';
            unreadSpan.dataset.count = '0';
            unreadSpan.textContent = '0';
            
            // Create actions container
            const actionsDiv = document.createElement('div');
            actionsDiv.className = 'feed-actions';
            
            // Create delete button
            const deleteButton = document.createElement('button');
            deleteButton.className = 'delete-feed';
            deleteButton.dataset.feedId = feed.id;
            deleteButton.title = 'Delete feed';
            deleteButton.textContent = '×';
            
            // Assemble structure
            actionsDiv.appendChild(deleteButton);
            rightDiv.appendChild(unreadSpan);
            rightDiv.appendChild(actionsDiv);
            feedItem.appendChild(titleSpan);
            feedItem.appendChild(rightDiv);
            
            // Add event listeners
            feedItem.addEventListener('click', (e) => {
                if (!e.target.classList.contains('delete-feed')) {
                    this.selectFeed(feed.id);
                }
            });
            
            deleteButton.addEventListener('click', (e) => {
                e.stopPropagation();
                this.deleteFeed(feed.id);
            });
            
            feedList.appendChild(feedItem);
        });
    }

    async selectFeed(feedId) {
        this.currentFeed = feedId;
        
        document.querySelectorAll('.feed-item').forEach(item => {
            item.classList.remove('active');
        });
        
        document.querySelector(`[data-feed-id="${feedId}"]`).classList.add('active');
        
        await this.loadArticles(feedId);
        
        const feedTitle = feedId === 'all' ? 'Articles' : 
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
            await this.updateUnreadCounts();
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
        
        // Remove existing event listeners by cloning the element
        const newArticleList = articleList.cloneNode(false);
        articleList.parentNode.replaceChild(newArticleList, articleList);
        // Update reference
        const updatedArticleList = document.getElementById('article-list');
        
        // Add event delegation for star buttons and article selection
        updatedArticleList.addEventListener('click', (e) => {
            // Handle star button clicks via event delegation
            if (e.target.classList.contains('star-btn')) {
                e.stopPropagation();
                e.preventDefault();
                this.toggleStar(parseInt(e.target.dataset.articleId));
                return;
            }
            
            // Handle article item clicks
            const articleItem = e.target.closest('.article-item');
            if (articleItem && !e.target.classList.contains('star-btn')) {
                const index = parseInt(articleItem.dataset.index);
                this.selectArticle(index);
            }
        });
        
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
            
            updatedArticleList.appendChild(articleItem);
        });
        
        // Auto-select the first article if articles exist
        if (this.articles.length > 0) {
            this.selectArticle(0);
        }
    }

    async selectArticle(index) {
        // Mark previous article as read if we're navigating away from one
        if (this.currentArticle !== null && this.currentArticle !== index) {
            await this.markCurrentArticleAsReadIfUnread();
        }
        
        this.currentArticle = index;
        
        document.querySelectorAll('.article-item').forEach(item => {
            item.classList.remove('active');
        });
        
        const articleItem = document.querySelector(`[data-index="${index}"]`);
        articleItem.classList.add('active');
        
        const article = this.articles[index];
        this.displayArticle(article);
        
        articleItem.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }
    
    async markCurrentArticleAsReadIfUnread() {
        if (this.currentArticle !== null && this.articles[this.currentArticle]) {
            const article = this.articles[this.currentArticle];
            if (!article.is_read) {
                this.markAsRead(article.id, true);
                const articleItem = document.querySelector(`[data-index="${this.currentArticle}"]`);
                if (articleItem) {
                    articleItem.classList.add('read');
                }
                article.is_read = true;
                await this.updateUnreadCounts();
            }
        }
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

    async toggleCurrentArticleRead() {
        if (this.currentArticle === null) return;
        
        const article = this.articles[this.currentArticle];
        this.markAsRead(article.id, !article.is_read);
        
        article.is_read = !article.is_read;
        const articleItem = document.querySelector(`[data-index="${this.currentArticle}"]`);
        articleItem.classList.toggle('read', article.is_read);
        
        await this.updateUnreadCounts();
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
            const response = await fetch(`/api/articles/${articleId}/star`, {
                method: 'POST'
            });
            
            if (!response.ok) {
                this.showError('Failed to toggle star');
                return;
            }
            
            const article = this.articles.find(a => a.id == articleId);
            if (article) {
                article.is_starred = !article.is_starred;
                const starBtn = document.querySelector(`.star-btn[data-article-id="${articleId}"]`);
                if (starBtn) {
                    starBtn.classList.toggle('starred', article.is_starred);
                }
            }
        } catch (error) {
            console.error('Failed to toggle star:', error);
            this.showError('Failed to toggle star: ' + error.message);
        }
    }

    async updateUnreadCounts() {
        try {
            const response = await fetch('/api/feeds/unread-counts');
            if (!response.ok) {
                console.error(`Failed to fetch unread counts: HTTP ${response.status}`);
                if (response.status === 401) {
                    console.error('Authentication failed for unread counts');
                }
                return;
            }
            
            const unreadCounts = await response.json();
            console.log('Received unread counts:', unreadCounts);
            
            // Update individual feed counts
            let totalUnread = 0;
            this.feeds.forEach(feed => {
                const unreadCount = unreadCounts[feed.id] || unreadCounts[feed.id.toString()] || 0;
                totalUnread += unreadCount;
                
                const countElement = document.querySelector(`[data-feed-id="${feed.id}"] .unread-count`);
                if (countElement) {
                    countElement.textContent = unreadCount;
                    countElement.dataset.count = unreadCount;
                } else {
                    console.warn(`Count element not found for feed ${feed.id}`);
                }
            });
            
            // Update "Articles" count
            const allUnreadElement = document.getElementById('all-unread-count');
            if (allUnreadElement) {
                allUnreadElement.textContent = totalUnread;
                allUnreadElement.dataset.count = totalUnread;
            } else {
                console.warn('All unread count element not found');
            }
            
            console.log(`Updated unread counts - total: ${totalUnread}`);
        } catch (error) {
            console.error('Error updating unread counts:', error);
        }
    }

    showAddFeedModal() {
        document.getElementById('add-feed-modal').style.display = 'block';
        document.getElementById('feed-url').focus();
    }

    hideAddFeedModal() {
        const modal = document.getElementById('add-feed-modal');
        const form = document.getElementById('add-feed-form');
        const submitButton = form.querySelector('button[type="submit"]');
        const cancelButton = document.getElementById('cancel-add-feed');
        const inputField = document.getElementById('feed-url');
        
        // Reset all form controls if they were in loading state
        const spinnerOverlay = submitButton.querySelector('.button-spinner-overlay');
        if (spinnerOverlay) {
            if (spinnerOverlay.stopAnimation) {
                spinnerOverlay.stopAnimation();
            }
            spinnerOverlay.remove();
        }
        submitButton.style.position = '';
        submitButton.disabled = false;
        cancelButton.disabled = false;
        inputField.disabled = false;
        
        modal.style.display = 'none';
        form.reset();
    }

    showHelpModal() {
        document.getElementById('help-modal').style.display = 'block';
    }

    hideHelpModal() {
        document.getElementById('help-modal').style.display = 'none';
    }

    async addFeed() {
        const url = document.getElementById('feed-url').value;
        const submitButton = document.querySelector('#add-feed-form button[type="submit"]');
        const cancelButton = document.getElementById('cancel-add-feed');
        const inputField = document.getElementById('feed-url');
        const originalText = submitButton.textContent;
        
        // Show loading state - disable all form controls
        submitButton.disabled = true;
        submitButton.style.position = 'relative';
        cancelButton.disabled = true;
        inputField.disabled = true;
        
        // Create spinner overlay that doesn't change button size
        const spinner = document.createElement('div');
        spinner.className = 'button-spinner-overlay';
        
        // Add inline styles to ensure visibility
        spinner.style.cssText = `
            position: absolute !important;
            top: 0 !important;
            left: 0 !important;
            right: 0 !important;
            bottom: 0 !important;
            width: 100% !important;
            height: 100% !important;
            background-color: rgba(26, 115, 232, 0.9) !important;
            display: flex !important;
            align-items: center !important;
            justify-content: center !important;
            z-index: 9999 !important;
            border-radius: 4px !important;
        `;
        
        const spinnerInner = document.createElement('div');
        spinnerInner.style.cssText = `
            width: 20px !important;
            height: 20px !important;
            border: 4px solid rgba(255, 255, 255, 0.3) !important;
            border-top: 4px solid #ffffff !important;
            border-radius: 50% !important;
            box-sizing: border-box !important;
        `;
        
        spinner.appendChild(spinnerInner);
        submitButton.appendChild(spinner);
        
        // Add JavaScript-based rotation animation
        let rotation = 0;
        let isAnimating = true;
        const animateSpinner = () => {
            if (!isAnimating) return;
            rotation += 6; // 6 degrees per frame
            spinnerInner.style.transform = `rotate(${rotation}deg)`;
            requestAnimationFrame(animateSpinner);
        };
        
        // Store reference to stop animation later
        spinner.stopAnimation = () => {
            isAnimating = false;
        };
        
        animateSpinner();
        
        try {
            const response = await fetch('/api/feeds', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url })
            });
            
            if (response.ok) {
                this.hideAddFeedModal();
                await this.loadFeeds();
                await this.updateUnreadCounts();
            } else {
                let errorMessage = `HTTP ${response.status}`;
                try {
                    const error = await response.json();
                    errorMessage = error.error || errorMessage;
                } catch (e) {
                    // Use default error message
                }
                this.showError('Failed to add feed: ' + errorMessage);
            }
        } catch (error) {
            this.showError('Failed to add feed: ' + error.message);
        } finally {
            // Always restore all form controls
            const spinnerOverlay = submitButton.querySelector('.button-spinner-overlay');
            if (spinnerOverlay) {
                if (spinnerOverlay.stopAnimation) {
                    spinnerOverlay.stopAnimation();
                }
                spinnerOverlay.remove();
            }
            submitButton.style.position = '';
            submitButton.disabled = false;
            cancelButton.disabled = false;
            inputField.disabled = false;
        }
    }

    async deleteFeed(feedId) {
        if (!confirm('Are you sure you want to remove this feed from your subscriptions?')) return;
        
        try {
            const response = await fetch(`/api/feeds/${feedId}`, {
                method: 'DELETE'
            });
            
            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || `HTTP ${response.status}`);
            }
            
            // Always clear current selection after delete
            this.currentFeed = null;
            this.currentArticle = null;
            this.articles = [];
            
            await this.loadFeeds();
            
            // Always go to "Articles" after delete
            this.selectFeed('all');
        } catch (error) {
            console.error('Delete feed error:', error);
            this.showError('Failed to delete feed: ' + error.message);
        }
    }

    async refreshFeeds() {
        try {
            await fetch('/api/feeds/refresh', { method: 'POST' });
            
            if (this.currentFeed) {
                await this.loadArticles(this.currentFeed);
            }
            
            await this.updateUnreadCounts();
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
                <div class="login-footer">
                    <a href="/privacy" target="_blank" class="privacy-link">Privacy Policy</a>
                </div>
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