# Features Guide

Complete guide to GoRead2's features and how to use them effectively.

## Reading Experience

### Font Choice
GoRead2 offers reading-optimized typography to enhance your reading experience:

- **Sans-serif mode**: Uses Inter font, optimized for screen reading with excellent legibility
- **Serif mode**: Uses Georgia font, providing a traditional reading experience
- **Toggle button**: Click the "Aa" button in the header or press `f` to switch between fonts
- **Persistent preference**: Your font choice is automatically saved and restored across sessions

The font setting only applies to article content, while the interface maintains consistent UI typography.

## Navigation

### Three-Pane Layout
- **Feed Pane (Left)**: Shows your subscribed feeds with unread counts
- **Article Pane (Center)**: Lists articles from the selected feed
- **Content Pane (Right)**: Displays the full article content

### Keyboard Shortcuts
GoRead2 provides efficient keyboard navigation:

| Shortcut | Action |
|----------|--------|
| `j` | Next article |
| `k` | Previous article |
| `o` or `Enter` | Open article in new tab |
| `m` | Mark article as read/unread |
| `s` | Star/unstar article |
| `r` | Refresh all feeds |
| `f` | Toggle font style (sans-serif ↔ serif) |

### Mobile and Tablet Navigation

#### Phone (Portrait Mode)
On mobile devices, use the bottom navigation bar to switch between panes:
- 📑 **Feeds**: View your feed subscriptions
- 📄 **Articles**: Browse articles from selected feed
- 📖 **Content**: Read the selected article

#### Tablet (Portrait Mode)
On tablets in portrait orientation, GoRead2 provides a reading-optimized layout:
- **Content pane** takes the full screen width for comfortable reading
- **Toggle button** (☰) appears in the bottom-right corner to access feeds and articles
- Tap the toggle button to show/hide the sidebar with feeds and articles
- Sidebar automatically hides when you select an article to maximize reading space
- Tap the dimmed area outside the sidebar to close it

## Feed Management

### Adding Feeds
1. Click "Add Feed" in the header
2. Enter a website URL or direct RSS feed URL
3. GoRead2 will automatically discover the RSS feed
4. The new feed appears in your feed list with recent articles

### Article Import Limits
When subscribing to a new feed, GoRead2 intelligently limits the number of articles imported to improve performance:

- **Default**: 100 most recent articles are imported as unread
- **Configurable**: Change this limit in Account Settings (0-10,000 articles)
- **Setting 0**: Import unlimited articles (use carefully with large feeds)
- **Purpose**: Prevents overwhelming your reading list with thousands of old articles

To adjust your import limit:
1. Go to `/account` or click your profile
2. Find the "Settings" section
3. Set your preferred "Maximum articles to import when adding a new feed"
4. Click "Save" to apply the setting

### OPML Import
Import feeds from other RSS readers:
1. Click "Import OPML" in the header
2. Select your exported OPML file (max 10MB)
3. GoRead2 imports all feeds and starts fetching articles

### Feed Subscription Limits
- **Free Trial**: 20 feeds for 30 days
- **GoRead2 Pro**: Unlimited feeds
- **Admin**: Unlimited access

## Article Management

### Read Status
- Articles are automatically marked as read when you navigate away
- Manually toggle read status with `m` key or the toggle button
- Filter articles by "Unread" or "All" in the article pane header

### Starring Articles
- Star important articles with `s` key or the star button (★)
- Starred articles are highlighted and easily accessible
- Use stars to bookmark articles for later reference

### Article Filtering
Use the radio buttons in the article pane header:
- **Unread**: Show only unread articles (default)
- **All**: Show all articles regardless of read status

## User Interface Features

### Real-time Updates
- Unread counts update automatically every 5 minutes
- Manual refresh with the "Refresh" button or `r` key
- Background polling keeps your feeds current

### Responsive Design
- Optimized for desktop, tablet, and mobile devices
- Touch-friendly interface on mobile devices
- Adaptive layout that works across screen sizes

### Privacy & Security
- All feed subscriptions and article status are private to your account
- Secure Google OAuth authentication
- No tracking or data sharing with third parties

## Subscription Features

### Trial Period
- 30-day free trial with up to 20 feeds
- Full access to all features during trial
- Automatic upgrade prompts when approaching limits

### GoRead2 Pro
- Unlimited RSS feeds
- Continued access to all features
- Priority support
- $2.99/month subscription

### Account Management
- View subscription status in the header
- Manage billing and subscription at `/account`
- Cancel or upgrade subscription at any time

## Tips for Best Experience

### Feed Organization
- Use descriptive feed names when adding feeds
- Regularly review and remove inactive feeds
- Take advantage of keyboard shortcuts for faster navigation

### Reading Workflow
1. Check "All Articles" for a quick overview
2. Use `j/k` keys to quickly scan headlines
3. Press `Enter` to open interesting articles in new tabs
4. Star articles you want to reference later
5. Use font toggle (`f`) to optimize reading comfort

### Performance
- GoRead2 automatically optimizes for performance
- Articles load progressively for faster browsing
- Unread counts sync in the background