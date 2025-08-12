#!/bin/bash

# WilliamBoard Development Server Setup Script (Simplified)
# This script sets up basic infrastructure without PostGIS for quick testing

set -e  # Exit on any error

echo "🏗️  WilliamBoard Development Setup (Simplified)"
echo "================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check prerequisites
print_status "Checking prerequisites..."

# Check Go
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.21+ from https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1)
print_success "Go found: $GO_VERSION"

# Check PostgreSQL
if ! command -v psql &> /dev/null; then
    print_error "PostgreSQL is not installed. Please install PostgreSQL 14+"
    echo "  macOS: brew install postgresql@14"
    echo "  Ubuntu: sudo apt install postgresql-14 postgresql-contrib"
    exit 1
fi

if ! command -v createdb &> /dev/null; then
    print_error "PostgreSQL client tools not found. Please install postgresql-client"
    exit 1
fi

print_success "PostgreSQL found"

# Check if PostgreSQL is running
if ! pg_isready -q; then
    print_warning "PostgreSQL is not running. Starting PostgreSQL..."
    if command -v brew &> /dev/null; then
        brew services start postgresql@14 || brew services start postgresql
    else
        sudo service postgresql start
    fi
    
    # Wait a moment for PostgreSQL to start
    sleep 2
    
    if ! pg_isready -q; then
        print_error "Failed to start PostgreSQL. Please start it manually:"
        echo "  macOS: brew services start postgresql"
        echo "  Ubuntu: sudo service postgresql start"
        exit 1
    fi
fi

print_success "PostgreSQL is running"

# Setup environment file
print_status "Setting up environment configuration..."

if [ ! -f .env ]; then
    print_status "Creating .env from template..."
    cp .env.example .env
    
    # Update .env for local development
    sed -i.bak 's|ENVIRONMENT=production|ENVIRONMENT=development|' .env
    sed -i.bak 's|PUBLIC_BASE_URL=https://your-app-name.onrender.com|PUBLIC_BASE_URL=http://localhost:8080|' .env
    sed -i.bak 's|UPLOAD_DIR=/data/uploads|UPLOAD_DIR=./uploads|' .env
    sed -i.bak 's|DATABASE_URL=postgres://user:pass@localhost:5432/williamboard?sslmode=disable|DATABASE_URL=postgres://localhost:5432/williamboard?sslmode=disable|' .env
    
    # Remove backup file
    rm -f .env.bak
    
    print_success "Created .env file for local development"
else
    print_success ".env file already exists"
fi

# Check for OpenAI API key
if ! grep -q "OPENAI_API_KEY=sk-" .env 2>/dev/null; then
    print_warning "OpenAI API key not found in .env file"
    echo "  Stage 1 (basic infrastructure) will work without it"
    echo "  For Stage 2 (GPT-4o vision), set OPENAI_API_KEY in .env"
    echo "  Get your key at: https://platform.openai.com/api-keys"
else
    print_success "OpenAI API key configured"
fi

# Setup database
print_status "Setting up database..."

# Check if database exists
if psql -lqt | cut -d \| -f 1 | grep -qw williamboard; then
    print_success "Database 'williamboard' already exists"
else
    print_status "Creating database 'williamboard'..."
    createdb williamboard
    print_success "Database 'williamboard' created"
fi

# Install essential extensions only (skip PostGIS for now)
print_status "Installing essential PostgreSQL extensions..."
psql williamboard -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";" > /dev/null 2>&1 || {
    print_warning "uuid-ossp extension not available (will use random UUIDs)"
}
print_success "Essential extensions installed"

print_warning "Skipping PostGIS for quick setup"
echo "  PostGIS is needed for advanced location features (Stage 3+)"
echo "  For now, location fields will be stored as text"
echo "  To install PostGIS later: brew install postgis && brew services restart postgresql"

# Create uploads directory
print_status "Setting up file storage..."
mkdir -p uploads
print_success "Uploads directory created"

# Install Go dependencies
print_status "Installing Go dependencies..."
go mod tidy
print_success "Go dependencies installed"

# Build the application
print_status "Building application..."
go build -o bin/api ./api/main.go
print_success "Application built successfully"

# Final setup summary
echo ""
echo "🎉 Development setup complete! (Simplified mode)"
echo "==============================================="
echo ""
echo "📋 Summary:"
echo "  • PostgreSQL database: williamboard (running)"
echo "  • Environment: development (.env configured)"
echo "  • File storage: ./uploads/ directory"
echo "  • Application: built as ./bin/api"
echo "  • PostGIS: skipped (text-based locations for now)"
echo ""

# Check OpenAI status
if grep -q "OPENAI_API_KEY=sk-" .env 2>/dev/null; then
    echo "🧠 GPT-4o Vision: ✅ API key configured (Stage 2 ready)"
else
    echo "🧠 GPT-4o Vision: ⚠️  No API key (Stage 1 only)"
    echo "   Add OPENAI_API_KEY=sk-your-key-here to .env for full functionality"
fi

echo ""
echo "🚀 Ready to start development server:"
echo "   ./start-dev.sh"
echo ""
echo "🧪 Test endpoints:"
echo "   Health check: curl http://localhost:8080/health"
echo "   Upload test:  curl -X POST http://localhost:8080/v1/uploads/signed-url -H 'Content-Type: application/json' -d '{\"contentType\": \"image/jpeg\"}'"
echo ""
echo "⚡ Quick test: ./test-api.sh"
echo "📚 Full docs: See README.md"
echo ""
echo "💡 Note: This simplified setup skips PostGIS. Location data will be stored as text."
echo "    Full GIS features can be added later with: brew install postgis"