# GitHub Repository Setup Instructions

Since GitHub CLI is not available, please follow these steps to create the GitHub repository and push the code:

## Step 1: Create GitHub Repository

1. **Go to GitHub.com and sign in**
2. **Click the "+" icon** in the top right corner
3. **Select "New repository"**
4. **Fill in repository details:**
   - Repository name: `goread2`
   - Description: `Google Reader inspired RSS reader built with Go and App Engine`
   - Set to Public or Private (your choice)
   - **DO NOT** initialize with README, .gitignore, or license (we already have these)

## Step 2: Get Repository URL

After creating the repository, you'll see a page with setup instructions. Copy the repository URL, which will look like:
```
https://github.com/YOUR_USERNAME/goread2.git
```

## Step 3: Add Remote and Push

Run these commands in your terminal (replace YOUR_USERNAME with your GitHub username):

```bash
# Add the GitHub repository as remote origin
git remote add origin https://github.com/YOUR_USERNAME/goread2.git

# Push the code to GitHub
git push -u origin main
```

## Alternative: Using SSH (if you have SSH keys set up)

If you prefer SSH and have SSH keys configured:

```bash
# Add remote with SSH URL
git remote add origin git@github.com:YOUR_USERNAME/goread2.git

# Push to GitHub
git push -u origin main
```

## Verification

After pushing, you should see all files in your GitHub repository:

- README.md with project description
- DEPLOYMENT.md with App Engine instructions  
- Source code in internal/ directory
- Web assets in web/ directory
- App Engine configuration files (app.yaml, cron.yaml)

## Repository Features to Enable

Consider enabling these GitHub features:

1. **Issues** - For bug tracking and feature requests
2. **Actions** - For CI/CD (optional)
3. **Security** - Dependabot for dependency updates
4. **Pages** - For documentation (optional)

## Repository Topics

Add these topics to help others discover your repository:
- `rss-reader`
- `google-reader`
- `golang`
- `app-engine`
- `web-application`
- `feed-aggregator`

The repository is now ready to be pushed to GitHub!