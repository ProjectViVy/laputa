#!/bin/bash
# run_integration_tests.sh - Setup and run integration tests for mempalace-go
# This script prepares the environment and runs the integration test suite

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$SCRIPT_DIR"
MEMPALACE_HOME="${MEMPALACE_HOME:-$HOME/.mempalace}"
TEST_HOME="/tmp/mempalace-integration-test"

echo "=== mempalace-go Integration Test Runner ==="
echo ""

# Step 1: Setup test home directory
echo "Step 1: Setting up test environment at $TEST_HOME..."
rm -rf "$TEST_HOME"
mkdir -p "$TEST_HOME/palace"
mkdir -p "$TEST_HOME/models"
echo "  Created $TEST_HOME"
echo ""

# Step 2: Create a config for the test environment
echo "Step 2: Creating test configuration..."
cat > "$TEST_HOME/config.json" << EOF
{
    "palace_path": "/tmp/mempalace-integration-test/palace",
    "models_dir": "/tmp/mempalace-integration-test/models",
    "model_name": "sentence-transformers/all-MiniLM-L6-v2"
}
EOF
echo "  Created $TEST_HOME/config.json"
echo ""

# Step 3: Run the integration tests
echo "Step 3: Running integration tests..."
echo "  Test home: $TEST_HOME"
echo ""

cd "$PROJECT_DIR"

export MEMPALACE_HOME="$TEST_HOME"

# Also create the palace dir at the hardcoded ~/.mempalace location that server looks for
mkdir -p "$HOME/.mempalace/palace"

# Run tests with the test home
# Use -p 1 to run packages sequentially (avoids port conflicts between tests)
go test -p 1 -v ./...

echo ""
echo "=== Test run complete ==="
