# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |

## Reporting a Vulnerability

If you discover a security vulnerability in Planck, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please use [GitHub Security Advisories](https://github.com/sabizmil/planck/security/advisories/new) to report the vulnerability privately.

### What to expect

- Acknowledgment within 48 hours
- A fix timeline based on severity
- Credit in the release notes (unless you prefer anonymity)

## Scope

Planck is a terminal UI that orchestrates external CLI tools (like Claude Code). Security concerns include:

- Command injection via configuration or user input
- Unauthorized file access or modification
- Information disclosure through session data or logs
- Vulnerabilities in dependency chain

Issues in the external tools Planck orchestrates (e.g., Claude Code itself) should be reported to their respective maintainers.
