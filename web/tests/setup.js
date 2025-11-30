// Test setup file for GoRead2 frontend tests

// Global test utilities
global.mockFetch = (responses = {}) => {
    const fetch = jest.fn();
    
    // Default responses
    const defaultResponses = {
        '/auth/me': { 
            ok: true, 
            json: () => Promise.resolve({ 
                user: { 
                    id: 1, 
                    name: 'Test User', 
                    email: 'test@example.com', 
                    avatar: 'https://example.com/avatar.jpg',
                    created_at: '2024-01-01T00:00:00Z'
                } 
            }) 
        },
        '/api/subscription': { 
            ok: true, 
            json: () => Promise.resolve({ 
                status: 'trial', 
                current_feeds: 5, 
                feed_limit: 20, 
                trial_days_remaining: 15,
                trial_ends_at: '2024-12-31T23:59:59Z'
            }) 
        },
        '/api/feeds': { 
            ok: true, 
            json: () => Promise.resolve([
                { id: 1, title: 'Test Feed 1', url: 'https://example.com/feed1' },
                { id: 2, title: 'Test Feed 2', url: 'https://example.com/feed2' }
            ]) 
        },
        '/api/feeds/unread-counts': { 
            ok: true, 
            json: () => Promise.resolve({ '1': 5, '2': 3 }) 
        }
    };

    // Merge with custom responses
    const allResponses = { ...defaultResponses, ...responses };

    fetch.mockImplementation((url, options = {}) => {
        const response = allResponses[url] || allResponses[url.split('?')[0]];
        
        if (response) {
            return Promise.resolve(response);
        }
        
        // Default to 404 for unmatched URLs
        return Promise.resolve({
            ok: false,
            status: 404,
            json: () => Promise.resolve({ error: 'Not found' })
        });
    });

    global.fetch = fetch;
    return fetch;
};

// Mock DOM APIs that might not be available in jsdom
global.scrollIntoView = jest.fn();
global.requestAnimationFrame = jest.fn(cb => setTimeout(cb, 16));
global.cancelAnimationFrame = jest.fn();

// Mock window.location methods
delete window.location;
window.location = {
    href: 'http://localhost:3000',
    origin: 'http://localhost:3000',
    pathname: '/',
    search: '',
    hash: '',
    assign: jest.fn(),
    replace: jest.fn(),
    reload: jest.fn()
};

// Setup DOM elements that the app expects
beforeEach(() => {
    document.body.innerHTML = '';
    
    // Create basic app structure
    const appHTML = `
        <div id="app" style="display: none;">
            <header class="header">
                <div class="header-actions"></div>
            </header>
            <div class="sidebar">
                <div id="feed-list">
                    <div class="feed-item special" data-feed-id="all">
                        <span class="feed-title">Articles</span>
                        <span id="all-unread-count" class="unread-count" data-count="0">0</span>
                    </div>
                </div>
                <div id="subscription-panel" class="hidden"></div>
            </div>
            <div class="main-content">
                <div class="article-pane">
                    <h2 id="article-pane-title">Articles</h2>
                    <div id="article-list"></div>
                </div>
                <div class="content-pane">
                    <h2 id="content-pane-title">Select an article</h2>
                    <div id="article-content"></div>
                </div>
            </div>
        </div>

        <!-- Modals -->
        <div id="add-feed-modal" class="modal" style="display: none;">
            <div class="modal-content">
                <span class="close">&times;</span>
                <h2>Add Feed</h2>
                <form id="add-feed-form">
                    <input type="url" id="feed-url" placeholder="Feed URL" required>
                    <div class="form-actions">
                        <button type="submit">Add Feed</button>
                        <button type="button" id="cancel-add-feed">Cancel</button>
                    </div>
                </form>
            </div>
        </div>

        <div id="help-modal" class="modal" style="display: none;">
            <div class="modal-content">
                <span class="close">&times;</span>
                <h2>Help</h2>
                <p>Keyboard shortcuts and help content</p>
            </div>
        </div>

        <div id="import-opml-modal" class="modal" style="display: none;">
            <div class="modal-content">
                <span class="close">&times;</span>
                <h2>Import OPML</h2>
                <form id="import-opml-form">
                    <input type="file" id="opml-file" accept=".opml,.xml" required>
                    <div class="form-actions">
                        <button type="submit">Import</button>
                        <button type="button" id="cancel-import-opml">Cancel</button>
                    </div>
                </form>
            </div>
        </div>

        <!-- Buttons -->
        <button id="add-feed-btn">Add Feed</button>
        <button id="refresh-btn">Refresh</button>
        <button id="help-btn">Help</button>
        <button id="font-toggle-btn" title="Toggle font style">Aa</button>
        <button id="import-opml-btn">Import OPML</button>

        <!-- Account Page Elements -->
        <div id="profile-info"></div>
        <div id="subscription-details"></div>
        <div id="usage-stats"></div>

        <!-- Confirmation Modal for Account Page -->
        <div id="confirm-modal" class="modal" style="display: none;">
            <div class="modal-content">
                <span class="close">&times;</span>
                <h2 id="confirm-title">Confirm Action</h2>
                <p id="confirm-message">Are you sure?</p>
                <div class="form-actions">
                    <button id="confirm-action" class="btn btn-primary">Confirm</button>
                    <button id="cancel-action" class="btn btn-secondary">Cancel</button>
                </div>
            </div>
        </div>
    `;

    document.body.innerHTML = appHTML;
    
    // Mock fetch by default
    mockFetch();
});

afterEach(() => {
    jest.clearAllMocks();
    document.body.innerHTML = '';
});