// Modal Manager - Lazy-loaded module for handling all modal interactions
// This module is dynamically imported on first modal use to reduce initial bundle size

export class ModalManager {
    constructor(app) {
        this.app = app;
    }

    init() {
        console.log('ModalManager initialized');
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

            const response = await fetch('/api/feeds/import', {
                method: 'POST',
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
