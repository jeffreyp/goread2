# GoRead2 Multi-User Setup Guide

GoRead2 now supports multiple users with Google OAuth authentication. Follow these steps to set up the application.

## Prerequisites

- Go 1.21 or later
- Google account for OAuth setup

## Google OAuth Setup

1. **Create a Google OAuth Application**
   - Go to [Google Cloud Console](https://console.developers.google.com/)
   - Create a new project or select an existing one
   - Enable the Google+ API (for user profile information)
   - Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client IDs"
   - Choose "Web application"
   - Add authorized redirect URI: `http://localhost:8080/auth/callback`
   - Note down your Client ID and Client Secret

2. **Configure Environment Variables**
   ```bash
   cp .env.example .env
   ```
   
   Edit `.env` and fill in your Google OAuth credentials:
   ```
   GOOGLE_CLIENT_ID=your_actual_client_id
   GOOGLE_CLIENT_SECRET=your_actual_client_secret
   GOOGLE_REDIRECT_URL=http://localhost:8080/auth/callback
   ```

## Running the Application

1. **Install Dependencies**
   ```bash
   go mod tidy
   ```

2. **Set Environment Variables**
   ```bash
   # Load environment variables (Linux/Mac)
   export $(cat .env | xargs)
   
   # Or for Windows PowerShell
   Get-Content .env | ForEach-Object { 
     $name, $value = $_.split('='); 
     Set-Item -Path "env:$name" -Value $value 
   }
   ```

3. **Start the Server**
   ```bash
   go run main.go
   ```

4. **Access the Application**
   - Open your browser and go to `http://localhost:8080`
   - Click "Sign in with Google" to authenticate
   - Start adding RSS feeds!

## Features

- **Multi-user support**: Each user has their own feeds and read status
- **Google OAuth authentication**: Secure login with your Google account
- **User-specific data**: Read status and starred articles are per-user
- **Shared feeds**: Multiple users can subscribe to the same RSS feed
- **Session management**: Secure session handling with automatic cleanup

## Database Schema

The application uses SQLite by default with the following tables:

- `users`: User profiles from Google OAuth
- `feeds`: RSS feed definitions
- `articles`: Article content from feeds
- `user_feeds`: Many-to-many relationship for user feed subscriptions
- `user_articles`: Per-user read/starred status for articles

## Deployment

For production deployment:

1. Set `GOOGLE_REDIRECT_URL` to your production domain
2. Update authorized redirect URIs in Google Console
3. Consider using Google Cloud Datastore by setting `GOOGLE_CLOUD_PROJECT`
4. Use environment-specific configuration for security

## Troubleshooting

- **"OAuth configuration error"**: Check that all environment variables are set
- **"Authentication required"**: Make sure you're logged in via Google
- **"Failed to add feed"**: Check server logs for feed parsing errors
- **Database errors**: Ensure write permissions for SQLite database file