// Modal Manager - Lazy-loaded module for handling all modal interactions
// This module is dynamically imported on first modal use to reduce initial bundle size

export class ModalManager {
    constructor(app) {
        this.app = app;
        this._trapHandlers = new Map(); // modal element → keydown handler
        this._returnFocus = new Map();  // modal element → element to restore focus to
    }

    // Helper to get auth headers from parent app
    getAuthHeaders(includeContentType = false) {
        if (this.app && this.app.getAuthHeaders) {
            return this.app.getAuthHeaders(includeContentType);
        }
        return includeContentType ? {'Content-Type': 'multipart/form-data'} : {};
    }

    init() {
        console.log('ModalManager initialized');
    }

    _startFocusTrap(modal) {
        this._returnFocus.set(modal, document.activeElement);

        const focusable = modal.querySelectorAll(
            'a[href], button:not(:disabled), input:not(:disabled), ' +
            'select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])'
        );
        if (!focusable.length) return;

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
        this._trapHandlers.set(modal, handler);
    }

    _stopFocusTrap(modal) {
        const handler = this._trapHandlers.get(modal);
        if (handler) {
            modal.removeEventListener('keydown', handler);
            this._trapHandlers.delete(modal);
        }
        const returnTo = this._returnFocus.get(modal);
        if (returnTo && returnTo.focus) returnTo.focus();
        this._returnFocus.delete(modal);
    }

    showAddFeedModal() {
        const modal = document.getElementById('add-feed-modal');
        modal.style.display = 'block';
        this._startFocusTrap(modal);
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

        this._stopFocusTrap(modal);
        modal.style.display = 'none';
        form.reset();
    }

    showHelpModal() {
        const modal = document.getElementById('help-modal');
        modal.style.display = 'block';
        this._startFocusTrap(modal);
    }

    hideHelpModal() {
        const modal = document.getElementById('help-modal');
        this._stopFocusTrap(modal);
        modal.style.display = 'none';
    }

    showImportOpmlModal() {
        const modal = document.getElementById('import-opml-modal');
        modal.style.display = 'block';
        this._startFocusTrap(modal);
    }

    hideImportOpmlModal() {
        const modal = document.getElementById('import-opml-modal');
        const form = document.getElementById('import-opml-form');
        this._stopFocusTrap(modal);
        modal.style.display = 'none';
        form.reset();
    }

    async importOpml() {
        const fileInput = document.getElementById('opml-file');
        const submitButton = document.querySelector('#import-opml-form button[type="submit"]');
        const cancelButton = document.getElementById('cancel-import-opml');
        const originalText = submitButton.textContent;

        if (!fileInput.files || fileInput.files.length === 0) {
            this.app.showError('Please select an OPML file');
            return;
        }

        const file = fileInput.files[0];

        // Basic file validation
        if (file.size > 10 * 1024 * 1024) { // 10MB limit
            this.app.showError('File is too large (max 10MB)');
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

            // Get CSRF token from app and add to FormData or headers
            const headers = {};
            if (this.app && this.app.csrfToken) {
                headers['X-CSRF-Token'] = this.app.csrfToken;
            }

            const response = await fetch('/api/feeds/import', {
                method: 'POST',
                headers: headers,
                body: formData
            });

            if (response.ok) {
                const result = await response.json();
                this.hideImportOpmlModal();
                await this.app.loadFeeds();
                await this.app.loadSubscriptionInfo();
                await this.app.updateUnreadCounts();
                this.app.updateSubscriptionDisplay();

                // Show success message
                const message = `Successfully imported ${result.imported_count} feed(s) from OPML file`;
                this.app.showSuccess(message);
            } else if (response.status === 402) { // Payment Required
                const error = await response.json();
                if (error.limit_reached) {
                    // Show partial success if some feeds were imported
                    if (error.imported_count > 0) {
                        await this.app.loadFeeds();
                        await this.app.updateUnreadCounts();
                        this.app.showSuccess(`Imported ${error.imported_count} feed(s) before reaching your limit.`);
                    }
                    this.app.showSubscriptionLimitModal(error);
                } else if (error.trial_expired) {
                    this.app.showTrialExpiredModal(error);
                } else {
                    this.app.showError(error.error || 'Subscription required');
                }
            } else {
                let errorMessage = `HTTP ${response.status}`;
                try {
                    const error = await response.json();
                    errorMessage = error.error || errorMessage;
                } catch (e) {
                    // Use default error message
                }
                this.app.showError('Failed to import OPML: ' + errorMessage);
            }
        } catch (error) {
            this.app.showError('Failed to import OPML: ' + error.message);
        } finally {
            // Always restore form controls
            submitButton.disabled = false;
            submitButton.textContent = originalText;
            cancelButton.disabled = false;
            fileInput.disabled = false;
        }
    }
}
