#!/bin/bash

# Database management script for Rekko
# Usage: ./scripts/db.sh [command]

set -e

# Load environment variables
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Default values
DB_HOST=${POSTGRES_HOST:-localhost}
DB_PORT=${POSTGRES_PORT:-5433}
DB_USER=${POSTGRES_USER:-rekko}
DB_PASSWORD=${POSTGRES_PASSWORD:-rekko}
DB_NAME=${POSTGRES_DB:-rekko_dev}
DATABASE_URL=${DATABASE_URL:-postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable}

MIGRATIONS_PATH="./migrations"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Check if migrate is installed
check_migrate() {
    if ! command -v migrate &> /dev/null; then
        print_error "golang-migrate is not installed"
        echo "Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        exit 1
    fi
}

# Commands
case "$1" in
    up)
        check_migrate
        print_info "Running migrations up..."
        migrate -path $MIGRATIONS_PATH -database "$DATABASE_URL" up
        print_success "Migrations applied successfully"
        ;;

    down)
        check_migrate
        print_info "Rolling back last migration..."
        migrate -path $MIGRATIONS_PATH -database "$DATABASE_URL" down 1
        print_success "Migration rolled back successfully"
        ;;

    reset)
        check_migrate
        print_info "Resetting database..."
        migrate -path $MIGRATIONS_PATH -database "$DATABASE_URL" drop -f
        migrate -path $MIGRATIONS_PATH -database "$DATABASE_URL" up
        print_success "Database reset successfully"
        ;;

    version)
        check_migrate
        migrate -path $MIGRATIONS_PATH -database "$DATABASE_URL" version
        ;;

    force)
        check_migrate
        if [ -z "$2" ]; then
            print_error "Usage: $0 force <version>"
            exit 1
        fi
        migrate -path $MIGRATIONS_PATH -database "$DATABASE_URL" force "$2"
        print_success "Version forced to $2"
        ;;

    status)
        print_info "Database status:"
        echo "  Host: $DB_HOST:$DB_PORT"
        echo "  Database: $DB_NAME"
        echo "  User: $DB_USER"

        # Check if docker container is running
        if docker ps --format '{{.Names}}' | grep -q "rekko-postgres"; then
            print_success "Container rekko-postgres is running"
        else
            print_error "Container rekko-postgres is not running"
            echo "Start with: docker compose up -d"
        fi

        # Check if database is accessible
        if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1" &> /dev/null; then
            print_success "Database is accessible"
        else
            print_error "Cannot connect to database"
        fi
        ;;

    psql)
        print_info "Connecting to database..."
        PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME
        ;;

    dump)
        OUTPUT_FILE=${2:-"./backups/rekko_$(date +%Y%m%d_%H%M%S).sql"}
        mkdir -p "$(dirname "$OUTPUT_FILE")"
        print_info "Creating database dump..."
        PGPASSWORD=$DB_PASSWORD pg_dump -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME > "$OUTPUT_FILE"
        print_success "Database dumped to: $OUTPUT_FILE"
        ;;

    restore)
        if [ -z "$2" ]; then
            print_error "Usage: $0 restore <dump_file>"
            exit 1
        fi
        print_info "Restoring database from $2..."
        PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME < "$2"
        print_success "Database restored successfully"
        ;;

    seed)
        print_info "Seeding database with test data..."
        PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << 'EOF'
-- Create test tenant
INSERT INTO tenants (name, api_key_hash, settings, is_active)
VALUES (
    'Test Tenant',
    'test_hash_' || md5(random()::text),
    '{"plan": "pro", "max_faces": 1000}',
    true
)
ON CONFLICT DO NOTHING;

SELECT 'Seed completed. Tenant count: ' || count(*) FROM tenants;
EOF
        print_success "Database seeded successfully"
        ;;

    *)
        echo "Rekko Database Management"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  up        - Run all pending migrations"
        echo "  down      - Rollback last migration"
        echo "  reset     - Drop and recreate database"
        echo "  version   - Show current migration version"
        echo "  force     - Force migration version"
        echo "  status    - Show database status"
        echo "  psql      - Connect to database with psql"
        echo "  dump      - Create database backup"
        echo "  restore   - Restore database from backup"
        echo "  seed      - Seed database with test data"
        echo ""
        echo "Environment:"
        echo "  DATABASE_URL=$DATABASE_URL"
        ;;
esac
