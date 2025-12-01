/**
 * Animation utilities for GoRead2
 * Provides micro-interactions and smooth animations throughout the app
 */

class AnimationManager {
    constructor() {
        this.init();
    }

    init() {
        this.setupHeaderScrollShadow();
        this.setupModalAnimations();
        this.setupRippleEffects();
        this.setupScrollToTop();
    }

    /**
     * Add shadow to header on scroll
     */
    setupHeaderScrollShadow() {
        const header = document.querySelector('.header');
        if (!header) return;

        const observeScrollPanes = () => {
            // Observe content pane scroll for desktop
            const contentPane = document.querySelector('.content-pane');
            if (contentPane) {
                contentPane.addEventListener('scroll', () => {
                    if (contentPane.scrollTop > 10) {
                        header.classList.add('scrolled');
                    } else {
                        header.classList.remove('scrolled');
                    }
                });
            }

            // Observe article pane scroll
            const articlePane = document.querySelector('.article-pane');
            if (articlePane) {
                articlePane.addEventListener('scroll', () => {
                    if (articlePane.scrollTop > 10) {
                        header.classList.add('scrolled');
                    } else {
                        header.classList.remove('scrolled');
                    }
                });
            }

            // Observe main window scroll
            window.addEventListener('scroll', () => {
                if (window.scrollY > 10) {
                    header.classList.add('scrolled');
                } else {
                    header.classList.remove('scrolled');
                }
            });
        };

        observeScrollPanes();
    }

