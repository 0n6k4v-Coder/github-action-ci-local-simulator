#!/bin/bash
# Full cleanup between development phases - more aggressive

set -e

# Parse flags
PRUNE_IMAGES=false
for arg in "$@"; do
    case $arg in
        --prune-images)
            PRUNE_IMAGES=true
            shift
            ;;
    esac
done

echo "=== Full Cleanup ==="
echo "Mode: $(if $PRUNE_IMAGES; then echo 'PRUNE_IMAGES (removes ALL images)'; else echo 'SAFE (preserves images from last 24h)'; fi)"
echo ""

# Ask for confirmation
if $PRUNE_IMAGES; then
    read -p "This will remove ALL unused Docker resources INCLUDING ALL images. Continue? [y/N] " -n 1 -r
else
    read -p "This will remove ALL unused Docker resources (preserving recent images). Continue? [y/N] " -n 1 -r
fi
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

echo "Step 1/6: Stopping all containers..."
docker ps -q | xargs -r docker stop -t 2 2>/dev/null || true

echo "Step 2/6: Removing all containers..."
docker ps -a -q | xargs -r docker rm -f 2>/dev/null || true

echo "Step 3/6: Removing unused images..."
if $PRUNE_IMAGES; then
    echo "  [PRUNE_IMAGES] Removing ALL unused images (no time filter)..."
    docker image prune -a -f 2>/dev/null || true
else
    echo "  [SAFE] Removing unused images from >24h ago..."
    docker image prune -a -f --filter "until=24h" 2>/dev/null || true
fi

echo "Step 4/6: Removing unused volumes..."
docker volume prune -f 2>/dev/null || true

echo "Step 5/6: Cleaning build cache..."
docker builder prune -a -f 2>/dev/null || true

echo "Step 6/6: Removing test artifacts..."
# Project directories
for dir in .gacils-cache .gacils-artifacts; do
    if [ -d "$dir" ]; then
        echo "  Removing $dir..."
        rm -rf "$dir"
    fi
done

# Binary
if [ -f "gacils" ]; then
    echo "  Removing gacils binary..."
    rm -f gacils
fi

# Temp directories
for dir in /tmp/parallel-test /tmp/pip-test /tmp/gacils-test /tmp/verify-fix; do
    if [ -d "$dir" ]; then
        echo "  Removing $dir..."
        rm -rf "$dir"
    fi
done

echo ""
echo "=== Full Cleanup Complete ==="
echo ""
echo "Run 'make audit' to see current state"
