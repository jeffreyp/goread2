# GoRead2 - RSS Reader

A modern RSS reader inspired by Google Reader, built with Go and featuring a clean three-pane interface.

## Features

- **Three-pane layout**: Feed list → Article list → Article content (just like Google Reader)
- **RSS/Atom feed support**: Add and manage multiple RSS feeds
- **Real-time updates**: Background polling for new articles every 30 minutes
- **Article management**: Mark articles as read/unread, star favorites
- **Keyboard shortcuts**: Navigate efficiently with vim-like shortcuts
- **Clean UI**: Fast, responsive interface with modern design
- **Self-hosted**: No external dependencies, runs locally

## Screenshot

The interface features:
- **Left pane**: Feed subscriptions with unread counts
- **Center pane**: Article list with read/unread status
- **Right pane**: Full article content with original formatting

## Keyboard Shortcuts

- `j` - Next article
- `k` - Previous article  
- `o` / `Enter` - Open article in new tab
- `m` - Mark current article as read/unread
- `s` - Star/unstar current article
- `r` - Refresh all feeds

## Installation

### Local Development

#### Prerequisites
- Go 1.22 or later
- SQLite3 (automatically included with go-sqlite3)

#### Setup

1. **Clone or download the project:**
   ```bash
   git clone https://github.com/jeffreyp/goread2.git
   cd goread2
   ```

2. **Install dependencies:**
   ```bash
   go mod tidy
   ```

3. **Build the application:**
   ```bash
   go build -o goread2 .
   ```

4. **Run the server:**
   ```bash
   ./goread2
   ```

5. **Open in browser:**
   Navigate to `http://localhost:8080`

### Google App Engine Deployment

For production deployment on Google App Engine:

1. **Prerequisites:**
   - Google Cloud Project with billing enabled
   - Google Cloud SDK installed
   - App Engine and Datastore APIs enabled

2. **Deploy:**
   ```bash
   gcloud app deploy app.yaml
   gcloud app deploy cron.yaml
   ```

3. **Access:**
   ```bash
   gcloud app browse
   ```

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed deployment instructions.

## Usage

### Adding Feeds

1. Click the "Add Feed" button in the header
2. Enter the RSS/Atom feed URL
3. The feed will be fetched and articles imported automatically

### Reading Articles

1. Select a feed from the left panel to view its articles
2. Click "All Articles" to see articles from all feeds
3. Click on any article to read it in the right panel
4. Articles are automatically marked as read when clicked

### Managing Articles

- **Star articles**: Click the star icon next to any article
- **Mark as read/unread**: Use the keyboard shortcut `m` or the API
- **Delete feeds**: Hover over a feed and click the × button

### API Endpoints

The application provides a REST API:

- `GET /api/feeds` - List all feeds
- `POST /api/feeds` - Add new feed
- `DELETE /api/feeds/:id` - Delete feed
- `GET /api/feeds/:id/articles` - Get articles for feed
- `GET /api/feeds/all/articles` - Get all articles
- `POST /api/articles/:id/read` - Mark article read/unread
- `POST /api/articles/:id/star` - Toggle article star
- `POST /api/feeds/refresh` - Manually refresh all feeds

## Configuration

### Database
- **Local Development**: Uses SQLite and creates a `goread2.db` file in the current directory
- **Google App Engine**: Automatically uses Google Cloud Datastore for scalability and reliability

### Feed Refresh Interval
- **Local Development**: Background goroutine refreshes feeds every 30 minutes
- **Google App Engine**: Cron job (defined in `cron.yaml`) refreshes feeds every 30 minutes

### Port Configuration
The server runs on port 8080 by default. Change this in `main.go`:
```go
log.Fatal(r.Run(":8080")) // Change port here
```

## Project Structure

```
goread2/
├── main.go                 # Application entry point
├── internal/
│   ├── database/
│   │   └── schema.go       # Database models and initialization
│   ├── handlers/
│   │   └── feed_handler.go # HTTP request handlers
│   └── services/
│       └── feed_service.go # Business logic and RSS parsing
└── web/
    ├── templates/
    │   └── index.html      # Main HTML template
    └── static/
        ├── css/
        │   └── styles.css  # Google Reader-inspired CSS
        └── js/
            └── app.js      # Frontend JavaScript application
```

## Architecture

- **Backend**: Go with Gin web framework
- **Database**: SQLite3 for simplicity and portability
- **Frontend**: Vanilla JavaScript with modern CSS
- **RSS Parsing**: Built-in XML parsing for RSS/Atom feeds
- **Background Jobs**: Goroutine-based feed polling

## Development

### Adding New Features

1. **Database changes**: Modify `internal/database/schema.go`
2. **API endpoints**: Add handlers in `internal/handlers/`
3. **Business logic**: Extend services in `internal/services/`
4. **Frontend**: Update `web/static/js/app.js` and templates

### Testing Feeds

Popular RSS feeds for testing:
- `https://feeds.bbci.co.uk/news/rss.xml` - BBC News
- `https://rss.cnn.com/rss/edition.rss` - CNN News
- `https://feeds.feedburner.com/TechCrunch` - TechCrunch

## Troubleshooting

### Common Issues

**"Failed to fetch feed" error:**
- Check if the URL is a valid RSS/Atom feed
- Ensure the server can access external URLs
- Some feeds may require User-Agent headers (not currently implemented)

**Database locked error:**
- Stop any running instances of the application
- Delete `goread2.db` if corrupted (will lose data)

**Port already in use:**
- Change the port in `main.go` or stop the conflicting service
- Use `lsof -i :8080` to find what's using the port

## License

This project is open source. Feel free to modify and distribute.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## Inspiration

This project recreates the beloved Google Reader experience that was discontinued in 2013. The goal is to provide a fast, clean, and efficient RSS reading experience with the familiar three-pane layout that made Google Reader so popular among power users.
