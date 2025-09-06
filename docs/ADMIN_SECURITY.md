# Admin Security Guide

## Overview

The GoRead2 admin system uses a secure database-based token authentication system to prevent unauthorized access. All admin tokens are cryptographically generated, hashed, and stored in the database.

## Security Vulnerability Fixed

**Previous Issue**: The original admin system only required setting environment variables `ADMIN_TOKEN` and `ADMIN_TOKEN_VERIFY` to the same value, allowing anyone with server access to create admin tokens.

**Current Solution**: Database-based token system:
1. Cryptographically secure 64-character (32-byte) random tokens
2. SHA-256 hashed storage in database
3. Token validation against database records
4. Token revocation and usage tracking
5. Bootstrap protection with security warnings

## Token Management System

### Admin Token Properties
- **Format**: 64-character hexadecimal string (32 random bytes)
- **Storage**: SHA-256 hash stored in `admin_tokens` table
- **Security**: Tokens are never stored in plain text
- **Tracking**: Creation time, last used time, and active/revoked status
- **Descriptions**: Human-readable descriptions for token identification

### ADMIN_TOKEN Environment Variable
- **Purpose**: Contains the actual admin token for authentication
- **Format**: Exactly 64 hexadecimal characters
- **Security**: Validated against database on every use
- **Usage**: Required for all admin commands except initial bootstrap

## Deployment Architecture

### Database Compatibility
The admin token system works with both database backends:

**SQLite (Local Development)**:
- Admin tokens stored in `admin_tokens` table
- Direct SQL queries for validation and management
- File-based database persistence

**Google Datastore (Production/GAE)**:
- Admin tokens stored as `AdminToken` entities
- Datastore queries with filters for validation
- Cloud-native scaling and persistence
- Automatic detection via `GOOGLE_CLOUD_PROJECT` environment variable

## Initial Setup (Bootstrap)

### Local Development (SQLite)

#### Step 1: Create First User and Admin
1. **Start the web application** and create your first user account through the normal signup process
2. **Set the user as admin** by directly updating the database:
   ```bash
   sqlite3 goread2.db "UPDATE users SET is_admin = 1 WHERE email = 'your@email.com';"
   ```

#### Step 2: Create First Admin Token
```bash
# Bootstrap requires an existing admin user in database
ADMIN_TOKEN="bootstrap" go run cmd/admin/main.go create-token "Initial setup"
```

### Google App Engine Deployment (Datastore)

#### Step 1: Create First User and Admin
1. **Deploy the application** to Google App Engine
2. **Create your first user account** through the web interface
3. **Set the user as admin** using Google Cloud Console:
   ```bash
   # Using gcloud CLI to update Datastore entity
   gcloud datastore entities update --kind=User --key-id=USER_ID --set-property=is_admin:boolean=true
   ```
   
   Or use the Google Cloud Console Datastore interface to manually set `is_admin = true`

#### Step 2: Create First Admin Token
```bash
# Set environment for GAE deployment
export GOOGLE_CLOUD_PROJECT="your-project-id"
ADMIN_TOKEN="bootstrap" go run cmd/admin/main.go create-token "GAE Initial setup"
```

**Security Validation**: The system will verify an admin user exists in Datastore before allowing token creation.

This will output a secure token like:
```
Token: e2de0bb29b84ea94d2785c1ebfc65b450c767078f52e0e6df1d68d9e0fd13b08
```

### Step 3: Set Environment Variable
```bash
export ADMIN_TOKEN="e2de0bb29b84ea94d2785c1ebfc65b450c767078f52e0e6df1d68d9e0fd13b08"
```

### Step 4: Verify Setup
```bash
go run cmd/admin/main.go list-users
```

## Admin Commands

### Token Management
- `create-token <description>`: Generate new admin token (bootstrap or with valid token)
- `list-tokens`: Display all admin tokens with metadata
- `revoke-token <token-id>`: Deactivate an admin token

### User Management
- `list-users`: Display all users in the system
- `user-info <email>`: Show detailed user information
- `set-admin <email> <true/false>`: Grant or revoke admin privileges
- `grant-months <email> <months>`: Grant free subscription months

**Security**: All commands except `create-token` require valid database token authentication.

## Token Lifecycle Management

### Creating Additional Tokens
```bash
# With existing valid token
ADMIN_TOKEN="your_existing_token" go run cmd/admin/main.go create-token "CI/CD automation"
```

**Security Warning**: Creating additional tokens when others exist requires confirmation to prevent unauthorized token creation.

### Listing Active Tokens
```bash
ADMIN_TOKEN="your_token" go run cmd/admin/main.go list-tokens
```

