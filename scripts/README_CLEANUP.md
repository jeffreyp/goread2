# Database Cleanup Guide

## Overview

This guide provides tools and procedures to clean up orphaned data in your GoRead2 production database. The recent commit "Fix orphaned feed data corrupting unread counts" indicates that your database has integrity issues that need to be resolved.

## Identified Issues

Based on the codebase analysis, your production database likely has:

1. **Orphaned Articles** - Articles pointing to deleted feeds
2. **Orphaned User Feeds** - User-feed associations pointing to deleted feeds or users
3. **Orphaned User Articles** - User-article relationships pointing to deleted articles or users
4. **Unused Feeds** - Feeds with no user associations (may be safe to keep)
5. **Corrupted Unread Counts** - Caused by orphaned datastore entities

## Safety First: Pre-Cleanup Steps

### 1. Test Locally First

```bash
# Test the cleanup approach on a local SQLite database
cd scripts/
go run test_cleanup_local.go setup
go run test_cleanup_local.go audit
go run test_cleanup_local.go cleanup
go run test_cleanup_local.go verify
```

### 2. Production Backup

**CRITICAL: Always backup before any cleanup!**

```bash
# Create a timestamped backup
export GOOGLE_CLOUD_PROJECT=your-project-id
go run backup_datastore.go $(date +%Y%m%d_%H%M%S)
```

## Production Cleanup Process

### Step 1: Audit Current State

```bash
# Run comprehensive audit to identify all issues
export GOOGLE_CLOUD_PROJECT=your-project-id
go run database_cleanup.go audit
```

This will show:
- Number of orphaned articles
- Number of orphaned user feeds
- Number of orphaned user articles
- Number of unused feeds
- Recommendations for cleanup

### Step 2: Dry Run Cleanup

```bash
# Preview what would be cleaned (NO CHANGES MADE)
go run database_cleanup.go cleanup --dry-run
```

Review the output carefully to ensure only orphaned data will be removed.

### Step 3: Execute Cleanup

```bash
# DESTRUCTIVE: Actually perform the cleanup
go run database_cleanup.go cleanup --execute
```

**WARNING**: This permanently deletes orphaned data. Ensure you have backups!

### Step 4: Verify Results

```bash
# Run audit again to confirm cleanup was successful
go run database_cleanup.go audit
```

Should show "Database integrity audit PASSED - No issues found!"

## What Each Script Does

### `test_cleanup_local.go`
- Creates a test SQLite database with known orphaned data
- Tests the cleanup logic safely before production use
- Validates that cleanup works correctly

### `backup_datastore.go`
- Creates backup copies of all entity types in Google Cloud Datastore
- Backup entities are stored with "_backup_SUFFIX" naming
- Essential safety measure before any destructive operations

### `database_cleanup.go`
- **audit**: Identifies all data integrity issues
- **cleanup --dry-run**: Shows what would be cleaned without making changes
- **cleanup --execute**: Performs actual cleanup (DESTRUCTIVE)
- **stats**: Shows database entity counts

## Safety Features

1. **Dry Run Mode**: Always preview changes before execution
2. **Confirmation Prompts**: Requires explicit "yes" confirmation for destructive operations
3. **Batch Processing**: Handles large datasets efficiently with batch operations
4. **Error Handling**: Continues cleanup even if some operations fail
5. **Detailed Logging**: Shows exactly what was cleaned and any errors

## Recovery Procedures

If something goes wrong during cleanup:

1. **Stop the application** to prevent further data corruption
2. **Restore from backup** using the backup entities created earlier
3. **Investigate the issue** before attempting cleanup again

## Expected Results

After successful cleanup:

1. **Improved Performance**: Unread count calculations will be faster and accurate
2. **Fixed UI Issues**: No more corrupted unread counts in the interface
3. **Cleaner Database**: Only valid, referenced data remains
4. **Better Reliability**: Reduced chance of future data integrity issues

## Monitoring Post-Cleanup

After cleanup, monitor:

1. **Unread Counts**: Verify they display correctly in the UI
2. **Feed Refresh**: Ensure feed updates work properly
3. **User Experience**: Check that all features work as expected
4. **Error Logs**: Watch for any new errors related to missing data

## Troubleshooting

### "Database locked" errors
- Ensure no other instances of the application are running
- Wait for any ongoing operations to complete

### "Permission denied" errors
- Verify your service account has Datastore admin permissions
- Check that GOOGLE_CLOUD_PROJECT is set correctly

### Large number of orphaned entities
- Run cleanup in smaller batches during low-traffic periods
- Consider running cleanup multiple times if timeouts occur

### Backup fails
- Check available storage space in your project
- Verify Datastore quotas haven't been exceeded

## Best Practices

1. **Schedule Regular Audits**: Run monthly audits to catch issues early
2. **Backup Before Major Changes**: Always backup before significant updates
3. **Test in Staging**: Test cleanup procedures on staging environment first
4. **Monitor After Changes**: Watch metrics and logs after any cleanup
5. **Document Issues**: Keep records of what issues were found and fixed

## Support

If you encounter issues:

1. Check the logs for specific error messages
2. Run the audit command to understand current state
3. Ensure all prerequisites are met (project ID, permissions, etc.)
4. Test the cleanup on a local SQLite database first

The cleanup tools are designed to be safe and comprehensive, but always prioritize caution when working with production data.