    /**
     * Animate modals with proper show/hide transitions
     */
    setupModalAnimations() {
        // Override modal display to add animation class
        const originalDisplayStyle = CSSStyleDeclaration.prototype.setProperty;

        // Monitor modal visibility changes
        const observer = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                if (mutation.type === 'attributes' && mutation.attributeName === 'style') {
                    const modal = mutation.target;
                    if (modal.classList.contains('modal')) {
                        if (modal.style.display === 'block') {
                            modal.classList.add('show');
                        } else if (modal.style.display === 'none') {
                            modal.classList.remove('show');
                        }
                    }
                }
            });
        });

        // Observe all modals
        document.querySelectorAll('.modal').forEach(modal => {
            observer.observe(modal, { attributes: true, attributeFilter: ['style'] });
        });
    }

    /**
     * Add ripple effect to tappable elements on mobile
     */
    setupRippleEffects() {
        const createRipple = (event) => {
            const button = event.currentTarget;

            // Don't add ripple if reduced motion is preferred
            if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
                return;
            }

            const circle = document.createElement('span');
            const diameter = Math.max(button.clientWidth, button.clientHeight);
            const radius = diameter / 2;

            const rect = button.getBoundingClientRect();
            circle.style.width = circle.style.height = `${diameter}px`;
            circle.style.left = `${event.clientX - rect.left - radius}px`;
            circle.style.top = `${event.clientY - rect.top - radius}px`;
            circle.classList.add('ripple-effect');

            // Remove existing ripples
            const existingRipple = button.querySelector('.ripple-effect');
            if (existingRipple) {
                existingRipple.remove();
            }

            button.appendChild(circle);

            // Remove ripple after animation
            setTimeout(() => {
                circle.remove();
            }, 600);
        };

        // Add ripple to buttons and interactive elements
        const selectors = [
            '.btn',
            '.mobile-nav-btn',
            '.article-item',
            '.feed-item',
            '.action-btn'
        ];

        selectors.forEach(selector => {
            document.querySelectorAll(selector).forEach(element => {
                element.addEventListener('click', createRipple);
            });
        });

        // Use MutationObserver to add ripple to dynamically added elements
        const observer = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                mutation.addedNodes.forEach((node) => {
                    if (node.nodeType === 1) { // Element node
                        selectors.forEach(selector => {
                            if (node.matches && node.matches(selector)) {
                                node.addEventListener('click', createRipple);
                            }
                            // Check children
                            node.querySelectorAll && node.querySelectorAll(selector).forEach(element => {
                                element.addEventListener('click', createRipple);
                            });
                        });
                    }
                });
            });
        });

        observer.observe(document.body, { childList: true, subtree: true });
    }

    /**
     * Animate article items when they're added to the list
     */
    animateArticleItem(articleElement, delay = 0) {
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
            return;
        }

        setTimeout(() => {
            articleElement.classList.add('fade-in');
        }, delay);
    }

    /**
     * Animate article removal (e.g., when marking as read)
     */
    animateArticleRemoval(articleElement, callback) {
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
            if (callback) callback();
            return;
        }

        articleElement.classList.add('slide-out');
        setTimeout(() => {
            if (callback) callback();
        }, 300);
    }

    /**
     * Bounce animation for starring articles
     */
    animateStar(starButton) {
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
            return;
        }

        starButton.classList.add('bounce');
        setTimeout(() => {
            starButton.classList.remove('bounce');
        }, 400);
    }

    /**
     * Animate unread count badge changes
     */
    animateBadgeChange(badgeElement) {
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
            return;
        }

        badgeElement.classList.add('badge-change');
        setTimeout(() => {
            badgeElement.classList.remove('badge-change');
        }, 400);
    }

    /**
     * Pulse animation for new articles notification
     */
    pulseNewArticles(badgeElement) {
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
            return;
        }

        badgeElement.classList.add('pulse');
        // Remove pulse after 3 cycles
        setTimeout(() => {
            badgeElement.classList.remove('pulse');
        }, 3000);
    }

    /**
     * Show success checkmark animation
     */
    showSuccessCheckmark(container) {
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
            return;
        }

        const checkmark = document.createElement('div');
        checkmark.className = 'success-checkmark';
        checkmark.textContent = '✓';
        container.style.position = 'relative';
        container.appendChild(checkmark);

        setTimeout(() => {
            checkmark.remove();
        }, 600);
    }

    /**
     * Shake animation for errors
     */
    shakeElement(element) {
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
            return;
        }

        element.classList.add('animate-shake');
        setTimeout(() => {
            element.classList.remove('animate-shake');
        }, 500);
    }

    /**
     * Setup smooth scroll to top button
     */
    setupScrollToTop() {
        // Create scroll to top button
        const scrollBtn = document.createElement('button');
        scrollBtn.innerHTML = '↑';
        scrollBtn.className = 'scroll-to-top';
        scrollBtn.setAttribute('aria-label', 'Scroll to top');
        scrollBtn.style.cssText = `
            position: fixed;
            bottom: 80px;
            right: 20px;
            width: 48px;
            height: 48px;
            border-radius: 50%;
            background-color: #1a73e8;
            color: white;
            border: none;
            font-size: 24px;
            cursor: pointer;
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
            z-index: 999;
            opacity: 0;
            visibility: hidden;
            transition: opacity 0.3s ease, visibility 0.3s ease, transform 0.2s ease;
            display: none;
        `;

        document.body.appendChild(scrollBtn);

        const showScrollBtn = () => {
            const contentPane = document.querySelector('.content-pane');
            const scrollY = contentPane ? contentPane.scrollTop : window.scrollY;

            if (scrollY > 500) {
                scrollBtn.style.display = 'block';
                setTimeout(() => {
                    scrollBtn.style.opacity = '1';
                    scrollBtn.style.visibility = 'visible';
                }, 10);
            } else {
                scrollBtn.style.opacity = '0';
                scrollBtn.style.visibility = 'hidden';
                setTimeout(() => {
                    scrollBtn.style.display = 'none';
                }, 300);
            }
        };

        // Listen to scroll events
        const contentPane = document.querySelector('.content-pane');
        if (contentPane) {
            contentPane.addEventListener('scroll', showScrollBtn);
        }
        window.addEventListener('scroll', showScrollBtn);

        // Scroll to top on click
        scrollBtn.addEventListener('click', () => {
            const contentPane = document.querySelector('.content-pane');
            if (contentPane) {
                contentPane.scrollTo({ top: 0, behavior: 'smooth' });
            } else {
                window.scrollTo({ top: 0, behavior: 'smooth' });
            }
        });

        // Add hover effect
        scrollBtn.addEventListener('mouseenter', () => {
            scrollBtn.style.transform = 'scale(1.1)';
        });
        scrollBtn.addEventListener('mouseleave', () => {
            scrollBtn.style.transform = 'scale(1)';
        });

        // Add active effect
        scrollBtn.addEventListener('mousedown', () => {
            scrollBtn.style.transform = 'scale(0.95)';
        });
        scrollBtn.addEventListener('mouseup', () => {
            scrollBtn.style.transform = 'scale(1.1)';
        });
    }

    /**
     * Fade in content pane when switching articles
     */
    fadeInContent(contentElement) {
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
            return;
        }

        contentElement.style.opacity = '0';
        setTimeout(() => {
            contentElement.style.transition = 'opacity 0.3s ease';
            contentElement.style.opacity = '1';
        }, 10);
    }
}

// Initialize animation manager when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.animationManager = new AnimationManager();
    });
} else {
    window.animationManager = new AnimationManager();
}

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = AnimationManager;
}