### Revoking Compromised Tokens
```bash
# Revoke token by ID (from list-tokens output)
ADMIN_TOKEN="your_token" go run cmd/admin/main.go revoke-token 2
```

## Migration from Previous System

If you were using the old environment variable system:

1. **Remove old variables**:
   ```bash
   unset ADMIN_TOKEN_VERIFY
   unset SERVER_SECRET_KEY
   ```

2. **Bootstrap new system**:
   ```bash
   ADMIN_TOKEN="bootstrap" go run cmd/admin/main.go create-token "Migration to secure system"
   ```

3. **Set new secure token**:
   ```bash
   export ADMIN_TOKEN="new_generated_64_char_token_from_step_2"
   ```

## Security Best Practices

### Token Security
- Store tokens in secure password managers or encrypted configuration
- Never commit tokens to version control or logs
- Use descriptive names when creating tokens for audit purposes
- Regularly rotate tokens (create new, revoke old)
- Immediately revoke tokens if compromised or when team members leave

### Environment Management
- Use separate tokens per environment (dev, staging, production)
- Restrict server access to authorized personnel only
- Monitor token usage through `list-tokens` command
- Keep audit logs of admin operations

### Operational Security
- Use principle of least privilege (create tokens only when needed)
- Regularly review active tokens and revoke unused ones
- Set up alerting for admin command usage in production
- Document token ownership and purpose

## Error Messages Guide

### "ADMIN_TOKEN must be exactly 64 characters (32 bytes as hex)"
- **Cause**: Token format is invalid (not 64 hexadecimal characters)
- **Solution**: Use `create-token` command to generate valid token

### "Invalid ADMIN_TOKEN - token not found in database or inactive"
- **Cause**: Token doesn't exist in database or has been revoked
- **Solution**: Generate new token with `create-token` or contact administrator

### "ADMIN_TOKEN environment variable must be set"
- **Cause**: Missing required environment variable
- **Solution**: Set ADMIN_TOKEN with valid 64-character token

### Bootstrap Security Errors
- **"No admin users found in database"**: Initial token creation blocked
- **Solution**: Create user account and set as admin in database first

### Token Creation Security Warnings
- **"Admin tokens already exist"**: Shown when creating additional tokens
- **Solution**: Confirm with 'y' if you're authorized to create tokens

## Database Schema

The secure admin system adds the `admin_tokens` table:

```sql
CREATE TABLE admin_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash TEXT UNIQUE NOT NULL,    -- SHA-256 hash of token
    description TEXT NOT NULL,          -- Human-readable description  
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT 1         -- 0 = revoked, 1 = active
);
```

## Future Enhancements

Planned security improvements:
- Token expiration dates with automatic cleanup
- Admin action audit logging with IP addresses
- Multi-factor authentication for token creation
- Role-based permissions (read-only vs full admin)
- Integration with external identity providers

## Testing and Validation

The admin token security system includes comprehensive test coverage:

### Unit Tests (`internal/services/`)
- **SQLite Backend**: 6 test suites with 20+ individual test cases covering token generation, validation, listing, revocation, and lifecycle management
- **Datastore Backend**: 6 test suites for Google App Engine deployment compatibility (requires emulator)
- **Security Features**: Cryptographic token generation, database validation, bootstrap protection
- **Edge Cases**: Invalid formats, non-existent tokens, already-revoked tokens, concurrent operations

### Integration Tests (`test/integration/`)
- **Admin Command Testing**: Full CLI command validation with proper environment setup
- **Bootstrap Security**: Tests preventing unauthorized token creation without existing admin users
- **Token Lifecycle**: End-to-end testing of create → validate → list → revoke operations
- **Security Warnings**: Verification of prompts when creating additional tokens

### Running Admin Security Tests

```bash
# Run all admin token unit tests
go test ./internal/services -run "TestGenerateAdminToken|TestValidateAdminToken|TestListAdminTokens|TestRevokeAdminToken|TestHasAdminTokens|TestAdminTokenUniqueGeneration" -v

# Run Datastore tests (requires emulator)
DATASTORE_EMULATOR_HOST=localhost:8081 go test ./internal/services -run "TestDatastore" -v

# Run security integration tests  
go test ./test/integration -run "TestAdminSecurity" -v

# Run all tests with coverage
go test ./internal/services -cover
```

## Support

For security questions or issues:
1. Review this documentation thoroughly
2. **Run the security test suite** to validate token generation and validation
3. Check database connectivity and schema
4. Verify test coverage with `go test -cover ./internal/services`
5. Contact system administrator for enterprise deployment guidance