#!/bin/bash
# Test that a release tag gives the expected features
# This verifies the release from a user's perspective
# Usage: ./scripts/test-install-from-tag.sh <tag>

set -e

TAG=$1
if [ -z "$TAG" ]; then
    echo "Usage: $0 <tag>"
    echo "Example: $0 v1.4.1"
    exit 1
fi

# Save the repo root directory
REPO_DIR=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$REPO_DIR"

echo "=== Testing Install from Tag ==="
echo "Tag: $TAG"
echo "Date: $(date)"
echo ""

ERRORS=0

# Check if tag exists
if ! git rev-parse --verify "$TAG" > /dev/null 2>&1; then
    echo "❌ ERROR: Tag $TAG does not exist"
    exit 1
fi

echo "Step 1: Checking clean command exists in tag..."
# Check if clean.go exists in the tag (implemented vs stub)
if git show ${TAG}:internal/cli/clean.go > /dev/null 2>&1; then
    echo "✅ clean.go exists in tag ${TAG}"
else
    echo "❌ clean.go missing in tag ${TAG} (clean command was not implemented)"
    ERRORS=$((ERRORS + 1))
fi

echo ""
echo "Step 2: Checking stub commands in tag..."
# Get commands.go from the tag and check for stubs returning nil
STUB_OUTPUT=$(git show ${TAG}:internal/cli/commands.go 2>/dev/null || echo "")

if [ -n "$STUB_OUTPUT" ]; then
    # Check for stubs that return nil (bad)
    STUBS_NIL=$(python3 -c '
import sys
lines = sys.stdin.readlines()
bad = []
for i, line in enumerate(lines):
    if "not implemented" in line:
        for j in range(i+1, min(i+6, len(lines))):
            if "return nil" in lines[j]:
                bad.append(f"line {i+1}: {line.strip()}")
                break
if bad:
    print("\n".join(bad))
' <<< "$STUB_OUTPUT" || true)

    if [ -n "$STUBS_NIL" ]; then
        echo "❌ Found stubs that return nil instead of errors in tag ${TAG}:"
        echo "$STUBS_NIL"
        echo ""
        echo "  These stubs give users exit code 0 instead of a proper error."
        ERRORS=$((ERRORS + 1))
    else
        echo "✅ All stubs return errors (not nil) in tag ${TAG}"
    fi
else
    echo "⚠️  Could not check commands.go in tag ${TAG}"
fi

echo ""
echo "Step 3: Checking clean command is implemented in tag..."
# Verify clean command is not a stub in commands.go
CLEAN_STUB=$(echo "$STUB_OUTPUT" | grep -B2 -A3 'clean.*not implemented' || true)
if [ -n "$CLEAN_STUB" ]; then
    echo "❌ clean command is still a stub in tag ${TAG}:"
    echo "$CLEAN_STUB"
    ERRORS=$((ERRORS + 1))
else
    # Double-check: clean command should either be implemented (clean.go exists) or have a non-nil error
    if git show ${TAG}:internal/cli/clean.go > /dev/null 2>&1; then
        echo "✅ clean command is implemented (clean.go exists) in tag ${TAG}"
    else
        echo "❌ clean command is stub (no clean.go) in tag ${TAG}"
        ERRORS=$((ERRORS + 1))
    fi
fi

echo ""
echo "Step 4: Checking README mentions clean command..."
README_OUTPUT=$(git show ${TAG}:README.md 2>/dev/null || echo "")
if [ -n "$README_OUTPUT" ]; then
    if echo "$README_OUTPUT" | grep -q "gacils clean"; then
        echo "✅ README.md mentions 'gacils clean' in tag ${TAG}"
    else
        echo "❌ README.md does not mention 'gacils clean' in tag ${TAG}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "⚠️  Could not check README.md in tag ${TAG}"
fi

echo ""
echo "Step 5: Checking CHANGELOG for tag version..."
VERSION=${TAG#v}
CHANGELOG_OUTPUT=$(git show ${TAG}:CHANGELOG.md 2>/dev/null || echo "")
if [ -n "$CHANGELOG_OUTPUT" ]; then
    if echo "$CHANGELOG_OUTPUT" | grep -q "## \[${VERSION}"; then
        echo "✅ CHANGELOG.md has section for ${VERSION} in tag ${TAG}"
    else
        echo "❌ CHANGELOG.md missing section '## [${VERSION}]' in tag ${TAG}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "⚠️  Could not check CHANGELOG.md in tag ${TAG}"
fi

# Summary
echo ""
echo "=== Install Test Summary ==="
if [ $ERRORS -eq 0 ]; then
    echo "✅ All checks passed for tag ${TAG}"
    echo ""
    echo "Users installing ${TAG} will get all claimed features."
    exit 0
else
    echo "❌ ${ERRORS} check(s) failed for tag ${TAG}"
    echo ""
    echo "Users installing ${TAG} will encounter issues:"
    echo "  - clean command may be a stub"
    echo "  - Some stub commands may return nil (exit code 0)"
    exit 1
fi
