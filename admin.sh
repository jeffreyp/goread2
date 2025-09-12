#!/bin/bash

# GoRead2 Admin Management Script
# SECURITY: This script requires a valid 64-character ADMIN_TOKEN from the database
# Generate tokens with: ./admin.sh create-token "description"
# Usage: ./admin.sh <command> [args]

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# SECURITY CHECK: Require ADMIN_TOKEN
if [ -z "$ADMIN_TOKEN" ]; then
    echo -e "${RED}🔒 SECURITY ERROR: ADMIN_TOKEN environment variable is required${NC}"
    echo ""
    echo "This must be a valid 64-character admin token from the database."
    echo ""
    echo "To get started:"
    echo "  1. Create user account through web interface"
    echo "  2. Set as admin: sqlite3 goread2.db \"UPDATE users SET is_admin = 1 WHERE email = 'your@email.com'\""
    echo "  3. Generate token: ADMIN_TOKEN=\"bootstrap\" $0 create-token \"Initial setup\""
    echo "  4. Use the generated 64-character token"
    echo ""
    exit 1
fi

# Check if go is available
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
    exit 1
fi

# Function to show usage
show_usage() {
    echo -e "${YELLOW}GoRead2 Admin Management${NC}"
    echo ""
    echo "Usage: $0 <command> [args]"
    echo ""
    echo "Token Management:"
    echo "  create-token <description>    - Generate new admin token"
    echo "  list-tokens                   - List all admin tokens"
    echo "  revoke-token <id>             - Revoke admin token"
    echo ""
    echo "User Management:"
    echo "  list                          - List all users"
    echo "  admin <email> <on|off>        - Grant/revoke admin access"
    echo "  grant <email> <months>        - Grant free months"
    echo "  info <email>                  - Show user information"
    echo "  fix-sub <email>               - Fix subscription status from Stripe"
    echo "  set-sub-id <email> <sub-id>   - Update Stripe subscription ID"  
    echo ""
    echo "Examples:"
    echo "  $0 create-token \"Production server\""
    echo "  $0 list-tokens"
    echo "  $0 list"
    echo "  $0 admin your-email@gmail.com on"
    echo "  $0 grant user@example.com 6"
    echo "  $0 info user@example.com"
    echo ""
}

# Check arguments
if [ $# -lt 1 ]; then
    show_usage
    exit 1
fi

COMMAND=$1

case $COMMAND in
    "create-token")
        if [ $# -ne 2 ]; then
            echo -e "${RED}Error: Usage: $0 create-token <description>${NC}"
            exit 1
        fi
        
        DESCRIPTION=$2
        echo -e "${YELLOW}🔑 Creating new admin token: $DESCRIPTION${NC}"
        go run cmd/admin/main.go create-token "$DESCRIPTION"
        ;;
    
    "list-tokens")
        echo -e "${YELLOW}🔑 Listing all admin tokens...${NC}"
        go run cmd/admin/main.go list-tokens
        ;;
    
    "revoke-token")
        if [ $# -ne 2 ]; then
            echo -e "${RED}Error: Usage: $0 revoke-token <token-id>${NC}"
            exit 1
        fi
        
        TOKEN_ID=$2
        if ! [[ "$TOKEN_ID" =~ ^[0-9]+$ ]]; then
            echo -e "${RED}Error: Token ID must be a number${NC}"
            exit 1
        fi
        
        echo -e "${YELLOW}🚫 Revoking admin token ID: $TOKEN_ID${NC}"
        go run cmd/admin/main.go revoke-token "$TOKEN_ID"
        ;;

    "list")
        echo -e "${YELLOW}📋 Listing all users...${NC}"
        go run cmd/admin/main.go list-users
        ;;
    
    "admin")
        if [ $# -ne 3 ]; then
            echo -e "${RED}Error: Usage: $0 admin <email> <on|off>${NC}"
            exit 1
        fi
        
        EMAIL=$2
        STATUS=$3
        
        case $STATUS in
            "on"|"true"|"yes")
                BOOL_STATUS="true"
                echo -e "${YELLOW}👑 Granting admin access to $EMAIL...${NC}"
                ;;
            "off"|"false"|"no")
                BOOL_STATUS="false"
                echo -e "${YELLOW}👤 Revoking admin access from $EMAIL...${NC}"
                ;;
            *)
                echo -e "${RED}Error: Status must be 'on' or 'off'${NC}"
                exit 1
                ;;
        esac
        
        go run cmd/admin/main.go set-admin "$EMAIL" "$BOOL_STATUS"
        ;;
    
    "grant")
        if [ $# -ne 3 ]; then
            echo -e "${RED}Error: Usage: $0 grant <email> <months>${NC}"
            exit 1
        fi
        
        EMAIL=$2
        MONTHS=$3
        
        # Validate months is a number
        if ! [[ "$MONTHS" =~ ^[0-9]+$ ]]; then
            echo -e "${RED}Error: Months must be a positive number${NC}"
            exit 1
        fi
        
        echo -e "${YELLOW}🎁 Granting $MONTHS free months to $EMAIL...${NC}"
        go run cmd/admin/main.go grant-months "$EMAIL" "$MONTHS"
        ;;
    
    "info")
        if [ $# -ne 2 ]; then
            echo -e "${RED}Error: Usage: $0 info <email>${NC}"
            exit 1
        fi
        
        EMAIL=$2
        echo -e "${YELLOW}ℹ️  Getting user information for $EMAIL...${NC}"
        go run cmd/admin/main.go user-info "$EMAIL"
        ;;
    
    "fix-sub")
        if [ $# -ne 2 ]; then
            echo -e "${RED}Error: Usage: $0 fix-sub <email>${NC}"
            exit 1
        fi
        
        EMAIL=$2
        echo -e "${YELLOW}🔧 Fixing subscription status for $EMAIL...${NC}"
        go run cmd/admin/main.go fix-subscription "$EMAIL"
        ;;
    
    "set-sub-id")
        if [ $# -ne 3 ]; then
            echo -e "${RED}Error: Usage: $0 set-sub-id <email> <subscription-id>${NC}"
            exit 1
        fi
        
        EMAIL=$2
        SUB_ID=$3
        echo -e "${YELLOW}🔄 Updating subscription ID for $EMAIL...${NC}"
        go run cmd/admin/main.go set-subscription-id "$EMAIL" "$SUB_ID"
        ;;
    
    *)
        echo -e "${RED}Error: Unknown command '$COMMAND'${NC}"
        show_usage
        exit 1
        ;;
esac