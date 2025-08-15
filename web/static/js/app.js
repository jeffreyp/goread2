class GoReadApp {
    constructor() {
        this.currentFeed = null;
        this.currentArticle = null;
        this.feeds = [];
        this.articles = [];
        this.user = null;
        this.articleFilter = 'unread'; // Default to showing unread articles
        this.subscriptionInfo = null;
        
        this.init();
    }

    async init() {
        await this.checkAuth();
        if (this.user) {
            this.bindEvents();
            await this.loadSubscriptionInfo();
            this.loadFeeds();
            this.setupKeyboardShortcuts();
            this.startUnreadCountSync();
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

    async loadSubscriptionInfo() {
        try {
            const response = await fetch('/api/subscription');
            if (response.ok) {
                this.subscriptionInfo = await response.json();
            } else {
                console.error('Failed to load subscription info:', response.status);
            }
        } catch (error) {
            console.error('Error loading subscription info:', error);
        }
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

        const importOpmlBtn = document.getElementById('import-opml-btn');
        if (importOpmlBtn) {
            importOpmlBtn.addEventListener('click', () => {
                this.showImportOpmlModal();
            });
        }

        // Handle close buttons for all modals
        document.querySelectorAll('.close').forEach(closeBtn => {
            closeBtn.addEventListener('click', (e) => {
                const modal = e.target.closest('.modal');
                if (modal.id === 'add-feed-modal') {
                    this.hideAddFeedModal();
                } else if (modal.id === 'help-modal') {
                    this.hideHelpModal();
                } else if (modal.id === 'import-opml-modal') {
                    this.hideImportOpmlModal();
                }
            });
        });

        const cancelBtn = document.getElementById('cancel-add-feed');
        if (cancelBtn) {
            cancelBtn.addEventListener('click', () => {
                this.hideAddFeedModal();
            });
        }

        const cancelImportBtn = document.getElementById('cancel-import-opml');
        if (cancelImportBtn) {
            cancelImportBtn.addEventListener('click', () => {
                this.hideImportOpmlModal();
            });
        }

        const addFeedForm = document.getElementById('add-feed-form');
        if (addFeedForm) {
            addFeedForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.addFeed();
            });
        }

        const importOpmlForm = document.getElementById('import-opml-form');
        if (importOpmlForm) {
            importOpmlForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.importOpml();
            });
        }

        window.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                if (e.target.id === 'add-feed-modal') {
                    this.hideAddFeedModal();
                } else if (e.target.id === 'help-modal') {
                    this.hideHelpModal();
                } else if (e.target.id === 'import-opml-modal') {
                    this.hideImportOpmlModal();
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

        // Set up article filter listeners
        document.querySelectorAll('input[name="article-filter"]').forEach(radio => {
            radio.addEventListener('change', (e) => {
                this.articleFilter = e.target.value;
                this.applyArticleFilter();
            });
        });

        // Ensure radio button state matches the initial articleFilter value
        const checkedRadio = document.querySelector(`input[name="article-filter"][value="${this.articleFilter}"]`);
        if (checkedRadio) {
            checkedRadio.checked = true;
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
            // Batch feeds and unread counts requests
            const [feedsResponse, countsResponse] = await Promise.all([
                fetch('/api/feeds'),
                fetch('/api/feeds/unread-counts')
            ]);
            
            if (!feedsResponse.ok) {
                throw new Error(`HTTP ${feedsResponse.status}`);
            }
            
            const feedsData = await feedsResponse.json();
            this.feeds = feedsData;
            
            if (Array.isArray(this.feeds)) {
                this.renderFeeds();
                
                // Process unread counts if available
                if (countsResponse.ok) {
                    const unreadCounts = await countsResponse.json();
                    this.applyUnreadCounts(unreadCounts);
                }
                
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
        
        const feedTitle = 'Articles';
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
        
        // Use DocumentFragment for better performance
        const fragment = document.createDocumentFragment();
        
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
        
        // Batch DOM operations using fragment
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
            
            fragment.appendChild(articleItem);
        });
        
        // Single DOM append operation
        updatedArticleList.appendChild(fragment);
        
        // Apply current filter after rendering
        this.applyArticleFilter();
        
        // Auto-select the first visible article if any exist
        const visibleArticles = document.querySelectorAll('.article-item:not(.filtered-out)');
        if (visibleArticles.length > 0) {
            const firstVisibleIndex = parseInt(visibleArticles[0].dataset.index);
            this.selectArticle(firstVisibleIndex);
        } else {
            // No visible articles, clear current selection and show placeholder
            this.currentArticle = null;
            document.getElementById('article-content').innerHTML = '<div class="placeholder"><p>No articles to display.</p></div>';
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
                try {
                    // Optimistically update UI first for instant feedback
                    article.is_read = true;
                    const articleItem = document.querySelector(`[data-index="${this.currentArticle}"]`);
                    if (articleItem) {
                        articleItem.classList.add('read');
                        
                        // If showing unread only and article is now read, hide it
                        if (this.articleFilter === 'unread') {
                            articleItem.classList.add('filtered-out');
                            articleItem.style.display = 'none';
                        }
                    }
                    
                    // Update unread counts immediately (optimistically)
                    if (article.feed_id) {
                        this.updateUnreadCountsOptimistically(article.feed_id, -1);
                    } else {
                        // When viewing "all articles", we need to determine which feed this article belongs to
                        this.updateUnreadCountsForCurrentFeed(-1);
                    }
                    
                    // Then make the API call
                    await this.markAsRead(article.id, true);
                } catch (error) {
                    // If marking as read failed, revert the optimistic changes
                    console.error('Failed to mark article as read, reverting UI state');
                    article.is_read = false;
                    if (articleItem) {
                        articleItem.classList.remove('read');
                        if (this.articleFilter === 'unread') {
                            articleItem.classList.remove('filtered-out');
                            articleItem.style.display = '';
                        }
                    }
                    // Revert the unread count change
                    if (article.feed_id) {
                        this.updateUnreadCountsOptimistically(article.feed_id, 1);
                    } else {
                        this.updateUnreadCountsForCurrentFeed(1);
                    }
                }
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
        
    }

    selectNextArticle() {
        if (this.currentArticle === null || this.articles.length === 0) return;
        
        // Find next visible article
        for (let i = this.currentArticle + 1; i < this.articles.length; i++) {
            const articleItem = document.querySelector(`[data-index="${i}"]`);
            if (articleItem && !articleItem.classList.contains('filtered-out')) {
                this.selectArticle(i);
                return;
            }
        }
    }

    selectPreviousArticle() {
        if (this.currentArticle === null || this.articles.length === 0) return;
        
        // Find previous visible article
        for (let i = this.currentArticle - 1; i >= 0; i--) {
            const articleItem = document.querySelector(`[data-index="${i}"]`);
            if (articleItem && !articleItem.classList.contains('filtered-out')) {
                this.selectArticle(i);
                return;
            }
        }
    }

    selectNextVisibleArticle() {
        // Try to select next visible article, otherwise select previous visible article
        const visibleArticles = document.querySelectorAll('.article-item:not(.filtered-out)');
        if (visibleArticles.length === 0) {
            this.currentArticle = null;
            document.getElementById('article-content').innerHTML = '<div class="placeholder"><p>No articles to display.</p></div>';
            return;
        }
        
        // Find the first visible article after current index
        for (let i = this.currentArticle + 1; i < this.articles.length; i++) {
            const articleItem = document.querySelector(`[data-index="${i}"]`);
            if (articleItem && !articleItem.classList.contains('filtered-out')) {
                this.selectArticle(i);
                return;
            }
        }
        
        // If no next visible article, select the first visible one
        const firstVisibleIndex = parseInt(visibleArticles[0].dataset.index);
        this.selectArticle(firstVisibleIndex);
    }

    openCurrentArticle() {
        if (this.currentArticle === null) return;
        
        const article = this.articles[this.currentArticle];
        window.open(article.url, '_blank');
    }

    async toggleCurrentArticleRead() {
        if (this.currentArticle === null) return;
        
        const article = this.articles[this.currentArticle];
        const oldReadState = article.is_read;
        const newReadState = !article.is_read;
        
        try {
            // Optimistically update UI first for instant feedback
            article.is_read = newReadState;
            const articleItem = document.querySelector(`[data-index="${this.currentArticle}"]`);
            articleItem.classList.toggle('read', article.is_read);
            
            // Update unread counts immediately (optimistically)
            const countChange = newReadState ? -1 : 1; // -1 when marking as read, +1 when marking as unread
            if (article.feed_id) {
                this.updateUnreadCountsOptimistically(article.feed_id, countChange);
            } else {
                this.updateUnreadCountsForCurrentFeed(countChange);
            }
            
            // If showing unread only and article is now read, hide it
            if (this.articleFilter === 'unread' && article.is_read) {
                articleItem.classList.add('filtered-out');
                articleItem.style.display = 'none';
                
                // Select next visible article
                this.selectNextVisibleArticle();
            }
            
            // Then make the API call
            await this.markAsRead(article.id, newReadState);
        } catch (error) {
            // If API call failed, revert the optimistic changes
            console.error('Failed to toggle article read status, reverting UI state');
            article.is_read = oldReadState;
            const articleItem = document.querySelector(`[data-index="${this.currentArticle}"]`);
            articleItem.classList.toggle('read', article.is_read);
            
            // Revert the unread count change
            const revertCountChange = oldReadState ? -1 : 1;
            if (article.feed_id) {
                this.updateUnreadCountsOptimistically(article.feed_id, revertCountChange);
            } else {
                this.updateUnreadCountsForCurrentFeed(revertCountChange);
            }
            
            // If we had hidden the article, show it again
            if (this.articleFilter === 'unread' && !article.is_read) {
                articleItem.classList.remove('filtered-out');
                articleItem.style.display = '';
            }
        }
    }

    applyArticleFilter() {
        const articleItems = document.querySelectorAll('.article-item');
        const articleList = document.getElementById('article-list');
        
        // Remove any existing placeholders
        const existingPlaceholder = articleList.querySelector('.article-list-placeholder');
        if (existingPlaceholder) {
            existingPlaceholder.remove();
        }
        
        articleItems.forEach((item) => {
            const articleIndex = parseInt(item.dataset.index);
            const article = this.articles[articleIndex];
            
            if (!article) {
                // If article data is missing, hide the item
                item.classList.add('filtered-out');
                item.style.display = 'none';
                return;
            }
            
            const shouldShow = this.articleFilter === 'all' || 
                              (this.articleFilter === 'unread' && !article.is_read);
            
            if (shouldShow) {
                item.classList.remove('filtered-out');
                item.style.display = '';
            } else {
                item.classList.add('filtered-out');
                item.style.display = 'none';
            }
        });

        // Check if any articles are visible after filtering
        const visibleArticles = document.querySelectorAll('.article-item:not(.filtered-out)');
        
        if (visibleArticles.length === 0) {
            // No visible articles, clear current selection and show placeholder
            this.currentArticle = null;
            document.getElementById('article-content').innerHTML = '<div class="placeholder"><p>No articles to display.</p></div>';
            
            // Show placeholder in article list
            const placeholder = document.createElement('div');
            placeholder.className = 'placeholder article-list-placeholder';
            placeholder.innerHTML = '<p>No unread articles in this feed.</p>';
            articleList.appendChild(placeholder);
        } else {
            // Remove placeholder if articles are visible
            const placeholder = articleList.querySelector('.article-list-placeholder');
            if (placeholder) {
                placeholder.remove();
            }
            
            if (this.currentArticle !== null) {
                // Check if current article is still visible
                const currentArticleItem = document.querySelector(`[data-index="${this.currentArticle}"]`);
                if (currentArticleItem && currentArticleItem.classList.contains('filtered-out')) {
                    // Current article is now hidden, select first visible article
                    const firstVisibleIndex = parseInt(visibleArticles[0].dataset.index);
                    this.selectArticle(firstVisibleIndex);
                }
            }
        }
    }

    toggleCurrentArticleStar() {
        if (this.currentArticle === null) return;
        
        const article = this.articles[this.currentArticle];
        this.toggleStar(article.id);
    }

    async markAsRead(articleId, isRead) {
        try {
            const response = await fetch(`/api/articles/${articleId}/read`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ is_read: isRead })
            });
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
        } catch (error) {
            console.error('Failed to mark article as read:', error);
            throw error; // Re-throw so callers can handle the failure
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
            this.applyUnreadCounts(unreadCounts);
        } catch (error) {
            console.error('Error updating unread counts:', error);
        }
    }

    applyUnreadCounts(unreadCounts) {
        
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
        
    }

    updateUnreadCountsOptimistically(feedId, countChange) {
        // Update the specific feed's unread count immediately
        const feedCountElement = document.querySelector(`[data-feed-id="${feedId}"] .unread-count`);
        if (feedCountElement) {
            const currentCount = parseInt(feedCountElement.dataset.count) || 0;
            const newCount = Math.max(0, currentCount + countChange); // Don't go below 0
            feedCountElement.textContent = newCount;
            feedCountElement.dataset.count = newCount;
        }
        
        // Update the "All Articles" total count
        const allUnreadElement = document.getElementById('all-unread-count');
        if (allUnreadElement) {
            const currentTotal = parseInt(allUnreadElement.dataset.count) || 0;
            const newTotal = Math.max(0, currentTotal + countChange); // Don't go below 0
            allUnreadElement.textContent = newTotal;
            allUnreadElement.dataset.count = newTotal;
        }
    }

    updateUnreadCountsForCurrentFeed(countChange) {
        // When viewing "all articles", we only update the total count
        // since we don't know which specific feed the article belongs to
        if (this.currentFeed === 'all') {
            const allUnreadElement = document.getElementById('all-unread-count');
            if (allUnreadElement) {
                const currentTotal = parseInt(allUnreadElement.dataset.count) || 0;
                const newTotal = Math.max(0, currentTotal + countChange); // Don't go below 0
                allUnreadElement.textContent = newTotal;
                allUnreadElement.dataset.count = newTotal;
            }
        } else if (this.currentFeed) {
            // For specific feeds, update both the feed and total counts
            this.updateUnreadCountsOptimistically(this.currentFeed, countChange);
        }
    }

    // Periodically sync unread counts with server to correct any drift
    startUnreadCountSync() {
        // Sync every 5 minutes to catch any discrepancies
        setInterval(() => {
            this.updateUnreadCounts();
        }, 5 * 60 * 1000); // 5 minutes
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

    showImportOpmlModal() {
        document.getElementById('import-opml-modal').style.display = 'block';
    }

    hideImportOpmlModal() {
        const modal = document.getElementById('import-opml-modal');
        const form = document.getElementById('import-opml-form');
        modal.style.display = 'none';
        form.reset();
    }

    async importOpml() {
        const fileInput = document.getElementById('opml-file');
        const submitButton = document.querySelector('#import-opml-form button[type="submit"]');
        const cancelButton = document.getElementById('cancel-import-opml');
        const originalText = submitButton.textContent;

        if (!fileInput.files || fileInput.files.length === 0) {
            this.showError('Please select an OPML file');
            return;
        }

        const file = fileInput.files[0];

        // Basic file validation
        if (file.size > 10 * 1024 * 1024) { // 10MB limit
            this.showError('File is too large (max 10MB)');
            return;
        }

        // Show loading state
        submitButton.disabled = true;
        submitButton.textContent = 'Importing...';
        cancelButton.disabled = true;
        fileInput.disabled = true;

        try {
            const formData = new FormData();
            formData.append('opml', file);

            const response = await fetch('/api/feeds/import', {
                method: 'POST',
                body: formData
            });

            if (response.ok) {
                const result = await response.json();
                this.hideImportOpmlModal();
                await this.loadFeeds();
                await this.loadSubscriptionInfo();
                await this.updateUnreadCounts();
                this.updateSubscriptionDisplay();
                
                // Show success message
                const message = `Successfully imported ${result.imported_count} feed(s) from OPML file`;
                this.showSuccess(message);
            } else if (response.status === 402) { // Payment Required
                const error = await response.json();
                if (error.limit_reached) {
                    // Show partial success if some feeds were imported
                    if (error.imported_count > 0) {
                        await this.loadFeeds();
                        await this.updateUnreadCounts();
                        this.showSuccess(`Imported ${error.imported_count} feed(s) before reaching your limit.`);
                    }
                    this.showSubscriptionLimitModal(error);
                } else if (error.trial_expired) {
                    this.showTrialExpiredModal(error);
                } else {
                    this.showError(error.error || 'Subscription required');
                }
            } else {
                let errorMessage = `HTTP ${response.status}`;
                try {
                    const error = await response.json();
                    errorMessage = error.error || errorMessage;
                } catch (e) {
                    // Use default error message
                }
                this.showError('Failed to import OPML: ' + errorMessage);
            }
        } catch (error) {
            this.showError('Failed to import OPML: ' + error.message);
        } finally {
            // Always restore form controls
            submitButton.disabled = false;
            submitButton.textContent = originalText;
            cancelButton.disabled = false;
            fileInput.disabled = false;
        }
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
                await this.loadSubscriptionInfo();
                await this.updateUnreadCounts();
                this.updateSubscriptionDisplay();
            } else if (response.status === 402) { // Payment Required
                const error = await response.json();
                if (error.limit_reached) {
                    this.showSubscriptionLimitModal(error);
                } else if (error.trial_expired) {
                    this.showTrialExpiredModal(error);
                } else {
                    this.showError(error.error || 'Subscription required');
                }
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
            await this.loadSubscriptionInfo();
            this.updateSubscriptionDisplay();
            
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

    showSuccess(message) {
        const existingSuccess = document.querySelector('.success');
        if (existingSuccess) {
            existingSuccess.remove();
        }
        
        const successDiv = document.createElement('div');
        successDiv.className = 'success';
        successDiv.textContent = message;
        
        document.body.appendChild(successDiv);
        
        setTimeout(() => {
            successDiv.remove();
        }, 5000);
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Subscription management methods
    showSubscriptionLimitModal(error) {
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.style.display = 'block';
        modal.innerHTML = `
            <div class="modal-content">
                <span class="close">&times;</span>
                <h2>Upgrade to Pro</h2>
                <p>You've reached the free limit of ${error.current_limit} feeds.</p>
                <p>Upgrade to <strong>GoRead2 Pro</strong> for:</p>
                <ul>
                    <li>Unlimited RSS feeds</li>
                    <li>Priority support</li>
                    <li>Advanced features</li>
                </ul>
                <p style="font-size: 18px; font-weight: 600; margin: 20px 0;">Only $2.99/month</p>
                <div class="form-actions">
                    <button id="upgrade-btn" class="btn btn-primary">Upgrade to Pro</button>
                    <button id="cancel-upgrade" class="btn btn-secondary">Cancel</button>
                </div>
            </div>
        `;

        document.body.appendChild(modal);

        const closeModal = () => {
            modal.remove();
        };

        modal.querySelector('.close').addEventListener('click', closeModal);
        modal.querySelector('#cancel-upgrade').addEventListener('click', closeModal);
        modal.querySelector('#upgrade-btn').addEventListener('click', () => {
            this.startUpgradeProcess();
            closeModal();
        });

        // Close on outside click
        modal.addEventListener('click', (e) => {
            if (e.target === modal) closeModal();
        });
    }

    showTrialExpiredModal(error) {
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.style.display = 'block';
        modal.innerHTML = `
            <div class="modal-content">
                <span class="close">&times;</span>
                <h2>Free Trial Expired</h2>
                <p>Your 30-day free trial has ended.</p>
                <p>Subscribe to <strong>GoRead2 Pro</strong> to continue using the service with:</p>
                <ul>
                    <li>Unlimited RSS feeds</li>
                    <li>Continued access to all articles</li>
                    <li>Priority support</li>
                </ul>
                <p style="font-size: 18px; font-weight: 600; margin: 20px 0;">Only $2.99/month</p>
                <div class="form-actions">
                    <button id="subscribe-btn" class="btn btn-primary">Subscribe Now</button>
                    <button id="cancel-subscribe" class="btn btn-secondary">Cancel</button>
                </div>
            </div>
        `;

        document.body.appendChild(modal);

        const closeModal = () => {
            modal.remove();
        };

        modal.querySelector('.close').addEventListener('click', closeModal);
        modal.querySelector('#cancel-subscribe').addEventListener('click', closeModal);
        modal.querySelector('#subscribe-btn').addEventListener('click', () => {
            this.startUpgradeProcess();
            closeModal();
        });

        // Close on outside click
        modal.addEventListener('click', (e) => {
            if (e.target === modal) closeModal();
        });
    }

    async startUpgradeProcess() {
        try {
            // Get Stripe configuration
            const configResponse = await fetch('/api/stripe/config');
            if (!configResponse.ok) {
                throw new Error('Stripe not configured');
            }
            const config = await configResponse.json();

            // Create checkout session
            const checkoutResponse = await fetch('/api/subscription/checkout', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });

            if (!checkoutResponse.ok) {
                const error = await checkoutResponse.json();
                throw new Error(error.error || 'Failed to create checkout session');
            }

            const session = await checkoutResponse.json();

            // Redirect to Stripe Checkout
            window.location.href = session.session_url;
            
        } catch (error) {
            console.error('Upgrade process failed:', error);
            if (error.message === 'Stripe not configured') {
                this.showError('Payment processing is not available. Please contact support.');
            } else {
                this.showError('Failed to start upgrade process: ' + error.message);
            }
        }
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
            // Update header with user info and subscription status
            const headerActions = document.querySelector('.header-actions');
            
            // Create subscription status element
            const subscriptionStatus = this.createSubscriptionStatusElement();
            
            const userInfo = document.createElement('div');
            userInfo.className = 'user-info';
            userInfo.innerHTML = `
                ${subscriptionStatus}
                <span class="user-name">${this.escapeHtml(this.user.name)}</span>
                <img class="user-avatar" src="${this.user.avatar}" alt="User Avatar" width="32" height="32">
                <a href="/account" class="btn btn-secondary">Account</a>
                <button id="logout-btn" class="btn btn-secondary">Logout</button>
            `;
            headerActions.appendChild(userInfo);

            document.getElementById('logout-btn').addEventListener('click', () => {
                this.logout();
            });
        }
    }

    createSubscriptionStatusElement() {
        if (!this.subscriptionInfo) {
            return '<div class="subscription-status loading">Loading...</div>';
        }

        const info = this.subscriptionInfo;
        let statusHTML = '';
        let statusClass = '';

        if (info.status === 'unlimited') {
            // When subscription system is disabled, show unlimited status
            statusClass = 'unlimited';
            statusHTML = `
                <div class="subscription-status ${statusClass}">
                    <span class="status-badge">UNLIMITED</span>
                    <span class="status-text">Unlimited feeds</span>
                </div>
            `;
        } else if (info.status === 'admin') {
            statusClass = 'admin';
            statusHTML = `
                <div class="subscription-status ${statusClass}">
                    <span class="status-badge">ADMIN</span>
                    <span class="status-text">Unlimited access</span>
                </div>
            `;
        } else if (info.status === 'active') {
            statusClass = 'pro';
            statusHTML = `
                <div class="subscription-status ${statusClass}">
                    <span class="status-badge">PRO</span>
                    <span class="status-text">Unlimited feeds</span>
                </div>
            `;
        } else if (info.status === 'free_months') {
            statusClass = 'free';
            statusHTML = `
                <div class="subscription-status ${statusClass}">
                    <span class="status-badge">FREE</span>
                    <span class="status-text">Free months remaining</span>
                </div>
            `;
        } else if (info.status === 'trial') {
            statusClass = 'trial';
            const daysLeft = info.trial_days_remaining || 0;
            statusHTML = `
                <div class="subscription-status ${statusClass}">
                    <span class="status-badge">TRIAL</span>
                    <span class="status-text">${info.current_feeds}/${info.feed_limit} feeds • ${daysLeft} days left</span>
                </div>
            `;
        } else {
            statusClass = 'expired';
            statusHTML = `
                <div class="subscription-status ${statusClass}">
                    <span class="status-badge">EXPIRED</span>
                    <span class="status-text">Subscribe to continue</span>
                </div>
            `;
        }

        return statusHTML;
    }

    updateSubscriptionDisplay() {
        // Update header subscription status
        const subscriptionElement = document.querySelector('.subscription-status');
        if (subscriptionElement) {
            const newStatusHTML = this.createSubscriptionStatusElement();
            const tempDiv = document.createElement('div');
            tempDiv.innerHTML = newStatusHTML;
            const newElement = tempDiv.firstElementChild;
            subscriptionElement.replaceWith(newElement);
        }
        
        // Update sidebar subscription panel
        this.updateSubscriptionPanel();
    }

    updateSubscriptionPanel() {
        const panel = document.getElementById('subscription-panel');
        if (!panel) return;

        if (!this.subscriptionInfo) {
            panel.classList.add('hidden');
            return;
        }

        panel.classList.remove('hidden');
        const info = this.subscriptionInfo;
        let panelHTML = '';

        if (info.status === 'admin') {
            panelHTML = `
                <div class="subscription-info admin">
                    <div>
                        <div class="status">Admin Access</div>
                        <div class="details">Unlimited feeds</div>
                    </div>
                </div>
            `;
        } else if (info.status === 'active') {
            panelHTML = `
                <div class="subscription-info pro">
                    <div>
                        <div class="status">GoRead2 Pro</div>
                        <div class="details">Unlimited feeds</div>
                    </div>
                </div>
            `;
        } else if (info.status === 'free_months') {
            panelHTML = `
                <div class="subscription-info free">
                    <div>
                        <div class="status">Free Months</div>
                        <div class="details">Unlimited feeds</div>
                    </div>
                </div>
            `;
        } else if (info.status === 'trial') {
            const daysLeft = info.trial_days_remaining || 0;
            const feedsUsed = info.current_feeds || 0;
            const feedLimit = info.feed_limit || 20;
            const isNearLimit = feedsUsed >= feedLimit - 3; // Show warning when 3 or fewer feeds left
            
            panelHTML = `
                <div class="subscription-info trial">
                    <div>
                        <div class="status">Free Trial</div>
                        <div class="details">${feedsUsed}/${feedLimit} feeds • ${daysLeft} days left</div>
                    </div>
                    ${(isNearLimit || daysLeft <= 7) ? '<button class="upgrade-btn" onclick="app.startUpgradeProcess()">Upgrade</button>' : ''}
                </div>
            `;
        } else {
            panelHTML = `
                <div class="subscription-info expired">
                    <div>
                        <div class="status">Trial Expired</div>
                        <div class="details">Subscribe to continue</div>
                    </div>
                    <button class="upgrade-btn" onclick="app.startUpgradeProcess()">Subscribe</button>
                </div>
            `;
        }

        panel.innerHTML = panelHTML;
    }
}

document.addEventListener('DOMContentLoaded', () => {
    window.app = new GoReadApp();
});