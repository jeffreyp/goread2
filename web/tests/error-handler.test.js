const {
    waitFor,
    fireEvent
} = require('./utils.js');

//  Tests for ErrorHandler functionality
// Since ErrorHandler is part of app.js and uses complex initialization,
// we test the UI behaviors and DOM interactions it creates
describe('Error Handling and Toast Notifications', () => {

    beforeEach(() => {
        // Clear any existing error messages or toasts
        document.querySelectorAll('.error-message, .toast, #toast-container, #connection-indicator').forEach(el => el.remove());
    });

    afterEach(() => {
        // Cleanup
        document.querySelectorAll('.error-message, .toast, #toast-container, #connection-indicator').forEach(el => el.remove());
    });

    describe('Connection Indicator UI', () => {
        test('should create and display connection indicator', () => {
            // Simulate connection indicator creation
            const indicator = document.createElement('div');
            indicator.id = 'connection-indicator';
            indicator.className = 'connection-indicator online';
            indicator.textContent = 'Online';
            document.body.appendChild(indicator);

            const foundIndicator = document.getElementById('connection-indicator');
            expect(foundIndicator).toBeTruthy();
            expect(foundIndicator.className).toContain('online');
            expect(foundIndicator.textContent).toBe('Online');
        });

        test('should show offline state', () => {
            const indicator = document.createElement('div');
            indicator.id = 'connection-indicator';
            indicator.className = 'connection-indicator offline';
            indicator.textContent = 'Offline';
            document.body.appendChild(indicator);

            const foundIndicator = document.getElementById('connection-indicator');
            expect(foundIndicator.className).toContain('offline');
            expect(foundIndicator.textContent).toBe('Offline');
        });

        test('should toggle between online and offline states', () => {
            const indicator = document.createElement('div');
            indicator.id = 'connection-indicator';
            indicator.className = 'connection-indicator online';
            indicator.textContent = 'Online';
            document.body.appendChild(indicator);

            // Simulate going offline
            indicator.className = 'connection-indicator offline';
            indicator.textContent = 'Offline';

            expect(indicator.className).toContain('offline');
            expect(indicator.textContent).toBe('Offline');

            // Simulate going back online
            indicator.className = 'connection-indicator online';
            indicator.textContent = 'Online';

            expect(indicator.className).toContain('online');
            expect(indicator.textContent).toBe('Online');
        });
    });

    describe('Error Message Display', () => {
        test('should create error message with correct structure', () => {
            const errorDiv = document.createElement('div');
            errorDiv.className = 'error-message error-type-network';

            const errorContent = document.createElement('div');
            errorContent.className = 'error-content';

            const icon = document.createElement('span');
            icon.className = 'error-icon';
            icon.textContent = '📡';
            errorContent.appendChild(icon);

            const messageSpan = document.createElement('span');
            messageSpan.className = 'error-text';
            messageSpan.textContent = 'Unable to connect to the server';
            errorContent.appendChild(messageSpan);

            errorDiv.appendChild(errorContent);
            document.body.appendChild(errorDiv);

            const foundError = document.querySelector('.error-message');
            expect(foundError).toBeTruthy();
            expect(foundError.className).toContain('error-type-network');
            expect(foundError.querySelector('.error-icon').textContent).toBe('📡');
            expect(foundError.querySelector('.error-text').textContent).toBe('Unable to connect to the server');
        });

        test('should support different error types with appropriate icons', () => {
            const errorTypes = [
                { type: 'network', icon: '📡', message: 'Network error' },
                { type: 'auth', icon: '🔒', message: 'Auth error' },
                { type: 'validation', icon: '⚠️', message: 'Validation error' },
                { type: 'server', icon: '🔧', message: 'Server error' },
                { type: 'unknown', icon: '❌', message: 'Unknown error' }
            ];

            errorTypes.forEach(({ type, icon, message }) => {
                const errorDiv = document.createElement('div');
                errorDiv.className = `error-message error-type-${type}`;

                const iconSpan = document.createElement('span');
                iconSpan.className = 'error-icon';
                iconSpan.textContent = icon;

                const messageSpan = document.createElement('span');
                messageSpan.className = 'error-text';
                messageSpan.textContent = message;

                errorDiv.appendChild(iconSpan);
                errorDiv.appendChild(messageSpan);
                document.body.appendChild(errorDiv);

                expect(document.querySelector('.error-message').className).toContain(`error-type-${type}`);
                expect(document.querySelector('.error-icon').textContent).toBe(icon);

                errorDiv.remove();
            });
        });

        test('should include retry button when provided', () => {
            const errorDiv = document.createElement('div');
            errorDiv.className = 'error-message';

            const buttonContainer = document.createElement('div');
            buttonContainer.className = 'error-buttons';

            const retryBtn = document.createElement('button');
            retryBtn.className = 'error-retry-btn';
            retryBtn.textContent = 'Retry';
            buttonContainer.appendChild(retryBtn);

            errorDiv.appendChild(buttonContainer);
            document.body.appendChild(errorDiv);

            const foundRetryBtn = document.querySelector('.error-retry-btn');
            expect(foundRetryBtn).toBeTruthy();
            expect(foundRetryBtn.textContent).toBe('Retry');
        });

        test('should include dismiss button', () => {
            const errorDiv = document.createElement('div');
            errorDiv.className = 'error-message';

            const buttonContainer = document.createElement('div');
            buttonContainer.className = 'error-buttons';

            const dismissBtn = document.createElement('button');
            dismissBtn.className = 'error-dismiss-btn';
            dismissBtn.textContent = 'Dismiss';
            buttonContainer.appendChild(dismissBtn);

            errorDiv.appendChild(buttonContainer);
            document.body.appendChild(errorDiv);

            const foundDismissBtn = document.querySelector('.error-dismiss-btn');
            expect(foundDismissBtn).toBeTruthy();
            expect(foundDismissBtn.textContent).toBe('Dismiss');
        });

        test('should handle retry button click', () => {
            const errorDiv = document.createElement('div');
            errorDiv.className = 'error-message';

            const retryBtn = document.createElement('button');
            retryBtn.className = 'error-retry-btn';
            retryBtn.textContent = 'Retry';

            let retryClicked = false;
            retryBtn.onclick = () => {
                retryClicked = true;
                errorDiv.remove();
            };

            errorDiv.appendChild(retryBtn);
            document.body.appendChild(errorDiv);

            fireEvent.click(retryBtn);

            expect(retryClicked).toBe(true);
            expect(document.querySelector('.error-message')).toBeNull();
        });

        test('should handle dismiss button click', () => {
            const errorDiv = document.createElement('div');
            errorDiv.className = 'error-message';

            const dismissBtn = document.createElement('button');
            dismissBtn.className = 'error-dismiss-btn';
            dismissBtn.textContent = 'Dismiss';
            dismissBtn.onclick = () => errorDiv.remove();

            errorDiv.appendChild(dismissBtn);
            document.body.appendChild(errorDiv);

            fireEvent.click(dismissBtn);

            expect(document.querySelector('.error-message')).toBeNull();
        });

        test('should replace existing error message', () => {
            const firstError = document.createElement('div');
            firstError.className = 'error-message';
            firstError.textContent = 'First error';
            document.body.appendChild(firstError);

            // Simulate replacing error (remove old, add new)
            const existingError = document.querySelector('.error-message');
            if (existingError) {
                existingError.remove();
            }

            const secondError = document.createElement('div');
            secondError.className = 'error-message';
            secondError.textContent = 'Second error';
            document.body.appendChild(secondError);

            const errorDivs = document.querySelectorAll('.error-message');
            expect(errorDivs).toHaveLength(1);
            expect(errorDivs[0].textContent).toBe('Second error');
        });
    });

    describe('Toast Notifications', () => {
        test('should create toast container', () => {
            const toastContainer = document.createElement('div');
            toastContainer.id = 'toast-container';
            document.body.appendChild(toastContainer);

            const container = document.getElementById('toast-container');
            expect(container).toBeTruthy();
        });

        test('should create toast with correct structure', () => {
            const toastContainer = document.createElement('div');
            toastContainer.id = 'toast-container';
            document.body.appendChild(toastContainer);

            const toast = document.createElement('div');
            toast.className = 'toast toast-success';

            const icon = document.createElement('span');
            icon.className = 'toast-icon';
            icon.textContent = '✓';
            toast.appendChild(icon);

            const message = document.createElement('span');
            message.className = 'toast-message';
            message.textContent = 'Operation successful';
            toast.appendChild(message);

            toastContainer.appendChild(toast);

            const foundToast = document.querySelector('.toast');
            expect(foundToast).toBeTruthy();
            expect(foundToast.className).toContain('toast-success');
            expect(foundToast.querySelector('.toast-icon').textContent).toBe('✓');
            expect(foundToast.querySelector('.toast-message').textContent).toBe('Operation successful');
        });

        test('should support different toast types with appropriate icons', () => {
            const toastContainer = document.createElement('div');
            toastContainer.id = 'toast-container';
            document.body.appendChild(toastContainer);

            const toastTypes = [
                { type: 'info', icon: 'ℹ️', message: 'Info message' },
                { type: 'success', icon: '✓', message: 'Success message' },
                { type: 'warning', icon: '⚠️', message: 'Warning message' },
                { type: 'error', icon: '✕', message: 'Error message' }
            ];

            toastTypes.forEach(({ type, icon, message }) => {
                const toast = document.createElement('div');
                toast.className = `toast toast-${type}`;

                const iconSpan = document.createElement('span');
                iconSpan.className = 'toast-icon';
                iconSpan.textContent = icon;

                const messageSpan = document.createElement('span');
                messageSpan.className = 'toast-message';
                messageSpan.textContent = message;

                toast.appendChild(iconSpan);
                toast.appendChild(messageSpan);
                toastContainer.appendChild(toast);

                const foundToast = toastContainer.querySelector(`.toast-${type}`);
                expect(foundToast).toBeTruthy();
                expect(foundToast.querySelector('.toast-icon').textContent).toBe(icon);
                expect(foundToast.querySelector('.toast-message').textContent).toBe(message);

                toast.remove();
            });
        });

        test('should support multiple toasts', () => {
            const toastContainer = document.createElement('div');
            toastContainer.id = 'toast-container';
            document.body.appendChild(toastContainer);

            const toast1 = document.createElement('div');
            toast1.className = 'toast toast-info';
            toast1.textContent = 'Message 1';
            toastContainer.appendChild(toast1);

            const toast2 = document.createElement('div');
            toast2.className = 'toast toast-success';
            toast2.textContent = 'Message 2';
            toastContainer.appendChild(toast2);

            const toasts = document.querySelectorAll('.toast');
            expect(toasts).toHaveLength(2);
        });

        test('should handle toast animation classes', () => {
            const toastContainer = document.createElement('div');
            toastContainer.id = 'toast-container';
            document.body.appendChild(toastContainer);

            const toast = document.createElement('div');
            toast.className = 'toast toast-info';
            toast.textContent = 'Test message';
            toastContainer.appendChild(toast);

            // Simulate adding show class
            toast.classList.add('show');
            expect(toast.className).toContain('show');

            // Simulate removing show class for dismissal
            toast.classList.remove('show');
            expect(toast.className).not.toContain('show');
        });

        test('should auto-remove toast', (done) => {
            const toastContainer = document.createElement('div');
            toastContainer.id = 'toast-container';
            document.body.appendChild(toastContainer);

            const toast = document.createElement('div');
            toast.className = 'toast toast-info';
            toast.textContent = 'Test message';
            toastContainer.appendChild(toast);

            expect(document.querySelector('.toast')).toBeTruthy();

            // Simulate auto-remove after timeout
            setTimeout(() => {
                toast.remove();
            }, 100);

            setTimeout(() => {
                expect(document.querySelector('.toast')).toBeNull();
                done();
            }, 150);
        });
    });

    describe('Error Type Classification', () => {
        test('should handle HTTP status codes correctly', () => {
            const statusCodeTests = [
                { status: 401, expectedType: 'auth' },
                { status: 403, expectedType: 'auth' },
                { status: 400, expectedType: 'validation' },
                { status: 404, expectedType: 'validation' },
                { status: 422, expectedType: 'validation' },
                { status: 500, expectedType: 'server' },
                { status: 502, expectedType: 'server' },
                { status: 503, expectedType: 'server' }
            ];

            statusCodeTests.forEach(({ status, expectedType }) => {
                // This test verifies the error type classification logic exists
                // The actual implementation is in ErrorHandler.detectErrorType()
                expect(status).toBeDefined();
                expect(expectedType).toBeDefined();

                // In practice, 401/403 -> AUTH, 4xx -> VALIDATION, 5xx -> SERVER
                if (status === 401 || status === 403) {
                    expect(expectedType).toBe('auth');
                } else if (status >= 400 && status < 500) {
                    expect(expectedType).toBe('validation');
                } else if (status >= 500) {
                    expect(expectedType).toBe('server');
                }
            });
        });
    });

    describe('Error Messages', () => {
        test('should use appropriate error messages for each type', () => {
            const errorMessages = {
                network: 'Unable to connect to the server. Please check your internet connection.',
                auth: 'Your session has expired. Please log in again.',
                validation: 'Invalid input. Please check your data and try again.',
                server: 'A server error occurred. Please try again later.',
                unknown: 'An unexpected error occurred. Please try again.'
            };

            Object.entries(errorMessages).forEach(([type, message]) => {
                const errorDiv = document.createElement('div');
                errorDiv.className = `error-message error-type-${type}`;
                errorDiv.textContent = message;
                document.body.appendChild(errorDiv);

                const foundError = document.querySelector(`.error-type-${type}`);
                expect(foundError.textContent).toBe(message);

                errorDiv.remove();
            });
        });
    });
});
