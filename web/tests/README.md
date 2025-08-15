# GoRead2 Frontend Testing

This directory contains comprehensive frontend tests for the GoRead2 RSS reader application.

## Test Structure

### Core Functionality Tests (`app-core.test.js`)
Tests the fundamental frontend functionality without requiring full application classes:

- **DOM Manipulation**: Feed list rendering, article list creation, modal handling
- **Event Handling**: Click events, keyboard shortcuts, event delegation
- **API Interaction**: Fetch mocking, error handling, subscription limits
- **Utility Functions**: HTML escaping, date formatting, unread count updates
- **UI State Management**: Active states, error/success messages
- **Form Validation**: File size validation, form submission states

### Test Utilities (`utils.js`)
Provides reusable testing utilities:
- `waitFor()` - Wait for async operations
- `fireEvent` - Simulate user interactions
- `createMockResponse()` - Mock API responses
- `createTestArticles()` - Generate test data
- `expectElementToHaveClass()` - DOM assertion helpers

### Test Setup (`setup.js`)
Configures the test environment with:
- Global fetch mocking
- DOM structure setup
- Default API responses
- Mock browser APIs

## Running Tests

```bash
# Install dependencies
npm install

# Run all tests
npm test

# Run with coverage
npm run test:coverage

# Run in watch mode
npm run test:watch
```

## Test Coverage

The tests cover:
- ✅ DOM manipulation and rendering
- ✅ Event handling and user interactions
- ✅ API mocking and error scenarios
- ✅ Form validation and submission
- ✅ UI state management
- ✅ Utility function validation
- ✅ Modal and dialog interactions
- ✅ Keyboard navigation
- ✅ Error handling and display

## Test Data

Test utilities provide realistic data:
- Sample RSS feeds with metadata
- Test articles with read/starred states
- Mock user authentication
- Subscription status scenarios

## Mocking Strategy

Tests use comprehensive mocking for:
- **Fetch API**: All HTTP requests are mocked
- **DOM Events**: Simulated user interactions
- **Browser APIs**: Window.location, localStorage, etc.
- **File APIs**: File upload simulation

## Architecture Notes

The frontend uses vanilla JavaScript with class-based architecture:
- `GoReadApp` - Main RSS reader application
- `AccountApp` - Account management interface

Tests focus on core functionality rather than class implementation to provide stable, maintainable test coverage that doesn't break when implementation details change.

## Adding New Tests

When adding new features:

1. Add test data generators to `utils.js` if needed
2. Create new test suites in `app-core.test.js`
3. Mock any new API endpoints in `setup.js`
4. Follow the existing pattern of testing behavior, not implementation

## Browser Compatibility

Tests use jsdom to simulate browser environment and are compatible with:
- Modern JavaScript features (ES6+)
- DOM APIs used by the application
- Event handling and delegation
- CSS selector queries

## Continuous Integration

Tests are designed to run in CI environments with:
- No external dependencies
- Deterministic behavior
- Fast execution (< 1 second)
- Clear error messages