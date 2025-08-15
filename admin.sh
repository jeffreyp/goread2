#!/bin/bash

# GoRead2 Admin Management Script
# Usage: ./admin.sh <command> [args]

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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
    echo "Commands:"
    echo "  list                          - List all users"
    echo "  admin <email> <on|off>        - Grant/revoke admin access"
    echo "  grant <email> <months>        - Grant free months"
    echo "  info <email>                  - Show user information"
    echo ""
    echo "Examples:"
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
    
    *)
        echo -e "${RED}Error: Unknown command '$COMMAND'${NC}"
        show_usage
        exit 1
        ;;
esac