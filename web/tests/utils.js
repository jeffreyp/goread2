// Test utilities for GoRead2 frontend tests

/**
 * Wait for async operations to complete
 */
const waitFor = (condition, timeout = 1000) => {
    return new Promise((resolve, reject) => {
        const interval = 10;
        let elapsed = 0;
        
        const check = () => {
            if (condition()) {
                resolve();
            } else if (elapsed >= timeout) {
                reject(new Error(`Timeout waiting for condition after ${timeout}ms`));
            } else {
                elapsed += interval;
                setTimeout(check, interval);
            }
        };
        
        check();
    });
};

/**
 * Wait for DOM element to appear
 */
const waitForElement = (selector, timeout = 1000) => {
    return waitFor(() => document.querySelector(selector), timeout);
};

/**
 * Simulate user event
 */
const fireEvent = {
    click: (element) => {
        const event = new Event('click', { bubbles: true, cancelable: true });
        element.dispatchEvent(event);
    },
    
    keydown: (element, key, options = {}) => {
        const event = new KeyboardEvent('keydown', {
            key,
            bubbles: true,
            cancelable: true,
            ...options
        });
        element.dispatchEvent(event);
    },
    
    submit: (form) => {
        const event = new Event('submit', { bubbles: true, cancelable: true });
        form.dispatchEvent(event);
    },
    
    change: (element, value) => {
        element.value = value;
        const event = new Event('change', { bubbles: true, cancelable: true });
        element.dispatchEvent(event);
    },
    
    input: (element, value) => {
        element.value = value;
        const event = new Event('input', { bubbles: true, cancelable: true });
        element.dispatchEvent(event);
    }
};

/**
 * Create mock file for file input testing
 */
const createMockFile = (name = 'test.opml', content = '<opml></opml>', type = 'text/xml') => {
    const file = new File([content], name, { type });
    return file;
};

/**
 * Mock FormData for file upload testing
 */
const mockFormData = () => {
    const formData = {
        entries: new Map(),
        append: jest.fn((key, value) => {
            formData.entries.set(key, value);
        }),
        get: jest.fn((key) => formData.entries.get(key)),
        has: jest.fn((key) => formData.entries.has(key))
    };
    
    global.FormData = jest.fn(() => formData);
    return formData;
};

/**
 * Create a mock fetch response
 */
const createMockResponse = (data, options = {}) => {
    const { status = 200, ok = status >= 200 && status < 300 } = options;
    
    return {
        ok,
        status,
        json: jest.fn(() => Promise.resolve(data)),
        text: jest.fn(() => Promise.resolve(JSON.stringify(data)))
    };
};

/**
 * Create test articles data
 */
const createTestArticles = (count = 3) => {
    return Array.from({ length: count }, (_, i) => ({
        id: i + 1,
        feed_id: 1,
        title: `Test Article ${i + 1}`,
        url: `https://example.com/article${i + 1}`,
        content: `<p>Content for test article ${i + 1}</p>`,
        description: `Description for test article ${i + 1}`,
        author: `Author ${i + 1}`,
        published_at: new Date(Date.now() - i * 86400000).toISOString(), // i days ago
        created_at: new Date(Date.now() - i * 86400000).toISOString(),
        is_read: i === 0, // First article is read
        is_starred: i === 1 // Second article is starred
    }));
};

/**
 * Create test feeds data
 */
const createTestFeeds = (count = 2) => {
    return Array.from({ length: count }, (_, i) => ({
        id: i + 1,
        title: `Test Feed ${i + 1}`,
        url: `https://example.com/feed${i + 1}`,
        description: `Description for test feed ${i + 1}`,
        created_at: new Date(Date.now() - i * 86400000).toISOString(),
        updated_at: new Date().toISOString(),
        last_fetch: new Date().toISOString()
    }));
};

/**
 * Helper to trigger DOMContentLoaded
 */
const triggerDOMContentLoaded = () => {
    const event = new Event('DOMContentLoaded');
    document.dispatchEvent(event);
};

/**
 * Helper to load a JavaScript class/module in test environment
 */
const loadScript = async (scriptPath) => {
    const fs = require('fs');
    const path = require('path');
    
    const fullPath = path.join(__dirname, '../..', scriptPath);
    const scriptContent = fs.readFileSync(fullPath, 'utf8');
    
    // Execute the script in the current context
    eval(scriptContent);
};

/**
 * Assert element has class
 */
const expectElementToHaveClass = (element, className) => {
    expect(element.classList.contains(className)).toBe(true);
};

/**
 * Assert element does not have class
 */
const expectElementNotToHaveClass = (element, className) => {
    expect(element.classList.contains(className)).toBe(false);
};

/**
 * Assert element is visible
 * Note: In jsdom, offsetParent may be null even for visible elements,
 * so we only check style.display
 */
const expectElementToBeVisible = (element) => {
    expect(element.style.display).not.toBe('none');
};

/**
 * Assert element is hidden
 */
const expectElementToBeHidden = (element) => {
    expect(element.style.display === 'none' || element.offsetParent === null).toBe(true);
};

// Export all utilities
module.exports = {
    waitFor,
    waitForElement,
    fireEvent,
    createMockFile,
    mockFormData,
    createMockResponse,
    createTestArticles,
    createTestFeeds,
    triggerDOMContentLoaded,
    loadScript,
    expectElementToHaveClass,
    expectElementNotToHaveClass,
    expectElementToBeVisible,
    expectElementToBeHidden
};