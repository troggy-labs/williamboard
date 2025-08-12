#!/bin/bash

# WilliamBoard Development Server Starter
# Run this after ./dev.sh to start the development server

set -e

echo "🚀 Starting WilliamBoard Development Server"
echo "==========================================="

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Check if setup has been run
if [ ! -f "bin/api" ]; then
    echo "❌ Application not built. Please run './dev.sh' first to set up the development environment."
    exit 1
fi

if [ ! -f ".env" ]; then
    echo "❌ Environment file missing. Please run './dev.sh' first to set up the development environment."
    exit 1
fi

# Check PostgreSQL
if ! pg_isready -q; then
    print_warning "PostgreSQL is not running. Starting PostgreSQL..."
    if command -v brew &> /dev/null; then
        brew services start postgresql@14 || brew services start postgresql
    else
        sudo service postgresql start
    fi
    sleep 2
fi

# Load environment and show configuration
print_status "Loading environment configuration..."
source .env

echo "📋 Server Configuration:"
echo "  • App: $APP_NAME"
echo "  • Port: $PORT"
echo "  • Environment: $ENVIRONMENT"
echo "  • Upload Directory: $UPLOAD_DIR"

# Check OpenAI configuration
if [[ "$OPENAI_API_KEY" == "your-openai-api-key-here" ]] || [[ -z "$OPENAI_API_KEY" ]]; then
    print_warning "Stage 1 mode: No OpenAI API key configured"
    echo "  • File uploads and storage will work"
    echo "  • GPT-4o vision analysis disabled (returns 0 events)"
    echo "  • Set OPENAI_API_KEY in .env for Stage 2 functionality"
else
    print_success "Stage 2 mode: GPT-4o vision analysis enabled"
    echo "  • Full event detection and extraction active"
fi

echo ""
print_status "Starting API server..."

# Start the server with environment loaded
exec ./bin/api