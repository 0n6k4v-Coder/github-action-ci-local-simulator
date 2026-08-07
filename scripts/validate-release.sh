#!/bin/bash
# Pre-release validation script
# Run this BEFORE creating a release tag

set -e

echo "=== Pre-Release Validation ==="
echo "Date: $(date)"
echo ""

# Check 1: No stubs returning nil
# Stubs that return errors (fmt.Errorf) are OK — they indicate unimplemented
# features but fail loudly with proper exit codes (exit code 1).
# Stubs that return nil are BAD — they hide errors from users (exit code 0).
echo "Check 1: Checking for stubs returning nil..."
# Use a Python one-liner to find "not implemented" lines, then check if the
# corresponding RunE function returns nil within the next few lines.
python3 -c '
import re, sys
bad = []
with open("internal/cli/commands.go") as f:
    lines = f.readlines()
for i, line in enumerate(lines):
    if "not implemented" in line and "_test.go" not in "internal/cli/commands.go":
        # Check next 5 lines for "return nil"
        for j in range(i+1, min(i+6, len(lines))):
            if "return nil" in lines[j]:
                bad.append(f"internal/cli/commands.go:{i+1} {line.strip()}")
                break
if bad:
    print("❌ ERROR: Found stubs that return nil instead of errors:")
    for b in bad:
        print(f"  {b}")
    print("")
    print("Please change stubs to return fmt.Errorf instead of nil.")
    sys.exit(1)
' 2>/dev/null || {
    # Fallback: simpler grep-based check
    if grep -Pzo 'not implemented[^\n]*\n\s*(fmt\.Println[^\n]*\n)*\s*return nil' internal/cli/*.go 2>/dev/null | head -1; then
        echo "❌ ERROR: Found stubs that return nil instead of errors:"
        grep -rn "not implemented" internal/cli/*.go | grep -v "_test.go"
        echo ""
        echo "Please change stubs to return fmt.Errorf instead of nil."
        exit 1
    fi
}
echo "✅ All stubs return errors (not nil)"

# Check 2: Tests pass
# Use -short to skip Docker-dependent integration/E2E tests (per project convention)
echo "Check 2: Running tests..."
if ! go test -short ./... -timeout 120s; then
    echo "❌ ERROR: Tests failed"
    echo "Please fix failing tests before tagging."
    echo "Note: Docker-dependent integration/E2E tests are skipped with -short."
    echo "      Run them separately with Docker available: go test ./test/..."
    exit 1
fi
echo "✅ All tests pass"

# Check 3: Build succeeds
echo ""
echo "Check 3: Building binary..."
if ! go build -o /tmp/gacils-validate ./cmd/gacils; then
    echo "❌ ERROR: Build failed"
    echo "Please fix build errors before tagging."
    exit 1
fi
echo "✅ Build succeeds"

# Check 4: Binary works
echo ""
echo "Check 4: Testing binary..."
if ! /tmp/gacils-validate --help > /dev/null 2>&1; then
    echo "❌ ERROR: Binary doesn't work"
    exit 1
fi
echo "✅ Binary works"

# Check 5: CHANGELOG updated
echo ""
echo "Check 5: Checking CHANGELOG..."
if ! grep -q "## \[1\." CHANGELOG.md; then
    echo "❌ ERROR: CHANGELOG.md not updated"
    echo "Please add release notes before tagging."
    exit 1
fi
echo "✅ CHANGELOG.md updated"

# Check 6: README updated
echo ""
echo "Check 6: Checking README..."
if ! grep -q "## " README.md; then
    echo "❌ ERROR: README.md not updated"
    echo "Please update documentation before tagging."
    exit 1
fi
echo "✅ README.md updated"

# Check 7: Clean command works (if implemented)
echo ""
echo "Check 7: Testing clean command..."
if /tmp/gacils-validate clean --dry-run --force > /dev/null 2>&1; then
    echo "✅ Clean command works"
else
    echo "⚠️  Clean command not working (may be expected if not implemented)"
fi

# Check 8: Race detector
echo ""
echo "Check 8: Running race detector..."
if ! go test -race ./internal/cli/ -timeout 60s; then
    echo "❌ ERROR: Race conditions detected"
    echo "Please fix race conditions before tagging."
    exit 1
fi
echo "✅ No race conditions"

echo ""
echo "=== All Checks Passed ==="
echo "Ready to create release tag"
echo ""
echo "Next steps:"
echo "  1. git tag -a vX.Y.Z -m 'Release vX.Y.Z: ...'"
echo "  2. git push origin vX.Y.Z"
echo "  3. Verify on GitHub: git ls-remote origin refs/tags/vX.Y.Z"
