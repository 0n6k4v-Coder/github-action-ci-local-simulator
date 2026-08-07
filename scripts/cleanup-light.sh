#!/bin/bash
# Light cleanup after test runs - safe for frequent use

set -e

echo "=== Light Cleanup ==="
echo ""

# Stop running containers (safe - they can be restarted)
running=$(docker ps -q | wc -l)
if [ "$running" -gt 0 ]; then
    echo "Stopping $running running container(s)..."
    docker ps -q | xargs -r docker stop -t 2
else
    echo "No running containers"
fi

# Remove stopped containers (safe - they're already done)
stopped=$(docker ps -a -f 'status=exited' -q | wc -l)
if [ "$stopped" -gt 0 ]; then
    echo "Removing $stopped stopped container(s)..."
    docker ps -a -f 'status=exited' -q | xargs -r docker rm -f
else
    echo "No stopped containers to remove"
fi

# Remove dangling images only (safe - they're unused)
dangling=$(docker images -f 'dangling=true' -q | wc -l)
if [ "$dangling" -gt 0 ]; then
    echo "Removing $dangling dangling image(s)..."
    docker image prune -f
else
    echo "No dangling images"
fi

# Clean test artifacts (safe - these are temporary)
for dir in .gacils-cache .gacils-artifacts; do
    if [ -d "$dir" ]; then
        echo "Removing $dir..."
        rm -rf "$dir"
    fi
done

# Remove gacils binary (safe - can be rebuilt)
if [ -f "gacils" ]; then
    echo "Removing gacils binary..."
    rm -f gacils
fi

echo ""
echo "=== Light Cleanup Complete ==="
