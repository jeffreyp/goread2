/**
 * Integration layer to connect animations with the main GoRead app
 * This file hooks into existing app functionality to trigger animations
 */

(function() {
    'use strict';

    // Wait for both DOM and animation manager to be ready
    const init = () => {
        if (!window.animationManager) {
            setTimeout(init, 100);
            return;
        }

        setupArticleAnimations();
        setupStarAnimations();
        setupBadgeAnimations();
        setupMarkReadAnimations();
        setupModalAnimations();
        setupFormValidation();
    };

    /**
     * Hook into article rendering to add fade-in animations
     */
    function setupArticleAnimations() {
        // Use MutationObserver to detect new articles being added
        const articleList = document.getElementById('article-list');
        if (!articleList) return;

        const observer = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                mutation.addedNodes.forEach((node) => {
                    if (node.classList && node.classList.contains('article-item')) {
                        // Add fade-in animation with stagger effect
                        const articles = articleList.querySelectorAll('.article-item');
                        const index = Array.from(articles).indexOf(node);
                        const delay = Math.min(index * 30, 300); // Max 300ms delay
                        window.animationManager.animateArticleItem(node, delay);
                    }
                });
            });
        });

        observer.observe(articleList, { childList: true });
    }

    /**
     * Hook into star button clicks to add bounce animation
     */
    function setupStarAnimations() {
        document.addEventListener('click', (e) => {
            // Check if clicked element is a star button or its parent
            const starBtn = e.target.closest('.action-btn[title*="star"], .action-btn[title*="Star"]');
            if (starBtn) {
                window.animationManager.animateStar(starBtn);
            }
        });
    }

    /**
     * Hook into unread count changes to add badge animations
     */
    function setupBadgeAnimations() {
        // Observe all unread count badges for changes
        const observeBadge = (badge) => {
            let previousCount = badge.textContent;

            const observer = new MutationObserver(() => {
                const currentCount = badge.textContent;
                if (currentCount !== previousCount) {
                    window.animationManager.animateBadgeChange(badge);

                    // If count increased, add pulse animation
                    const prevNum = parseInt(previousCount) || 0;
                    const currNum = parseInt(currentCount) || 0;
                    if (currNum > prevNum) {
                        window.animationManager.pulseNewArticles(badge);
                    }

                    previousCount = currentCount;
                }
            });

            observer.observe(badge, {
                childList: true,
                characterData: true,
                subtree: true,
                attributes: true,
                attributeFilter: ['data-count']
            });
        };

        // Observe existing badges
        document.querySelectorAll('.unread-count').forEach(observeBadge);

        // Observe new badges being added
        const listObserver = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                mutation.addedNodes.forEach((node) => {
                    if (node.classList && node.classList.contains('unread-count')) {
                        observeBadge(node);
                    }
                    if (node.querySelectorAll) {
                        node.querySelectorAll('.unread-count').forEach(observeBadge);
                    }
                });
            });
        });

        listObserver.observe(document.body, { childList: true, subtree: true });
    }

    /**
     * Hook into mark as read functionality to add slide-out animation
     */
    function setupMarkReadAnimations() {
        // This is a bit tricky since we need to intercept the removal
        // We'll use a MutationObserver on the article list
        const articleList = document.getElementById('article-list');
        if (!articleList) return;

        // Store reference to original remove method
        const originalRemove = Element.prototype.remove;

        // Override remove for article items
        Element.prototype.remove = function() {
            if (this.classList && this.classList.contains('article-item')) {
                // Only animate if unread filter is active
                const unreadFilter = document.querySelector('input[name="article-filter"][value="unread"]');
                if (unreadFilter && unreadFilter.checked) {
                    window.animationManager.animateArticleRemoval(this, () => {
                        originalRemove.call(this);
                    });
                    return;
                }
            }
            originalRemove.call(this);
        };
    }

    /**
     * Add animations to modal show/hide
     */
    function setupModalAnimations() {
        const modals = document.querySelectorAll('.modal');

        modals.forEach(modal => {
            // Store original display style
            const originalDisplay = modal.style.display;

            // Watch for style changes
            const observer = new MutationObserver((mutations) => {
                mutations.forEach((mutation) => {
                    if (mutation.attributeName === 'style') {
                        const currentDisplay = modal.style.display;

                        if (currentDisplay === 'block' && originalDisplay !== 'block') {
                            // Modal is being shown
                            modal.classList.add('show');

                            // Fade in the content pane when article is opened
                            if (modal.classList.contains('content-pane')) {
                                window.animationManager.fadeInContent(modal);
                            }
                        } else if (currentDisplay === 'none' || currentDisplay === '') {
                            // Modal is being hidden
                            setTimeout(() => {
                                modal.classList.remove('show');
                            }, 10);
                        }
                    }
                });
            });

            observer.observe(modal, { attributes: true, attributeFilter: ['style'] });
        });
    }

    /**
     * Add shake animation to form inputs on validation error
     */
    function setupFormValidation() {
        document.addEventListener('submit', (e) => {
            const form = e.target;
            if (!form.checkValidity()) {
                e.preventDefault();

                // Shake invalid inputs
                form.querySelectorAll(':invalid').forEach(input => {
                    window.animationManager.shakeElement(input);

                    // Also shake the parent form group if it exists
                    const formGroup = input.closest('.form-group');
                    if (formGroup) {
                        window.animationManager.shakeElement(formGroup);
                    }
                });
            }
        });

        // Also listen for invalid events
        document.addEventListener('invalid', (e) => {
            e.preventDefault();
            const input = e.target;
            window.animationManager.shakeElement(input);
        }, true);
    }

    /**
     * Add fade-in animation when content pane updates
     */
    function setupContentPaneAnimations() {
        const contentPane = document.getElementById('article-content');
        if (!contentPane) return;

        const observer = new MutationObserver((mutations) => {
            // Check if significant content was added (not just small updates)
            const significantChange = mutations.some(m =>
                m.addedNodes.length > 0 &&
                Array.from(m.addedNodes).some(n =>
                    n.nodeType === 1 && n.querySelector && n.querySelector('h1, .content')
                )
            );

            if (significantChange) {
                window.animationManager.fadeInContent(contentPane);
            }
        });

        observer.observe(contentPane, { childList: true, subtree: true });
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Also setup content pane animations
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', setupContentPaneAnimations);
    } else {
        setupContentPaneAnimations();
    }

})();
