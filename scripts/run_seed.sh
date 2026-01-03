#!/bin/bash

# ========================================
# Run Test Tenant Seed Script
# ========================================
# This script executes the seed_test_tenant.sql
# in the PostgreSQL Docker container
#
# Usage:
#   ./scripts/run_seed.sh
# ========================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔧 Rekko - Running Test Tenant Seed${NC}"
echo ""

# Check if Docker container is running
if ! docker compose ps postgres | grep -q "Up"; then
    echo -e "${RED}❌ PostgreSQL container is not running${NC}"
    echo "   Start it with: docker compose up -d postgres"
    exit 1
fi

echo -e "${GREEN}✓ PostgreSQL container is running${NC}"
echo ""

# Execute seed script
echo "📝 Executing seed script..."
docker compose exec -T postgres psql -U rekko -d rekko_dev < scripts/seed_test_tenant.sql

echo ""
echo -e "${GREEN}✅ Test tenant seeded successfully!${NC}"
echo ""
echo "You can now test the API with:"
echo -e "${YELLOW}  API Key: test-api-key-rekko-dev${NC}"
echo ""
