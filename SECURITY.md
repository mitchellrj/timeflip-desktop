# Security Policy

## Supported Versions

TimeFlip Desktop is pre-release software. Security fixes are provided on the active development branch and the latest published release, when releases are available.

Older commits, experimental branches, and locally modified builds are not supported.

## Reporting a Vulnerability

Please do not report security vulnerabilities through public GitHub issues.

Use GitHub private vulnerability reporting for this repository:

https://github.com/mitchellrj/timeflip-desktop/security/advisories/new

If private vulnerability reporting is unavailable, contact the maintainer privately through GitHub before publishing details.

Include as much detail as you can safely share:

- Affected version, commit, or build.
- Operating system and architecture.
- Steps to reproduce.
- Expected and observed impact.
- Whether local app data, device passwords, BLE traces, or generated artifacts are involved.

## Response Expectations

Security reports will be reviewed as soon as practical. The maintainer may ask for additional reproduction details, propose mitigations, or coordinate a fix before public disclosure.

If the report is accepted, the fix will normally be developed privately or with limited details until a patch is available. Public release notes may describe the issue at a high level without exposing exploit details.

## Sensitive Local Data

TimeFlip Desktop is local-first, but some local files can contain sensitive information:

- The SQLite database under the user application support/config directory.
- BLE trace logs, especially traces containing authorization, password, or raw device payloads.
- Crash logs and debug logs that include device identifiers.

Do not attach these files to public issues. Redact passwords, identifiers, and raw trace payloads before sharing diagnostics.
