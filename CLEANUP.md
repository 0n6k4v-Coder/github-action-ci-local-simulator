# Docker Cleanup Guide

This project uses Docker extensively for testing. This guide helps you manage Docker resources and prevent disk space waste.

## Quick Reference

```bash
make audit          # See current waste
make clean-docker   # Light cleanup (safe)
make clean-all      # Full cleanup (between phases)
make clean-all PRUNE_IMAGES=1 # Full cleanup + remove ALL images
```

## What Creates Waste?

### 1. Zombie Containers

**Cause:** Old bug (WaitContainer instead of StopContainer)
**Status:** Fixed in v1.4.0
**Cleanup:** `make clean-docker`

### 2. Unused Images

**Cause:** Pulling multiple Go versions for testing (1.22, 1.23, 1.26)
**Default cleanup:** Keeps images from last 24h (safe)
**Prune images mode:** `make clean-all PRUNE_IMAGES=1` removes all unused images

### 3. Build Cache

**Cause:** Docker caches build layers for faster rebuilds
**Cleanup:** `docker builder prune -a -f`

### 4. Volumes

**Cause:** Test volumes (gacils-tmp, gacils-cache, etc.)
**Cleanup:** `docker volume prune -f`

### 5. Test Artifacts

**Cause:** Integration tests create temp directories
**Cleanup:** `make clean-docker`

## Cleanup Strategies

### After Each Test Run (Recommended)

```bash
make clean-docker
```

- Removes stopped containers
- Removes dangling images
- Cleans test artifacts
- Safe to run frequently

### Between Development Phases

```bash
make clean-all
```

- Removes ALL unused Docker resources
- **Default:** Preserves images pulled within last 24h (safety)
- Removes test artifacts
- Frees maximum disk space
- Requires confirmation

### Prune Images Mode (Truly Clean)

```bash
make clean-all PRUNE_IMAGES=1
```

- Removes ALL unused Docker resources **including ALL images**
- No time filter — removes everything unused
- Use when you want a completely clean slate
- Will need to re-pull images on next run

### When Low on Disk Space

```bash
# Aggressive cleanup
make clean-all

# Also clean Go module cache (optional)
go clean -modcache
go clean -cache
```

## Monitoring

### Check Current State

```bash
make audit
```

Shows:
- Docker disk usage
- Container count and status
- Image count and size
- Volume count
- Test artifacts

### Manual Inspection

```bash
# Containers
docker ps -a

# Images
docker images

# Volumes
docker volume ls

# Build cache
docker builder du
```

## Best Practices

1. **Run `make clean-docker` after each test session**
   - Prevents waste accumulation
   - Safe and fast

2. **Run `make clean-all` between phases**
   - Frees maximum space
   - Start fresh for next phase

3. **Don't worry about Go module cache**
   - Re-downloading is fast
   - Only clean if desperate for space

4. **Monitor with `make audit`**
   - Run before cleanup to see what you're removing
   - Run after cleanup to verify

## Troubleshooting

### "Cannot connect to Docker daemon"

```bash
# Check if Docker is running
sudo systemctl status docker

# Start Docker
sudo systemctl start docker
```

### "Permission denied"

```bash
# Add user to docker group
sudo usermod -aG docker $USER
newgrp docker
```

### Cleanup script fails

```bash
# Run commands manually
docker ps -a -q | xargs -r docker rm -f
docker image prune -a -f --filter "until=24h"  # safe mode
docker image prune -a -f                        # prune images mode
docker volume prune -f
docker builder prune -a -f
```

## Files

- `scripts/audit-waste.sh` - Audit current waste
- `scripts/cleanup-light.sh` - Light cleanup
- `scripts/cleanup-full.sh` - Full cleanup
- `Makefile` - Build targets

## History

- v1.4.0 (2026-08-07): Initial cleanup infrastructure
- v1.4.1: Added `--prune-images` flag for complete image removal
