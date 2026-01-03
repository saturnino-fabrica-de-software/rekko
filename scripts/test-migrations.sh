#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Testing Database Migrations ===${NC}"

# Check if PostgreSQL is running
if ! docker compose ps postgres | grep -q "Up"; then
    echo -e "${RED}PostgreSQL is not running. Starting...${NC}"
    docker compose up -d postgres
    sleep 3
fi

# Database connection details
export PGHOST=localhost
export PGPORT=5432
export PGUSER=rekko
export PGPASSWORD=rekko_dev_pass
export PGDATABASE=rekko_dev

echo -e "${GREEN}✓ PostgreSQL is running${NC}"

# Test 1: Check if database exists
echo -e "\n${YELLOW}Test 1: Checking database connection...${NC}"
if psql -c "SELECT version();" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Database connection successful${NC}"
else
    echo -e "${RED}✗ Database connection failed${NC}"
    exit 1
fi

# Test 2: Check pgcrypto extension
echo -e "\n${YELLOW}Test 2: Checking pgcrypto extension...${NC}"
EXTENSION_CHECK=$(psql -t -c "SELECT COUNT(*) FROM pg_extension WHERE extname = 'pgcrypto';")
if [ "$EXTENSION_CHECK" -eq "1" ]; then
    echo -e "${GREEN}✓ pgcrypto extension installed${NC}"
else
    echo -e "${YELLOW}! pgcrypto not installed (will be installed by migration)${NC}"
fi

# Test 3: Check migration tables
echo -e "\n${YELLOW}Test 3: Checking schema...${NC}"

# Check tenants table
if psql -c "\d tenants" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ tenants table exists${NC}"

    # Validate columns
    COLUMNS=$(psql -t -c "SELECT column_name FROM information_schema.columns WHERE table_name = 'tenants' ORDER BY ordinal_position;")
    echo -e "  Columns: ${COLUMNS}"
else
    echo -e "${YELLOW}! tenants table does not exist (will be created by migration)${NC}"
fi

# Check api_keys table
if psql -c "\d api_keys" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ api_keys table exists${NC}"

    # Validate columns
    COLUMNS=$(psql -t -c "SELECT column_name FROM information_schema.columns WHERE table_name = 'api_keys' ORDER BY ordinal_position;")
    echo -e "  Columns: ${COLUMNS}"
else
    echo -e "${YELLOW}! api_keys table does not exist (will be created by migration)${NC}"
fi

# Test 4: Check indexes
echo -e "\n${YELLOW}Test 4: Checking indexes...${NC}"
INDEXES=$(psql -t -c "SELECT indexname FROM pg_indexes WHERE schemaname = 'public' ORDER BY indexname;")
if [ -n "$INDEXES" ]; then
    echo -e "${GREEN}✓ Indexes found:${NC}"
    echo "$INDEXES"
else
    echo -e "${YELLOW}! No indexes found (will be created by migration)${NC}"
fi

# Test 5: Check constraints
echo -e "\n${YELLOW}Test 5: Checking constraints...${NC}"
CONSTRAINTS=$(psql -t -c "SELECT conname, contype FROM pg_constraint WHERE conrelid::regclass::text IN ('tenants', 'api_keys') ORDER BY conname;")
if [ -n "$CONSTRAINTS" ]; then
    echo -e "${GREEN}✓ Constraints found:${NC}"
    echo "$CONSTRAINTS"
else
    echo -e "${YELLOW}! No constraints found (will be created by migration)${NC}"
fi

# Test 6: Insert test data
echo -e "\n${YELLOW}Test 6: Testing data insertion...${NC}"

# Insert tenant
TENANT_ID=$(psql -t -c "
INSERT INTO tenants (name, slug, plan, settings)
VALUES ('Test Tenant', 'test-tenant', 'pro', '{\"max_faces\": 1000}')
RETURNING id;
" 2>/dev/null | xargs)

if [ -n "$TENANT_ID" ]; then
    echo -e "${GREEN}✓ Tenant inserted: $TENANT_ID${NC}"

    # Insert API key
    API_KEY_ID=$(psql -t -c "
    INSERT INTO api_keys (tenant_id, name, key_hash, key_prefix, environment)
    VALUES ('$TENANT_ID', 'Test Key', 'hash123', 'rekko_test_abc1', 'test')
    RETURNING id;
    " 2>/dev/null | xargs)

    if [ -n "$API_KEY_ID" ]; then
        echo -e "${GREEN}✓ API Key inserted: $API_KEY_ID${NC}"

        # Cleanup test data
        psql -c "DELETE FROM api_keys WHERE id = '$API_KEY_ID';" > /dev/null 2>&1
        psql -c "DELETE FROM tenants WHERE id = '$TENANT_ID';" > /dev/null 2>&1
        echo -e "${GREEN}✓ Test data cleaned up${NC}"
    else
        echo -e "${RED}✗ Failed to insert API key${NC}"
    fi
else
    echo -e "${YELLOW}! Could not test data insertion (tables may not exist yet)${NC}"
fi

echo -e "\n${GREEN}=== Migration Tests Complete ===${NC}"
