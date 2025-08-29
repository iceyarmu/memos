#!/bin/bash

set -e

echo "=== Building Memos Docker Image ==="

# Check if pnpm is installed
if ! command -v pnpm &> /dev/null; then
    echo "Error: pnpm is not installed. Please install pnpm first."
    echo "You can install it with: npm install -g pnpm"
    exit 1
fi

# Build frontend
echo "=== Building Frontend ==="
cd web/
echo "Installing frontend dependencies..."
pnpm install
echo "Building frontend for production..."
pnpm release
cd ..

# Build Docker image
echo "=== Building Docker Image ==="

# Check if --platform flag is provided for multi-arch build
if [ "$1" == "--multi-arch" ]; then
    echo "Building multi-architecture image (requires Docker Buildx)..."
    
    # Check if buildx is available
    if ! docker buildx version &> /dev/null; then
        echo "Error: Docker Buildx is not available. Please install Docker Buildx for multi-arch builds."
        exit 1
    fi
    
    # Create buildx builder if it doesn't exist
    if ! docker buildx ls | grep -q "memos-builder"; then
        echo "Creating buildx builder..."
        docker buildx create --name memos-builder --use
        docker buildx inspect --bootstrap
    else
        docker buildx use memos-builder
    fi
    
    # Build multi-arch image (linux/amd64 and linux/arm64)
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        -f scripts/Dockerfile \
        -t memos:canary \
        --load \
        .
else
    # Build single architecture image
    echo "Building single architecture image..."
    docker build -f scripts/Dockerfile -t memos:canary .
fi

echo "=== Build Complete ==="
echo "Docker image 'memos:canary' has been built successfully!"
echo ""
echo "Usage:"
echo "  Run the container: docker run -d -p 5230:5230 -v ~/.memos/:/var/opt/memos memos:canary"
echo "  Build multi-arch:  ./build.sh --multi-arch"
