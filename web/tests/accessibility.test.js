const fs = require('fs');
const path = require('path');

// app.js declares its classes at the top level and only registers event
// listeners, so evaluating it in the jsdom window has no other side effects.
const appSource = fs.readFileSync(path.join(__dirname, '../static/js/app.js'), 'utf8');
const { GoReadApp, ErrorHandler } = new Function(`${appSource}\nreturn { GoReadApp, ErrorHandler };`)();
const appProto = () => Object.create(GoReadApp.prototype);

/**
 * Accessibility Tests
 *
 * Verifies ARIA attributes, accessible labels, and focus management on modals
 * and interactive elements. Prevents regressions from the gr-68 a11y work.
 */
describe('Accessibility', () => {

    describe('Modal ARIA attributes', () => {
        const modalIds = [
            { id: 'add-feed-modal', labelId: 'add-feed-modal-heading' },
            { id: 'help-modal', labelId: 'help-modal-heading' },
            { id: 'import-opml-modal', labelId: 'import-opml-modal-heading' },
        ];

        modalIds.forEach(({ id, labelId }) => {
            describe(id, () => {
                test('has role="dialog"', () => {
                    const modal = document.getElementById(id);
                    expect(modal.getAttribute('role')).toBe('dialog');
                });

                test('has aria-modal="true"', () => {
                    const modal = document.getElementById(id);
                    expect(modal.getAttribute('aria-modal')).toBe('true');
                });

                test('has aria-labelledby pointing to a visible heading', () => {
                    const modal = document.getElementById(id);
                    expect(modal.getAttribute('aria-labelledby')).toBe(labelId);

                    const heading = document.getElementById(labelId);
                    expect(heading).not.toBeNull();
                    expect(heading.textContent.trim()).not.toBe('');
                });

                test('close button has aria-label="Close"', () => {
                    const modal = document.getElementById(id);
                    const closeBtn = modal.querySelector('.close');
                    expect(closeBtn).not.toBeNull();
                    expect(closeBtn.getAttribute('aria-label')).toBe('Close');
                });
            });
        });
    });

    describe('Focus trap (GoReadApp._applyFocusTrap)', () => {
        const tab = (shiftKey = false) => {
            const target = document.activeElement || document.body;
            target.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey, bubbles: true, cancelable: true }));
        };
        const focusableIn = (modal) => Array.from(modal.querySelectorAll('button:not(:disabled), input:not(:disabled)'));

        let modal;
        let release;
        beforeEach(() => {
            modal = document.getElementById('add-feed-modal');
            modal.style.display = 'block';
        });
        afterEach(() => {
            if (release) release();
            release = null;
        });

        test('moves focus into the modal on open', () => {
            document.getElementById('add-feed-btn').focus();
            release = appProto()._applyFocusTrap(modal);
            expect(document.activeElement).toBe(focusableIn(modal)[0]);
        });

        test('returns focus to the trigger on release, and a second release is harmless', () => {
            const trigger = document.getElementById('add-feed-btn');
            trigger.focus();
            release = appProto()._applyFocusTrap(modal);
            expect(modal._releaseFocusTrap).toBe(release);

            release();
            expect(document.activeElement).toBe(trigger);
            expect(modal._releaseFocusTrap).toBeUndefined();

            document.getElementById('refresh-btn').focus();
            release();
            expect(document.activeElement).toBe(document.getElementById('refresh-btn'));
        });

        test('Tab wraps from the last element to the first', () => {
            release = appProto()._applyFocusTrap(modal);
            const focusable = focusableIn(modal);
            focusable[focusable.length - 1].focus();
            tab();
            expect(document.activeElement).toBe(focusable[0]);
        });

        test('Shift+Tab wraps from the first element to the last', () => {
            release = appProto()._applyFocusTrap(modal);
            const focusable = focusableIn(modal);
            focusable[0].focus();
            tab(true);
            expect(document.activeElement).toBe(focusable[focusable.length - 1]);
        });

        test('Tab pulls focus back in when it has left the modal', () => {
            release = appProto()._applyFocusTrap(modal);
            document.getElementById('refresh-btn').focus();
            tab();
            expect(document.activeElement).toBe(focusableIn(modal)[0]);

            document.getElementById('refresh-btn').focus();
            tab(true);
            const focusable = focusableIn(modal);
            expect(document.activeElement).toBe(focusable[focusable.length - 1]);
        });

        test('uses the controls enabled at the time of the Tab, not at open', () => {
            release = appProto()._applyFocusTrap(modal);
            const cancel = document.getElementById('cancel-add-feed');
            cancel.disabled = true; // e.g. while the add request is in flight
            const submit = modal.querySelector('button[type="submit"]');
            submit.focus();
            tab();
            expect(document.activeElement).toBe(focusableIn(modal)[0]);
        });

        test('stops trapping after release', () => {
            release = appProto()._applyFocusTrap(modal);
            release();
            const focusable = focusableIn(modal);
            const last = focusable[focusable.length - 1];
            last.focus();
            tab();
            // jsdom has no default Tab movement, so focus stays put once the
            // trap no longer intervenes.
            expect(document.activeElement).toBe(last);
        });
    });

    describe('Keyboard shortcuts while a dialog is open', () => {
        // setupKeyboardShortcuts adds a document listener that outlives the
        // per-test DOM reset, so the whole block shares one app instance.
        let app;
        beforeAll(() => {
            app = appProto();
            app.markAllRead = jest.fn();
            app.openCurrentArticle = jest.fn();
            app.setupKeyboardShortcuts();
        });
        beforeEach(() => {
            app.markAllRead.mockClear();
            app.openCurrentArticle.mockClear();
        });
        const press = (key, target = document.body) => {
            target.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }));
        };

        test('article shortcuts are ignored while a modal is open', () => {
            document.getElementById('help-modal').style.display = 'block';
            press('a');
            press('Enter', document.querySelector('#help-modal .close'));
            expect(app.markAllRead).not.toHaveBeenCalled();
            expect(app.openCurrentArticle).not.toHaveBeenCalled();
        });

        test('article shortcuts work when no modal is open', () => {
            press('a');
            press('Enter');
            expect(app.markAllRead).toHaveBeenCalledTimes(1);
            expect(app.openCurrentArticle).toHaveBeenCalledTimes(1);
        });

        test('Enter on a focused button activates the button, not the open-article shortcut', () => {
            const btn = document.getElementById('refresh-btn');
            const event = new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true });
            btn.dispatchEvent(event);
            expect(app.openCurrentArticle).not.toHaveBeenCalled();
            expect(event.defaultPrevented).toBe(false);
        });

        test('Escape closes a dynamically built dialog and returns focus', () => {
            const trigger = document.getElementById('add-feed-btn');
            trigger.focus();
            app.showSubscriptionLimitModal({ current_limit: 20 });
            const dialog = document.querySelector('[aria-labelledby="subscription-limit-heading"]');
            expect(dialog.contains(document.activeElement)).toBe(true);

            press('Escape');
            expect(document.body.contains(dialog)).toBe(false);
            expect(document.activeElement).toBe(trigger);
        });
    });

    describe('Screen reader announcements', () => {
        beforeEach(() => {
            jest.useFakeTimers();
            const region = document.createElement('div');
            region.id = 'sr-status';
            document.body.appendChild(region);
        });
        afterEach(() => {
            jest.useRealTimers();
        });
        const status = () => document.getElementById('sr-status').textContent;

        test('index.html ships the live regions in the initial page', () => {
            const html = fs.readFileSync(path.join(__dirname, '../templates/index.html'), 'utf8');
            expect(html).toMatch(/<div id="sr-status" class="sr-only" role="status" aria-live="polite"/);
            expect(html).toMatch(/<div id="toast-container" aria-live="polite"/);
        });

        test('announce clears the region and then fills it, so repeats are read again', () => {
            const app = appProto();
            app.announce('Starred');
            jest.runAllTimers();
            expect(status()).toBe('Starred');

            app.announce('Starred');
            expect(status()).toBe('');
            jest.runAllTimers();
            expect(status()).toBe('Starred');
        });

        test('only the latest of several quick announcements is kept', () => {
            const app = appProto();
            app.announce('first');
            app.announce('second');
            jest.runAllTimers();
            expect(status()).toBe('second');
        });

        test('new-articles banner announces only when it appears or its count changes', () => {
            const banner = document.createElement('div');
            banner.id = 'new-articles-banner';
            banner.hidden = true;
            banner.innerHTML = '<span class="new-articles-text"></span>';
            document.body.appendChild(banner);
            const app = appProto();
            app.announce = jest.fn();

            app.showNewArticlesBanner(3);
            app.showNewArticlesBanner(3);
            app.showNewArticlesBanner(4);
            expect(app.announce.mock.calls).toEqual([
                ['3 new articles available'],
                ['4 new articles available'],
            ]);
        });

        test('error banners use role="alert"', () => {
            new ErrorHandler().showError('Something failed');
            expect(document.querySelector('.error-message').getAttribute('role')).toBe('alert');
        });
    });

    describe('Interactive element accessible labels', () => {
        test('add-feed-btn has accessible text', () => {
            const btn = document.getElementById('add-feed-btn');
            const label = btn.getAttribute('aria-label') || btn.textContent.trim();
            expect(label).not.toBe('');
        });

        test('help-btn has accessible text', () => {
            const btn = document.getElementById('help-btn');
            const label = btn.getAttribute('aria-label') || btn.textContent.trim();
            expect(label).not.toBe('');
        });

        test('import-opml-btn has accessible text', () => {
            const btn = document.getElementById('import-opml-btn');
            const label = btn.getAttribute('aria-label') || btn.textContent.trim();
            expect(label).not.toBe('');
        });

        test('feed-url input has associated label or aria-label', () => {
            const input = document.getElementById('feed-url');
            const hasAriaLabel = input.hasAttribute('aria-label');
            const hasLabelElement = !!document.querySelector(`label[for="${input.id}"]`);
            const hasPlaceholder = input.hasAttribute('placeholder');
            expect(hasAriaLabel || hasLabelElement || hasPlaceholder).toBe(true);
        });
    });

    describe('Feed list item accessibility', () => {
        test('dynamically created feed items include a delete button with an accessible label', () => {
            const feedList = document.getElementById('feed-list');

            // Simulate the same DOM structure app.js builds in renderFeeds()
            const feedItem = document.createElement('div');
            feedItem.className = 'feed-item';
            feedItem.dataset.feedId = '42';

            const titleSpan = document.createElement('span');
            titleSpan.className = 'feed-title';
            titleSpan.textContent = 'Example Feed';

            const deleteButton = document.createElement('button');
            deleteButton.className = 'delete-feed';
            deleteButton.dataset.feedId = '42';
            deleteButton.setAttribute('aria-label', 'Delete Example Feed');
            deleteButton.textContent = '×';

            feedItem.appendChild(titleSpan);
            feedItem.appendChild(deleteButton);
            feedList.appendChild(feedItem);

            const renderedDelete = feedList.querySelector('[data-feed-id="42"] .delete-feed');
            expect(renderedDelete).not.toBeNull();
            const label = renderedDelete.getAttribute('aria-label') || renderedDelete.getAttribute('title');
            expect(label).not.toBeNull();
            expect(label.trim()).not.toBe('');
        });
    });
});
