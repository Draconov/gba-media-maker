# Security policy

## Supported version

Security fixes target the latest published release.

## Reporting a vulnerability

Please avoid filing a public issue for vulnerabilities involving arbitrary command execution, path traversal, local API authentication, downloaded-binary verification, or unintended network exposure. Use GitHub's private vulnerability-reporting feature when enabled for the repository.

Include reproduction steps, affected version, operating system, and impact. Do not include private videos or personal paths unless necessary.

## Local service design

The app binds to loopback only, uses a random per-session URL token, limits upload size, and verifies the pinned FFmpeg archive against the SHA-256 checksum list published with the pinned binary release. Changes that weaken any of those properties require an explicit security review.
