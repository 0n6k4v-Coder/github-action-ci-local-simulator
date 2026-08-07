#!/bin/bash
# Audit current Docker waste without cleanup

set -e

echo "=== Docker Waste Audit ==="
echo "Date: $(date)"
echo ""

echo "--- Disk Usage Summary ---"
docker system df
echo ""

echo "--- Containers ($(docker ps -a -q | wc -l) total) ---"
echo "Running: $(docker ps -q | wc -l)"
echo "Stopped: $(docker ps -a -f 'status=exited' -q | wc -l)"
echo "Paused: $(docker ps -a -f 'status=paused' -q | wc -l)"
docker ps -a --format 'table {{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Size}}\t{{.Names}}' 2>/dev/null | head -20
echo ""

echo "--- Images ($(docker images -q | wc -l) total) ---"
echo "Dangling: $(docker images -f 'dangling=true' -q | wc -l)"
docker images --format 'table {{.Repository}}\t{{.Tag}}\t{{.Size}}' 2>/dev/null | head -20
echo ""

echo "--- Volumes ($(docker volume ls -q | wc -l) total) ---"
echo "Dangling: $(docker volume ls -f 'dangling=true' -q | wc -l)"
docker volume ls --format 'table {{.Driver}}\t{{.Name}}' 2>/dev/null
echo ""

echo "--- Build Cache ---"
docker builder du 2>/dev/null | tail -5 || echo "Build cache info not available"
echo ""

echo "--- Test Artifacts ---"
for dir in /tmp/parallel-test /tmp/pip-test /tmp/gacils-test /tmp/verify-fix; do
    if [ -d "$dir" ]; then
        size=$(du -sh "$dir" 2>/dev/null | cut -f1)
        echo "$dir: $size"
    fi
done

for dir in .gacils-cache .gacils-artifacts; do
    if [ -d "$dir" ]; then
        size=$(du -sh "$dir" 2>/dev/null | cut -f1)
        echo "$dir: $size"
    fi
done

if [ -f "gacils" ]; then
    echo "gacils binary: $(du -sh gacils | cut -f1)"
fi
echo ""

echo "=== Audit Complete ==="
