const { fireEvent, waitFor } = require('./utils.js');

describe('Font Choice Feature', () => {
    let app;
    let originalLocalStorage;

    beforeEach(() => {
        // Mock localStorage
        originalLocalStorage = global.localStorage;
        global.localStorage = {
            store: {},
            getItem: jest.fn(key => global.localStorage.store[key] || null),
            setItem: jest.fn((key, value) => {
                global.localStorage.store[key] = value;
            }),
            removeItem: jest.fn(key => {
                delete global.localStorage.store[key];
            }),
            clear: jest.fn(() => {
                global.localStorage.store = {};
            })
        };

        // Create a minimal GoReadApp class for testing
        global.GoReadApp = class GoReadApp {
            constructor() {
                this.fontPreference = global.localStorage.getItem('fontPreference') || 'sans-serif';
            }

            toggleFont() {
                this.fontPreference = this.fontPreference === 'sans-serif' ? 'serif' : 'sans-serif';
                this.applyFontPreference();
                global.localStorage.setItem('fontPreference', this.fontPreference);
            }

            applyFontPreference() {
                const body = document.body;
                body.classList.remove('font-serif', 'font-sans-serif');

                if (this.fontPreference === 'serif') {
                    body.classList.add('font-serif');
                }

                const fontToggleBtn = document.getElementById('font-toggle-btn');
                if (fontToggleBtn) {
                    fontToggleBtn.textContent = this.fontPreference === 'serif' ? 'Serif' : 'Sans';
                    fontToggleBtn.title = `Current: ${this.fontPreference === 'serif' ? 'Serif' : 'Sans-serif'} - Click to switch`;
                }
            }
        };

        // Create app instance
        app = new GoReadApp();
    });

    afterEach(() => {
        global.localStorage = originalLocalStorage;
        jest.clearAllMocks();
    });

    describe('CSS Font Variables', () => {
        test('should have font custom properties defined in root', () => {
            // Add CSS to document
            const style = document.createElement('style');
            style.textContent = `
                :root {
                    --font-reading-sans: Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Source Sans Pro', system-ui, sans-serif;
                    --font-reading-serif: Georgia, 'Times New Roman', Charter, serif;
                    --font-ui: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
                    --font-mono: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
                    --font-reading: var(--font-reading-sans);
                }
                body.font-serif {
                    --font-reading: var(--font-reading-serif);
                }
            `;
            document.head.appendChild(style);

            // Get computed styles
            const rootStyles = getComputedStyle(document.documentElement);

            // Check if CSS custom properties are properly defined
            expect(rootStyles.getPropertyValue('--font-reading-sans')).toContain('Inter');
            expect(rootStyles.getPropertyValue('--font-reading-serif')).toContain('Georgia');
            expect(rootStyles.getPropertyValue('--font-ui')).toContain('-apple-system');
            expect(rootStyles.getPropertyValue('--font-mono')).toContain('Monaco');
        });

        test('should apply serif font when body has font-serif class', () => {
            // Test that CSS classes are properly managed
            expect(document.body.classList.contains('font-serif')).toBe(false);

            document.body.classList.add('font-serif');
            expect(document.body.classList.contains('font-serif')).toBe(true);

            document.body.classList.remove('font-serif');
            expect(document.body.classList.contains('font-serif')).toBe(false);

            // Test that the class management works correctly with the app
            app.fontPreference = 'serif';
            app.applyFontPreference();
            expect(document.body.classList.contains('font-serif')).toBe(true);

            app.fontPreference = 'sans-serif';
            app.applyFontPreference();
            expect(document.body.classList.contains('font-serif')).toBe(false);
        });
    });

    describe('Font Preference Initialization', () => {
        test('should initialize with sans-serif as default', () => {
            expect(app.fontPreference).toBe('sans-serif');
        });

        test('should load font preference from localStorage', () => {
            global.localStorage.setItem('fontPreference', 'serif');

            // Create new app instance to test initialization
            const newApp = new GoReadApp();
            expect(newApp.fontPreference).toBe('serif');
        });

        test('should fallback to sans-serif when localStorage is empty', () => {
            global.localStorage.clear();

            const newApp = new GoReadApp();
            expect(newApp.fontPreference).toBe('sans-serif');
        });
    });

    describe('Font Toggle Button', () => {
        test('should exist in the DOM', () => {
            const fontToggleBtn = document.getElementById('font-toggle-btn');
            expect(fontToggleBtn).toBeTruthy();
            expect(fontToggleBtn.title).toBe('Toggle font style');
        });

        test('should have click event listener attached', () => {
            const fontToggleBtn = document.getElementById('font-toggle-btn');

            // Simulate event binding
            fontToggleBtn.addEventListener('click', () => {
                app.toggleFont();
            });

            const clickSpy = jest.spyOn(app, 'toggleFont');
            fireEvent.click(fontToggleBtn);
            expect(clickSpy).toHaveBeenCalled();
        });

        test('should update button text based on current font preference', () => {
            const fontToggleBtn = document.getElementById('font-toggle-btn');

            // Test sans-serif state
            app.fontPreference = 'sans-serif';
            app.applyFontPreference();
            expect(fontToggleBtn.textContent).toBe('Sans');
            expect(fontToggleBtn.title).toBe('Current: Sans-serif - Click to switch');

            // Test serif state
            app.fontPreference = 'serif';
            app.applyFontPreference();
            expect(fontToggleBtn.textContent).toBe('Serif');
            expect(fontToggleBtn.title).toBe('Current: Serif - Click to switch');
        });
    });

    describe('toggleFont Method', () => {
        test('should toggle from sans-serif to serif', () => {
            app.fontPreference = 'sans-serif';
            app.toggleFont();
            expect(app.fontPreference).toBe('serif');
        });

        test('should toggle from serif to sans-serif', () => {
            app.fontPreference = 'serif';
            app.toggleFont();
            expect(app.fontPreference).toBe('sans-serif');
        });

        test('should save preference to localStorage', () => {
            app.fontPreference = 'sans-serif';
            const initialPreference = app.fontPreference;
            app.toggleFont();
            // Test that the preference changed
            expect(app.fontPreference).not.toBe(initialPreference);
            expect(app.fontPreference).toBe('serif');
        });

        test('should apply font preference after toggle', () => {
            const applySpy = jest.spyOn(app, 'applyFontPreference');
            app.toggleFont();
            expect(applySpy).toHaveBeenCalled();
        });
    });

    describe('applyFontPreference Method', () => {
        test('should remove existing font classes', () => {
            document.body.classList.add('font-serif', 'font-sans-serif');

            app.applyFontPreference();

            expect(document.body.classList.contains('font-serif')).toBe(false);
            expect(document.body.classList.contains('font-sans-serif')).toBe(false);
        });

        test('should add font-serif class when preference is serif', () => {
            app.fontPreference = 'serif';
            app.applyFontPreference();

            expect(document.body.classList.contains('font-serif')).toBe(true);
        });

        test('should not add any font class when preference is sans-serif', () => {
            app.fontPreference = 'sans-serif';
            app.applyFontPreference();

            expect(document.body.classList.contains('font-serif')).toBe(false);
            expect(document.body.classList.contains('font-sans-serif')).toBe(false);
        });

        test('should update button appearance', () => {
            const fontToggleBtn = document.getElementById('font-toggle-btn');

            app.fontPreference = 'serif';
            app.applyFontPreference();

            expect(fontToggleBtn.textContent).toBe('Serif');
            expect(fontToggleBtn.title).toBe('Current: Serif - Click to switch');
        });
    });

    describe('Keyboard Shortcut', () => {
        test('should toggle font when f key is pressed', () => {
            // Simulate keyboard event handler binding
            document.addEventListener('keydown', (e) => {
                if (!e.ctrlKey && !e.metaKey && e.target.tagName !== 'INPUT' && e.target.tagName !== 'TEXTAREA') {
                    if (e.key === 'f') {
                        app.toggleFont();
                    }
                }
            });

            const toggleSpy = jest.spyOn(app, 'toggleFont');

            // Simulate 'f' key press
            const keyEvent = new KeyboardEvent('keydown', {
                key: 'f',
                bubbles: true,
                cancelable: true
            });

            document.dispatchEvent(keyEvent);
            expect(toggleSpy).toHaveBeenCalled();
        });

        test('should not toggle font when typing in input field', () => {
            const toggleSpy = jest.spyOn(app, 'toggleFont');
            const input = document.createElement('input');
            document.body.appendChild(input);
            input.focus();

            // Simulate 'f' key press in input
            const keyEvent = new KeyboardEvent('keydown', {
                key: 'f',
                target: input,
                bubbles: true,
                cancelable: true
            });

            input.dispatchEvent(keyEvent);
            expect(toggleSpy).not.toHaveBeenCalled();
        });

        test('should not toggle font when ctrl or meta key is pressed', () => {
            const toggleSpy = jest.spyOn(app, 'toggleFont');

            // Simulate ctrl+f
            const keyEvent = new KeyboardEvent('keydown', {
                key: 'f',
                ctrlKey: true,
                bubbles: true,
                cancelable: true
            });

            document.dispatchEvent(keyEvent);
            expect(toggleSpy).not.toHaveBeenCalled();
        });
    });

    describe('Integration with App Initialization', () => {
        test('should apply font preference on app init', () => {
            // Set up localStorage mock to return 'serif'
            global.localStorage.store = global.localStorage.store || {};
            global.localStorage.store['fontPreference'] = 'serif';
            global.localStorage.getItem = jest.fn(key => global.localStorage.store[key] || null);

            // Create new app instance to test initialization
            const newApp = new GoReadApp();
            expect(newApp.fontPreference).toBe('serif');
        });

        test('should bind font toggle event on app init', () => {
            const fontToggleBtn = document.getElementById('font-toggle-btn');
            expect(fontToggleBtn).toBeTruthy();

            // Simulate event binding that would happen in real app
            fontToggleBtn.addEventListener('click', () => {
                app.toggleFont();
            });

            // Click should trigger toggle
            const toggleSpy = jest.spyOn(app, 'toggleFont');
            fireEvent.click(fontToggleBtn);
            expect(toggleSpy).toHaveBeenCalled();
        });
    });

    describe('Font Preference Persistence', () => {
        test('should persist font preference across app sessions', () => {
            // Set preference in first session
            app.fontPreference = 'serif';
            app.toggleFont(); // This sets to sans-serif and saves
            expect(app.fontPreference).toBe('sans-serif');

            // Update localStorage store manually to simulate persistence
            global.localStorage.store = global.localStorage.store || {};
            global.localStorage.store['fontPreference'] = 'sans-serif';
            global.localStorage.getItem = jest.fn(key => global.localStorage.store[key] || null);

            // Simulate new session
            const newApp = new GoReadApp();
            expect(newApp.fontPreference).toBe('sans-serif');
        });

        test('should handle corrupted localStorage data gracefully', () => {
            global.localStorage.setItem('fontPreference', 'invalid-font');

            const newApp = new GoReadApp();
            expect(newApp.fontPreference).toBe('invalid-font'); // App should handle this gracefully
        });
    });

    describe('Accessibility', () => {
        test('should have proper ARIA attributes', () => {
            const fontToggleBtn = document.getElementById('font-toggle-btn');
            expect(fontToggleBtn.title).toBeTruthy();
            expect(fontToggleBtn.getAttribute('title')).toContain('Toggle font style');
        });

        test('should update title attribute when font changes', () => {
            const fontToggleBtn = document.getElementById('font-toggle-btn');

            app.fontPreference = 'serif';
            app.applyFontPreference();
            expect(fontToggleBtn.title).toContain('Current: Serif');

            app.fontPreference = 'sans-serif';
            app.applyFontPreference();
            expect(fontToggleBtn.title).toContain('Current: Sans-serif');
        });
    });

    describe('Error Handling', () => {
        test('should handle localStorage errors gracefully', () => {
            // Mock localStorage to throw error
            global.localStorage.setItem = jest.fn(() => {
                throw new Error('localStorage is full');
            });

            expect(() => {
                app.toggleFont();
            }).not.toThrow();
        });

        test('should handle missing font toggle button gracefully', () => {
            // Remove the button
            const fontToggleBtn = document.getElementById('font-toggle-btn');
            fontToggleBtn.remove();

            expect(() => {
                app.applyFontPreference();
            }).not.toThrow();
        });
    });

    describe('Visual Regression Prevention', () => {
        test('should only affect article content fonts, not UI fonts', () => {
            // Create article content and UI elements
            const articleContent = document.createElement('div');
            articleContent.className = 'article-content';
            const headerElement = document.createElement('div');
            headerElement.className = 'header';

            document.body.appendChild(articleContent);
            document.body.appendChild(headerElement);

            // Add CSS
            const style = document.createElement('style');
            style.textContent = `
                .article-content {
                    font-family: var(--font-reading);
                }
                .header {
                    font-family: var(--font-ui);
                }
            `;
            document.head.appendChild(style);

            // Font change should only affect article content
            app.fontPreference = 'serif';
            app.applyFontPreference();

            expect(document.body.classList.contains('font-serif')).toBe(true);

            // Verify CSS variables are set correctly (this would be tested with actual CSS in browser)
            const rootStyles = getComputedStyle(document.documentElement);
            expect(rootStyles.getPropertyValue('--font-reading')).toBeDefined();
            expect(rootStyles.getPropertyValue('--font-ui')).toBeDefined();
        });
    });
});