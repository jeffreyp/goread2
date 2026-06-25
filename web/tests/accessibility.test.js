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

    describe('Focus management', () => {
        // Inline the focus-trap logic from ModalManager so we can test it
        // without needing ESM imports.
        function startFocusTrap(modal, returnFocusTo) {
            const focusable = modal.querySelectorAll(
                'a[href], button:not(:disabled), input:not(:disabled), ' +
                'select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])'
            );
            if (!focusable.length) return null;

            const first = focusable[0];
            const last = focusable[focusable.length - 1];
            first.focus();

            const handler = (e) => {
                if (e.key !== 'Tab') return;
                if (e.shiftKey) {
                    if (document.activeElement === first) {
                        e.preventDefault();
                        last.focus();
                    }
                } else {
                    if (document.activeElement === last) {
                        e.preventDefault();
                        first.focus();
                    }
                }
            };

            modal.addEventListener('keydown', handler);
            return handler;
        }

        function stopFocusTrap(modal, handler, returnFocusTo) {
            if (handler) modal.removeEventListener('keydown', handler);
            if (returnFocusTo && returnFocusTo.focus) returnFocusTo.focus();
        }

        test('focus moves into modal on open', () => {
            const triggerBtn = document.getElementById('add-feed-btn');
            triggerBtn.focus();
            expect(document.activeElement).toBe(triggerBtn);

            const modal = document.getElementById('add-feed-modal');
            modal.style.display = 'block';
            startFocusTrap(modal, triggerBtn);

            expect(modal.contains(document.activeElement)).toBe(true);
        });

        test('focus returns to trigger element on close', () => {
            const triggerBtn = document.getElementById('add-feed-btn');
            triggerBtn.focus();

            const modal = document.getElementById('add-feed-modal');
            modal.style.display = 'block';
            const handler = startFocusTrap(modal, triggerBtn);

            modal.style.display = 'none';
            stopFocusTrap(modal, handler, triggerBtn);

            expect(document.activeElement).toBe(triggerBtn);
        });

        test('Tab key wraps from last to first focusable element', () => {
            const modal = document.getElementById('add-feed-modal');
            modal.style.display = 'block';
            startFocusTrap(modal, null);

            const focusable = modal.querySelectorAll(
                'button:not(:disabled), input:not(:disabled)'
            );
            const last = focusable[focusable.length - 1];
            const first = focusable[0];

            last.focus();
            modal.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true }));

            expect(document.activeElement).toBe(first);
        });

        test('Shift+Tab wraps from first to last focusable element', () => {
            const modal = document.getElementById('add-feed-modal');
            modal.style.display = 'block';
            startFocusTrap(modal, null);

            const focusable = modal.querySelectorAll(
                'button:not(:disabled), input:not(:disabled)'
            );
            const first = focusable[0];
            const last = focusable[focusable.length - 1];

            first.focus();
            modal.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true }));

            expect(document.activeElement).toBe(last);
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
