/**
 * Article list scrolling tests.
 *
 * These load the real GoReadApp class from app.js and exercise
 * scrollArticleIntoView with stubbed element geometry, since jsdom does no
 * layout.
 */
const fs = require('fs');
const path = require('path');

// app.js declares its classes at the top level and only registers event
// listeners, so evaluating it in the jsdom window has no other side effects.
const appSource = fs.readFileSync(path.join(__dirname, '../static/js/app.js'), 'utf8');
const GoReadApp = new Function(`${appSource}\nreturn GoReadApp;`)();

// rect builds a DOMRect-like object from a top edge and a height.
function rect(top, height) {
    return { top, bottom: top + height, height, left: 0, right: 0, width: 0 };
}

// setup returns a scroll container whose viewport spans containerTop to
// containerTop + containerHeight, and an item whose rect is given relative
// to the page at the container's current scrollTop. Moving scrollTop moves
// the item's rect, as it would in a browser.
function setup({ containerTop = 100, containerHeight = 800, itemTop, itemHeight, scrollTop = 1000 }) {
    const container = document.createElement('div');
    const item = document.createElement('div');
    container.appendChild(item);
    // jsdom does no layout, so its scrollTop ignores writes. Use a plain
    // property instead.
    Object.defineProperty(container, 'scrollTop', { value: scrollTop, writable: true });
    const initialScrollTop = scrollTop;

    container.getBoundingClientRect = () => rect(containerTop, containerHeight);
    item.getBoundingClientRect = () => rect(itemTop - (container.scrollTop - initialScrollTop), itemHeight);

    const app = Object.create(GoReadApp.prototype);
    app.getScrollParent = () => container;
    return { app, container, item };
}

describe('scrollArticleIntoView', () => {
    test('a tall item below the viewport scrolls only until its top reaches the container top', () => {
        // The screenshot case: a 600px item starting 500px into an 800px
        // viewport. Two item-heights of lookahead would push its top 1300px
        // above the viewport.
        const { app, container, item } = setup({ itemTop: 600, itemHeight: 600 });

        app.scrollArticleIntoView(item);

        expect(item.getBoundingClientRect().top).toBeGreaterThanOrEqual(100);
        expect(item.getBoundingClientRect().bottom).toBeLessThanOrEqual(900);
        expect(container.scrollTop).toBeGreaterThan(1000);
    });

    test('an item taller than the viewport is aligned to the container top', () => {
        const { app, item } = setup({ itemTop: 700, itemHeight: 1200 });

        app.scrollArticleIntoView(item);

        expect(item.getBoundingClientRect().top).toBe(100);
    });

    test('a short item near the bottom keeps two item-heights of lookahead below it', () => {
        // An 80px item whose bottom is 20px above the viewport bottom.
        const { app, container, item } = setup({ itemTop: 800, itemHeight: 80 });

        app.scrollArticleIntoView(item);

        expect(item.getBoundingClientRect().bottom + 160).toBe(900);
        expect(container.scrollTop).toBe(1000 + 140);
    });

    test('an item above the viewport is aligned to the container top', () => {
        const { app, item } = setup({ itemTop: -200, itemHeight: 150 });

        app.scrollArticleIntoView(item);

        expect(item.getBoundingClientRect().top).toBe(100);
    });

    test('an item with room for its lookahead does not scroll', () => {
        const { app, container, item } = setup({ itemTop: 300, itemHeight: 100 });

        app.scrollArticleIntoView(item);

        expect(container.scrollTop).toBe(1000);
    });
});
