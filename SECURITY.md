# Security Policy

## Supported Versions

gacils releases are tagged as `vMAJOR.MINOR.PATCH` (e.g. `v1.4.2`).
Security fixes are applied to the **latest released minor series** only.

| Version | Status      | Security Support |
| ------- | ----------- | ---------------- |
| v1.4.x  | Current     | ✅ Fixes issued  |
| < v1.4  | End of life | ❌ No support    |

There is **no long-term support (LTS)** window. Users should upgrade to
the latest release promptly. When a security fix is issued, it will be
released as a new patch version on top of the current minor series.

To check your version:

```bash
gacils version
```

## Reporting a Vulnerability

**Use the GitHub "Report a vulnerability" button:**

1. Visit the repository's **Security** tab → **Advisories** →
   **"Report a vulnerability"**.
2. Fill in the form with details of the issue.
3. You will receive an automated acknowledgment, followed by a
   maintainer review.

> If the "Report a vulnerability" option is not visible, it may not yet
> be enabled for this repository. In that case, please **open a
> Discussion** in the GitHub repository marked as private (security
> related) or reach out to the maintainer through the contact information
> on their public profile, clearly stating that the message concerns a
> security vulnerability.

Please include the following in your report:

- A clear description of the vulnerability.
- Steps to reproduce (PoC).
- The potential impact.
- Any suggested fix (if known).

You should receive an initial response within **48 hours**. If you do
not, please follow up.

## Disclosure Policy

The project follows responsible disclosure:

```text
Private report received
        ↓
   Triage by maintainer
        ↓
   Fix developed (private)
        ↓
  Security release published
        ↓
    Public disclosure
```

The fix will be coordinated with the reporter before public disclosure.
The project will not publicly disclose details of a vulnerability until
a fix and advisory are available. Credit will be given to the reporter
unless they request anonymity.

## Security Scope

This project's relevant security surface:

- **Command execution**: gacils runs GitHub Actions steps inside Docker
  containers. It reads and executes workflow files from the local
  filesystem. Malicious workflow definitions can lead to code execution.
- **Container isolation**: Container images and service execution rely on
  the local Docker daemon. Misconfiguration can weaken isolation.
- **Secrets handling**: gacils masks secrets in logs (per the README).
  Any bypass of the masking mechanism is a security-relevant defect.
- **Dependency vulnerabilities**: The Go module dependencies declared in
  `go.mod` are scanned by CodeQL and monitored by Dependabot.
- **Expression evaluation**: The workflow expression engine evaluates
  user-supplied workflow expressions; injection or sandbox-escape issues
  are in-scope.

Out of scope (not considered security vulnerabilities):

- Vulnerabilities that require you to already run a malicious workflow
  on your own machine (this is expected behavior — gacils is a local
  simulator of trusted workflows).
- Denial-of-service in workflow parsing that does not affect the host
  outside of the local simulation process.

## Security Controls

This repository is configured with the following DevSecOps controls.
See `docs/DEVSECOPS.md` for the full documentation:

| Control              | Configuration                                   |
| -------------------- | ----------------------------------------------- |
| CI pipeline          | `.github/workflows/ci.yml`                      |
| SAST (CodeQL)        | `.github/workflows/codeql.yml`                  |
| Dependency updates   | `.github/dependabot.yml`                        |
| Secret scanning      | See "Repository-level Settings" below           |
| Branch protection    | See "Repository-level Settings" below           |
