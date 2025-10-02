class GoReadApp {
    constructor() {
        this.currentFeed = null;
        this.currentArticle = null;
        this.feeds = [];
        this.articles = [];
        this.user = null;
        this.articleFilter = 'unread'; // Default to showing unread articles
        this.subscriptionInfo = null;
        this.sessionStarted = true; // Track if this is a fresh session for filtering behavior
        this.authCheckFailed = false; // Track if we've already failed auth to avoid repeated requests

        // Performance optimizations
        this.throttleTimeout = null;

        // Font preference
        this.fontPreference = localStorage.getItem('fontPreference') || 'sans-serif';
        
        this.init();
    }

    async init() {
        await this.checkAuth();
        if (this.user) {
            // Show app immediately for better perceived performance
            this.showApp();
            this.bindEvents();
            this.setupKeyboardShortcuts();

            // Apply saved font preference
            this.applyFontPreference();
            
            // Load data in parallel but don't block UI
            Promise.all([
                this.loadSubscriptionInfo(),
                this.loadFeedsOptimized()
            ]).then(() => {
                console.log('Initial data loading complete');
                // Optional: Start background sync after initial load
                this.startUnreadCountSync();
            });
        } else {
            console.log('User not authenticated, showing login');
            this.showLogin();
        }
    }

    async checkAuth() {
        // If we've already determined the user isn't authenticated, don't make another request
        if (this.authCheckFailed) {
            return false;
        }
        
        try {
            const response = await fetch('/auth/me');
            if (response.ok) {
                const data = await response.json();
                this.user = data.user;
                this.authCheckFailed = false; // Reset flag on successful auth
                return true;
            }
            // Handle specific auth failure cases
            if (response.status === 401) {
                this.authCheckFailed = true; // Set flag to prevent future requests
                console.log('User not authenticated - showing login screen');
                return false;
            }
            // Other response errors
            console.warn('Auth check failed with status:', response.status);
            this.authCheckFailed = true;
            return false;
        } catch (error) {
            // Network or other errors (like server not running)
            console.warn('Unable to connect to authentication service:', error.message);
            this.authCheckFailed = true;
            return false;
        }
    }

    async loadSubscriptionInfo() {
        try {
            console.log('Loading subscription info...');
            const response = await fetch('/api/subscription');
            if (response.ok) {
                this.subscriptionInfo = await response.json();
                console.log('Subscription info loaded:', this.subscriptionInfo);
                // Update the subscription display in header and sidebar
                this.updateSubscriptionDisplay();
            } else if (response.status === 401) {
                // Handle authentication errors by redirecting to login
                console.log('Authentication failed while loading subscription info, showing login');
                this.authCheckFailed = true;
                this.showLogin();
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

        const fontToggleBtn = document.getElementById('font-toggle-btn');
        if (fontToggleBtn) {
            fontToggleBtn.addEventListener('click', () => {
                this.toggleFont();
            });
        }

        const importOpmlBtn = document.getElementById('import-opml-btn');
        if (importOpmlBtn) {
            importOpmlBtn.addEventListener('click', () => {
                this.showImportOpmlModal();
            });
        }

        // Setup touch swipe gestures for article navigation on phones
        this.setupSwipeGestures();

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
            importOpmlForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                const manager = await this.loadModalManager();
                manager.importOpml();
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
                // Reset session state when switching filters to apply proper filtering
                this.sessionStarted = true;
                this.applyArticleFilter();
                // Allow session navigation after filter change
                this.sessionStarted = false;
            });
        });

        // Ensure radio button state matches the initial articleFilter value
        const checkedRadio = document.querySelector(`input[name="article-filter"][value="${this.articleFilter}"]`);
        if (checkedRadio) {
            checkedRadio.checked = true;
        }

        // Mobile navigation events
        this.setupMobileNavigation();
    }

    setupMobileNavigation() {
        // Mobile menu toggle
        const mobileMenuBtn = document.getElementById('mobile-menu-btn');
        const headerActions = document.getElementById('header-actions');

        if (mobileMenuBtn && headerActions) {
            mobileMenuBtn.addEventListener('click', () => {
                headerActions.classList.toggle('show');
            });

            // Close mobile menu when clicking outside
            document.addEventListener('click', (e) => {
                if (!mobileMenuBtn.contains(e.target) && !headerActions.contains(e.target)) {
                    headerActions.classList.remove('show');
                }
            });
        }

        // Mobile pane navigation - only for phones (under 768px) in portrait mode
        const mobileNavButtons = document.querySelectorAll('.mobile-nav-btn');
        const feedPane = document.querySelector('.feed-pane');
        const articlePane = document.querySelector('.article-pane');
        const contentPane = document.querySelector('.content-pane');

        // Initialize mobile navigation - show content pane by default on phones in portrait only
        const isPortrait = window.matchMedia('(orientation: portrait)').matches;
        if (window.innerWidth < 768 && isPortrait) {
            // Start with content pane visible
            if (feedPane) feedPane.classList.remove('active');
            if (articlePane) articlePane.classList.remove('active');

            // Set content button as active
            const contentBtn = document.querySelector('[data-pane="content"]');
            if (contentBtn) {
                mobileNavButtons.forEach(btn => btn.classList.remove('active'));
                contentBtn.classList.add('active');
            }
        }

        if (mobileNavButtons.length > 0) {
            mobileNavButtons.forEach(btn => {
                btn.addEventListener('click', (e) => {
                    e.preventDefault();
                    e.stopPropagation();

                    // Only handle navigation on phones (under 768px) in portrait mode
                    const isPortrait = window.matchMedia('(orientation: portrait)').matches;
                    if (window.innerWidth >= 768 || !isPortrait) return;

                    const pane = btn.dataset.pane;
                    console.log('Mobile nav button clicked:', pane);

                    // Remove active class from all buttons
                    mobileNavButtons.forEach(b => b.classList.remove('active'));
                    // Add active class to clicked button
                    btn.classList.add('active');

                    // Hide all panes
                    if (feedPane) feedPane.classList.remove('active');
                    if (articlePane) articlePane.classList.remove('active');

                    // Show selected pane
                    if (pane === 'feeds' && feedPane) {
                        feedPane.classList.add('active');
                        console.log('Showing feeds pane');
                    } else if (pane === 'articles' && articlePane) {
                        articlePane.classList.add('active');
                        console.log('Showing articles pane');
                    } else if (pane === 'content') {
                        console.log('Showing content pane');
                    }
                    // Content pane is always visible as background
                });
            });
        }
    }

    updateMobileNavigation(pane) {
        // Check if we're in portrait mode on phones (landscape uses two-pane like tablets)
        const isPortrait = window.matchMedia('(orientation: portrait)').matches;

        // Only update on mobile screens under 768px in portrait mode
        if (window.innerWidth >= 768 || !isPortrait) return;

        const mobileNavButtons = document.querySelectorAll('.mobile-nav-btn');
        const feedPane = document.querySelector('.feed-pane');
        const articlePane = document.querySelector('.article-pane');

        // Remove active class from all buttons
        mobileNavButtons.forEach(btn => btn.classList.remove('active'));

        // Hide all panes
        if (feedPane) feedPane.classList.remove('active');
        if (articlePane) articlePane.classList.remove('active');

        // Show selected pane and activate corresponding button
        if (pane === 'feeds' && feedPane) {
            feedPane.classList.add('active');
            const feedsBtn = document.querySelector('[data-pane="feeds"]');
            if (feedsBtn) feedsBtn.classList.add('active');
        } else if (pane === 'articles' && articlePane) {
            articlePane.classList.add('active');
            const articlesBtn = document.querySelector('[data-pane="articles"]');
            if (articlesBtn) articlesBtn.classList.add('active');
        } else if (pane === 'content') {
            // Content pane is always visible as background, just activate the button
            const contentBtn = document.querySelector('[data-pane="content"]');
            if (contentBtn) contentBtn.classList.add('active');
        }
    }

    setupSwipeGestures() {
        const contentPane = document.querySelector('.content-pane');
        if (!contentPane) return;

        let touchStartX = 0;
        let touchStartY = 0;
        let touchEndX = 0;
        let touchEndY = 0;

        // Minimum swipe distance in pixels
        const minSwipeDistance = 50;
        // Maximum vertical movement allowed for horizontal swipe
        const maxVerticalMovement = 100;

        contentPane.addEventListener('touchstart', (e) => {
            // Enable swipes on phones (under 768px) and tablets in portrait mode
            if (window.innerWidth >= 1024) return;

            touchStartX = e.changedTouches[0].screenX;
            touchStartY = e.changedTouches[0].screenY;
        }, { passive: true });

        contentPane.addEventListener('touchend', (e) => {
            // Enable swipes on phones (under 768px) and tablets in portrait mode
            if (window.innerWidth >= 1024) return;
            if (this.currentArticle === null || this.articles.length === 0) return;

            touchEndX = e.changedTouches[0].screenX;
            touchEndY = e.changedTouches[0].screenY;

            const horizontalDistance = touchEndX - touchStartX;
            const verticalDistance = Math.abs(touchEndY - touchStartY);

            // Check if this is a horizontal swipe (not vertical scroll)
            if (Math.abs(horizontalDistance) > minSwipeDistance &&
                verticalDistance < maxVerticalMovement) {

                if (horizontalDistance < 0) {
                    // Swipe left - next article
                    this.selectNextArticleAndMarkCurrentAsRead();
                } else {
                    // Swipe right - previous article
                    this.selectPreviousArticle();
                }
            }
        }, { passive: true });
    }

    setupKeyboardShortcuts() {
        document.addEventListener('keydown', async (e) => {
            if (e.ctrlKey || e.metaKey) return;
            
            // Don't handle shortcuts when typing in input fields
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
            
            switch(e.key) {
                case 'j':
                    e.preventDefault();
                    await this.selectNextArticleAndMarkCurrentAsRead();
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
                case 'f':
                    e.preventDefault();
                    this.toggleFont();
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

    async loadFeedsOptimized() {
        try {
            // Show loading state for feeds while preserving special items
            const feedList = document.getElementById('feed-list');
            // Remove existing feed items and loading, but preserve special items
            const existingFeeds = feedList.querySelectorAll('.feed-item:not(.special)');
            const existingLoading = feedList.querySelector('.loading');
            existingFeeds.forEach(item => item.remove());
            if (existingLoading) existingLoading.remove();

            // Add loading indicator
            const loadingDiv = document.createElement('div');
            loadingDiv.className = 'loading';
            loadingDiv.textContent = 'Loading feeds...';
            feedList.appendChild(loadingDiv);

            document.getElementById('article-list').innerHTML = '<div class="loading">Loading articles...</div>';

            // Batch feeds and unread counts requests
            const [feedsResponse, countsResponse] = await Promise.all([
                fetch('/api/feeds'),
                fetch('/api/feeds/unread-counts')
            ]);

            if (!feedsResponse.ok) {
                // Handle authentication errors by redirecting to login
                if (feedsResponse.status === 401) {
                    console.log('Authentication failed while loading feeds, showing login');
                    this.authCheckFailed = true;
                    this.showLogin();
                    return;
                }
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
                    // Select "all" - now the element should exist
                    this.currentFeed = 'all';
                    const allElement = document.querySelector(`[data-feed-id="all"]`);
                    if (allElement) {
                        allElement.classList.add('active');
                    }
                    document.getElementById('article-pane-title').textContent = 'Articles';
                    
                    // Load articles immediately after unread counts are applied
                    this.loadArticles('all');
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
        console.log('RENDER FEEDS CALLED WITH:', this.feeds?.length || 0, 'feeds');
        if (!Array.isArray(this.feeds)) {
            return;
        }

        const feedList = document.getElementById('feed-list');

        // Remove existing feed items and loading indicator (not the "All" item)
        const existingFeeds = feedList.querySelectorAll('.feed-item:not(.special)');
        const loadingIndicator = feedList.querySelector('.loading');
        existingFeeds.forEach(item => item.remove());
        if (loadingIndicator) loadingIndicator.remove();

        // Sort feeds alphabetically by title
        const sortedFeeds = [...this.feeds].sort((a, b) => a.title.localeCompare(b.title));

        sortedFeeds.forEach((feed) => {
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

        // Update mobile navigation to show articles pane when feed is selected (portrait phones only)
        this.updateMobileNavigation('articles');

        // On tablets and phone landscape, scroll the articles pane into view within the feed pane
        const isLandscape = window.matchMedia('(orientation: landscape)').matches;
        if ((window.innerWidth >= 768 && window.innerWidth < 1024) ||
            (window.innerWidth < 768 && isLandscape)) {
            const articlePane = document.querySelector('.article-pane');
            if (articlePane) {
                articlePane.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
        }
    }

    async loadArticles(feedId, append = false) {
        try {
            if (!append) {
                document.getElementById('article-list').innerHTML = '<div class="loading">Loading articles...</div>';
                this.articles = [];
                this.articleOffset = 0;
            }
            
            const limit = 50;
            let url;
            if (feedId === 'all') {
                url = `/api/feeds/all/articles?limit=${limit}&offset=${this.articleOffset || 0}`;
            } else {
                url = `/api/feeds/${feedId}/articles`;
            }
            
            const response = await fetch(url);
            const newArticles = await response.json();
            
            if (append) {
                this.articles.push(...newArticles);
            } else {
                this.articles = newArticles;
            }
            
            this.articleOffset = (this.articleOffset || 0) + newArticles.length;
            this.hasMoreArticles = newArticles.length === limit;
            
            this.renderArticlesOptimized();
            
            // Add load more button if needed
            if (feedId === 'all' && this.hasMoreArticles && !append) {
                this.addLoadMoreButton();
            }
        } catch (error) {
            this.showError('Failed to load articles: ' + error.message);
        }
    }

    addLoadMoreButton() {
        const articleList = document.getElementById('article-list');
        
        // Remove existing load more button
        const existingButton = articleList.querySelector('.load-more-button');
        if (existingButton) {
            existingButton.remove();
        }
        
        if (this.hasMoreArticles) {
            const loadMoreDiv = document.createElement('div');
            loadMoreDiv.className = 'load-more-button';
            loadMoreDiv.style.cssText = 'padding: 20px; text-align: center; border-top: 1px solid #e1e5e9;';
            
            const button = document.createElement('button');
            button.className = 'btn btn-secondary';
            button.textContent = 'Load More Articles';
            button.onclick = () => this.loadMoreArticles();
            
            loadMoreDiv.appendChild(button);
            articleList.appendChild(loadMoreDiv);
        }
    }

    async loadMoreArticles() {
        if (this.currentFeed === 'all' && this.hasMoreArticles) {
            const button = document.querySelector('.load-more-button button');
            if (button) {
                button.textContent = 'Loading...';
                button.disabled = true;
            }
            
            await this.loadArticles('all', true);
            
            if (button) {
                button.textContent = 'Load More Articles';
                button.disabled = false;
            }
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
                ${article.description ? `<div class="article-description">${this.sanitizeContent(article.description, true)}</div>` : ''}
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

    renderArticlesOptimized() {
        const articleList = document.getElementById('article-list');
        
        if (this.articles.length === 0) {
            articleList.innerHTML = '<div class="placeholder">No articles found</div>';
            return;
        }
        
        // Render all loaded articles (pagination handles limiting on server side)
        const articlesToRender = this.articles;
        
        // Use DocumentFragment for better performance
        const fragment = document.createDocumentFragment();
        
        // Remove existing event listeners by cloning the element
        const newArticleList = articleList.cloneNode(false);
        articleList.parentNode.replaceChild(newArticleList, articleList);
        // Update reference
        const updatedArticleList = document.getElementById('article-list');
        
        // Add event delegation for star buttons and article selection
        // Use both click and touchend for better iPad support
        const handleArticleInteraction = (e) => {
            console.log('Article interaction:', e.type, e.target);
            
            // Handle star button clicks via event delegation
            if (e.target.classList.contains('star-btn')) {
                e.stopPropagation();
                e.preventDefault();
                console.log('Star button clicked');
                this.toggleStar(parseInt(e.target.dataset.articleId));
                return;
            }
            
            // Handle article selection
            const articleItem = e.target.closest('.article-item');
            if (articleItem && !e.target.classList.contains('star-btn')) {
                const index = parseInt(articleItem.dataset.index);
                console.log('Article selected, index:', index);
                this.selectArticle(index);
            }
        };
        
        updatedArticleList.addEventListener('click', handleArticleInteraction);
        // Add touchend for better iPad support
        updatedArticleList.addEventListener('touchend', (e) => {
            // Prevent the click event from firing after touchend
            e.preventDefault();
            handleArticleInteraction(e);
        });
        
        articlesToRender.forEach((article, i) => {
            const articleItem = document.createElement('div');
            articleItem.className = `article-item ${article.is_read ? 'read' : ''}`;
            articleItem.dataset.articleId = article.id;
            articleItem.dataset.index = i;  // Use actual index from slice
            
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
                ${article.description ? `<div class="article-description">${this.sanitizeContent(article.description, true)}</div>` : ''}
            `;
            
            fragment.appendChild(articleItem);
        });
        
        // Pagination load more button is handled separately by addLoadMoreButton method
        // No need to add it here as it's managed by the pagination system
        
        // Single DOM append operation
        updatedArticleList.appendChild(fragment);
        
        // Add load more button if needed (for paginated "all articles" view)
        if (this.currentFeed === 'all' && this.hasMoreArticles) {
            this.addLoadMoreButton();
        }
        
        // Apply current filter after rendering
        this.applyArticleFilter();
        
        // Mark session as no longer fresh after first article load
        this.sessionStarted = false;
        
        // Auto-select the first visible article if any exist
        const visibleArticles = document.querySelectorAll('.article-item:not(.filtered-out)');
        if (visibleArticles.length > 0) {
            const firstVisibleIndex = parseInt(visibleArticles[0].dataset.index);
            this.selectArticle(firstVisibleIndex);
        } else {
            this.currentArticle = null;
            document.getElementById('article-content').innerHTML = '<div class="placeholder"><p>No articles to display.</p></div>';
        }
    }

    loadMoreArticles() {
        // This can be enhanced later to load more articles incrementally
        this.renderArticles(); // Fall back to full render for now
    }

    async selectArticle(index) {
        // Auto-read behavior is disabled - users manually control read status with 'm' key

        this.currentArticle = index;

        document.querySelectorAll('.article-item').forEach(item => {
            item.classList.remove('active');
        });

        const articleItem = document.querySelector(`[data-index="${index}"]`);
        articleItem.classList.add('active');

        const article = this.articles[index];
        this.displayArticle(article);

        // Update mobile navigation to show content pane when article is selected (phones only)
        this.updateMobileNavigation('content');

        articleItem.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }


    sanitizeContent(content, stripImages = false) {
        // Use DOMPurify to sanitize HTML content, removing iframes and other potentially harmful elements
        if (typeof DOMPurify !== 'undefined') {
            const forbiddenTags = ['iframe', 'object', 'embed', 'applet', 'script'];
            
            // Add img to forbidden tags if we want to strip images (for previews)
            if (stripImages) {
                forbiddenTags.push('img');
            }
            
            const result = DOMPurify.sanitize(content, {
                FORBID_TAGS: forbiddenTags,
                FORBID_ATTR: ['onload', 'onclick', 'onerror', 'onmouseover']
            });
            
            return result;
        }
        // Fallback: remove iframes and optionally images if DOMPurify is not available
        let result = content.replace(/<iframe[^>]*>.*?<\/iframe>/gi, '<p><em>[Embedded content removed]</em></p>');
        if (stripImages) {
            result = result.replace(/<img[^>]*>/gi, '');
        }
        return result;
    }

    displayArticle(article) {
        const contentPane = document.getElementById('article-content');
        const publishedDate = new Date(article.published_at).toLocaleString();

        // Sanitize the article content to prevent iframe and other security issues
        const sanitizedContent = this.sanitizeContent(article.content || article.description || '<p>No content available.</p>');

        contentPane.innerHTML = `
            <h1>${this.escapeHtml(article.title)}</h1>
            <div class="meta">
                <span>${publishedDate}</span>
                ${article.author ? `<span>by ${this.escapeHtml(article.author)}</span>` : ''}
                <a href="${article.url}" target="_blank" rel="noopener">View Original</a>
            </div>
            <div class="content">
                ${sanitizedContent}
            </div>
        `;

        // Scroll the content pane to the top to ensure the new article is displayed from its beginning
        contentPane.scrollTop = 0;

    }

    async selectNextArticleAndMarkCurrentAsRead() {
        if (this.currentArticle === null || this.articles.length === 0) return;
        
        const currentArticle = this.articles[this.currentArticle];
        
        // Find the next unread article (since that's what we care about when filtering by unread)
        let nextArticleIndex = null;
        for (let i = this.currentArticle + 1; i < this.articles.length; i++) {
            const nextArticle = this.articles[i];
            if (this.articleFilter === 'unread') {
                // When filtering by unread, find next unread article
                if (!nextArticle.is_read) {
                    nextArticleIndex = i;
                    break;
                }
            } else {
                // When showing all articles, find next visible article
                const articleItem = document.querySelector(`[data-index="${i}"]`);
                if (articleItem && !articleItem.classList.contains('filtered-out')) {
                    nextArticleIndex = i;
                    break;
                }
            }
        }
        
        // Mark current article as read if it's unread
        if (!currentArticle.is_read) {
            try {
                // Update the data model
                currentArticle.is_read = true;
                
                // Update UI
                const currentArticleItem = document.querySelector(`[data-index="${this.currentArticle}"]`);
                currentArticleItem.classList.add('read');
                
                // Update unread counts
                const countChange = -1; // Marking as read
                if (currentArticle.feed_id) {
                    this.updateUnreadCountsOptimistically(currentArticle.feed_id, countChange);
                } else {
                    this.updateUnreadCountsForCurrentFeed(countChange);
                }
                
                // Article will be greyed out by the .read CSS class applied above
                // No need to hide it during session navigation
                
                // Make the API call
                await this.markAsRead(currentArticle.id, true);
            } catch (error) {
                console.error('Failed to mark article as read:', error);
                // Revert on error
                currentArticle.is_read = false;
                const currentArticleItem = document.querySelector(`[data-index="${this.currentArticle}"]`);
                currentArticleItem.classList.remove('read');
                // No need to show/hide since we no longer hide during session navigation
                // Revert unread count
                const revertCountChange = 1;
                if (currentArticle.feed_id) {
                    this.updateUnreadCountsOptimistically(currentArticle.feed_id, revertCountChange);
                } else {
                    this.updateUnreadCountsForCurrentFeed(revertCountChange);
                }
            }
        }
        
        // Navigate to next article if we found one
        if (nextArticleIndex !== null) {
            this.selectArticle(nextArticleIndex);
        } else {
            // No next article found - check what articles are still available
            if (this.articleFilter === 'unread') {
                // Check if any unread articles remain
                const hasUnreadArticles = this.articles.some((article, index) => 
                    index > this.currentArticle && !article.is_read);
                if (!hasUnreadArticles) {
                    this.currentArticle = null;
                    document.getElementById('article-content').innerHTML = '<div class="placeholder"><p>No more unread articles.</p></div>';
                }
            } else {
                // Check visible articles
                const visibleArticles = document.querySelectorAll('.article-item:not(.filtered-out)');
                if (visibleArticles.length === 0) {
                    this.currentArticle = null;
                    document.getElementById('article-content').innerHTML = '<div class="placeholder"><p>No articles to display.</p></div>';
                }
            }
        }
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
        
        // If no next article found, stay on current article
        // No action needed - current selection remains
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
        
        // If no previous article found, stay on current article
        // No action needed - current selection remains
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
            
            // No longer hide articles when marked as read during the session
            // They will be greyed out by the .read CSS class applied above
            // Articles are only completely filtered out when the page loads/refreshes
            
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
            
            // No need to show/hide articles since we no longer hide them during the session
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
                // Only hide read articles on fresh session/reload, not during navigation
                if (this.sessionStarted && this.articleFilter === 'unread' && article.is_read) {
                    item.classList.add('filtered-out');
                    item.style.display = 'none';
                } else if (this.articleFilter !== 'unread') {
                    item.classList.add('filtered-out');
                    item.style.display = 'none';
                } else {
                    // During session navigation, keep read articles visible but greyed out
                    item.classList.remove('filtered-out');
                    item.style.display = '';
                }
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
        // Handle empty or null unreadCounts
        if (!unreadCounts || typeof unreadCounts !== 'object') {
            unreadCounts = {};
        }
        
        // Update individual feed counts
        let totalUnread = 0;
        this.feeds.forEach(feed => {
            // Try multiple ways to match feed IDs (number, string, parseInt)
            let unreadCount = 0;
            const feedId = feed.id;
            
            if (unreadCounts.hasOwnProperty(feedId)) {
                unreadCount = unreadCounts[feedId];
            } else if (unreadCounts.hasOwnProperty(feedId.toString())) {
                unreadCount = unreadCounts[feedId.toString()];
            } else if (unreadCounts.hasOwnProperty(parseInt(feedId))) {
                unreadCount = unreadCounts[parseInt(feedId)];
            } else {
                // Try to find by any matching value
                for (const [key, value] of Object.entries(unreadCounts)) {
                    if (parseInt(key) === parseInt(feedId) || key.toString() === feedId.toString()) {
                        unreadCount = value;
                        break;
                    }
                }
            }
            
            // Ensure unreadCount is a number
            unreadCount = parseInt(unreadCount) || 0;
            totalUnread += unreadCount;
            
            const countElement = document.querySelector(`[data-feed-id="${feed.id}"] .unread-count`);
            if (countElement) {
                countElement.textContent = unreadCount;
                countElement.dataset.count = unreadCount;
            } else {
                // Try alternative selectors
                const altElement = document.querySelector(`[data-feed-id='${feed.id}'] .unread-count`);
                if (altElement) {
                    altElement.textContent = unreadCount;
                    altElement.dataset.count = unreadCount;
                }
            }
        });
        
        // Update "Articles" count
        const allUnreadElement = document.getElementById('all-unread-count');
        if (allUnreadElement) {
            allUnreadElement.textContent = totalUnread;
            allUnreadElement.dataset.count = totalUnread;
        }
        
        // Force a DOM refresh
        if (totalUnread > 0) {
            // Trigger a visual update to ensure changes are visible
            setTimeout(() => {
                const allElement = document.getElementById('all-unread-count');
                if (allElement && allElement.textContent === '0' && totalUnread > 0) {
                    allElement.textContent = totalUnread;
                    allElement.dataset.count = totalUnread;
                }
            }, 100);
        }
    }

    updateUnreadCountsOptimistically(feedId, countChange) {
        // Update DOM counts immediately (removed requestAnimationFrame to fix error)
        {
            // Ensure feedId is a string for selector matching  
            const feedIdStr = String(feedId);
            
            // Update the specific feed's unread count immediately
            const feedCountElement = document.querySelector(`[data-feed-id="${feedIdStr}"] .unread-count`);
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
            } else {
                console.warn('All unread count element not found');
            }
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

    // Lazy-load modal manager on first use
    async loadModalManager() {
        if (this.modalManager) return this.modalManager;

        console.log('Loading modal module...');
        const module = await import('./modals.js');
        this.modalManager = new module.ModalManager(this);
        this.modalManager.init();
        console.log('Modal module loaded');

        return this.modalManager;
    }

    async showAddFeedModal() {
        const manager = await this.loadModalManager();
        manager.showAddFeedModal();
    }

    hideAddFeedModal() {
        if (this.modalManager) {
            this.modalManager.hideAddFeedModal();
        }
    }

    async showHelpModal() {
        const manager = await this.loadModalManager();
        manager.showHelpModal();
    }

    hideHelpModal() {
        if (this.modalManager) {
            this.modalManager.hideHelpModal();
        }
    }

    async showImportOpmlModal() {
        const manager = await this.loadModalManager();
        manager.showImportOpmlModal();
    }

    hideImportOpmlModal() {
        if (this.modalManager) {
            this.modalManager.hideImportOpmlModal();
        }
    }

    toggleFont() {
        // Toggle between sans-serif and serif
        this.fontPreference = this.fontPreference === 'sans-serif' ? 'serif' : 'sans-serif';

        // Apply the font preference
        this.applyFontPreference();

        // Save to localStorage
        localStorage.setItem('fontPreference', this.fontPreference);

        console.log(`Font switched to: ${this.fontPreference}`);
    }

    applyFontPreference() {
        const body = document.body;

        // Remove existing font classes
        body.classList.remove('font-serif', 'font-sans-serif');

        // Apply new font class
        if (this.fontPreference === 'serif') {
            body.classList.add('font-serif');
        }

        // Update button appearance to show current state
        const fontToggleBtn = document.getElementById('font-toggle-btn');
        if (fontToggleBtn) {
            fontToggleBtn.textContent = this.fontPreference === 'serif' ? 'Serif' : 'Sans';
            fontToggleBtn.title = `Current: ${this.fontPreference === 'serif' ? 'Serif' : 'Sans-serif'} - Click to switch`;
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
            setTimeout(animateSpinner, 16);
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
                console.log('Feed added successfully, reloading data...');
                this.hideAddFeedModal();
                await this.loadFeeds();
                console.log('Reloading subscription info after adding feed...');
                await this.loadSubscriptionInfo();
                await this.updateUnreadCounts();
                console.log('Calling updateSubscriptionDisplay after adding feed...');
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
                console.error('Delete feed API error:', errorData);
                throw new Error(errorData.error || `HTTP ${response.status}`);
            }
            
            const result = await response.json();
            
            // Always clear current selection after delete
            this.currentFeed = null;
            this.currentArticle = null;
            this.articles = [];
            
            // Force a slight delay to ensure backend has processed the unsubscribe
            await new Promise(resolve => setTimeout(resolve, 200));
            
            await this.loadFeedsOptimized();
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
        const refreshBtn = document.getElementById('refresh-btn');
        const originalText = refreshBtn ? refreshBtn.textContent : '';
        
        try {
            // Show loading state
            if (refreshBtn) {
                refreshBtn.disabled = true;
                refreshBtn.textContent = 'Refreshing...';
            }
            
            console.log('Refreshing feeds...');
            const response = await fetch('/api/feeds/refresh', { method: 'POST' });
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
            
            console.log('Feeds refreshed successfully');
            
            // Reload articles for current feed to show any new content
            if (this.currentFeed) {
                await this.loadArticles(this.currentFeed);
            }
            
            // Update unread counts to reflect any changes
            await this.updateUnreadCounts();
            
            // Show success feedback
            this.showSuccess('Feeds refreshed successfully');
            
        } catch (error) {
            console.error('Failed to refresh feeds:', error);
            this.showError('Failed to refresh feeds: ' + error.message);
        } finally {
            // Restore button state
            if (refreshBtn) {
                refreshBtn.disabled = false;
                refreshBtn.textContent = originalText;
            }
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
                <div class="login-logo">
                    <img src="/static/goread2_logo.svg" alt="GoRead2 Logo" width="80" height="80">
                </div>
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
            // Reset auth check flag when attempting login
            this.authCheckFailed = false;
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
        } else if (info.status === 'admin' || info.status === 'admin_trial') {
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
        if (!panel) {
            console.log('ERROR: subscription-panel element not found in DOM');
            return;
        }

        console.log('Updating subscription panel with info:', this.subscriptionInfo);

        if (!this.subscriptionInfo) {
            console.log('No subscription info, hiding panel');
            panel.classList.add('hidden');
            return;
        }

        panel.classList.remove('hidden');
        const info = this.subscriptionInfo;
        console.log('Subscription status:', info.status);
        let panelHTML = '';

        if (info.status === 'unlimited') {
            panelHTML = `
                <div class="subscription-info unlimited">
                    <div>
                        <div class="status">Unlimited Access</div>
                        <div class="details">No subscription required</div>
                    </div>
                </div>
            `;
        } else if (info.status === 'admin' || info.status === 'admin_trial') {
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
            console.log('Falling through to expired case - status was:', info.status);
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
    try {
        console.log('DOM loaded, initializing GoReadApp');
        window.app = new GoReadApp();
        console.log('GoReadApp initialized:', window.app);
    } catch (error) {
        console.error('Error initializing GoReadApp:', error);
    }
});

// Global error handler to catch any unhandled errors
window.addEventListener('error', (event) => {
    console.error('Global error caught:', event.error);
});

window.addEventListener('unhandledrejection', (event) => {
    console.error('Unhandled promise rejection:', event.reason);
});