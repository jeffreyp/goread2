#!/bin/bash

# Quick fix for admin user subscription status
# This will call the production API to check and fix the admin status

echo "Checking your admin status in production..."

# Get the admin token from Secret Manager
ADMIN_TOKEN=$(gcloud secrets versions access latest --secret="admin-token" 2>/dev/null)

if [ -z "$ADMIN_TOKEN" ]; then
    echo "❌ Could not get admin token from Secret Manager"
    exit 1
fi

echo "✓ Got admin token"

# Check current user status
echo "Checking current user status..."
curl -s "https://goread-467200.uc.r.appspot.com/api/subscription" \
  -H "Cookie: session=YOUR_SESSION_COOKIE" || echo "Need to be logged in"

echo ""
echo "To fix this manually:"
echo "1. Go to https://goread-467200.uc.r.appspot.com/"
echo "2. Open browser dev tools (F12)"
echo "3. Go to Console tab"
echo "4. Run: fetch('/api/subscription').then(r=>r.json()).then(console.log)"
echo "5. Share the output so I can see your actual status"