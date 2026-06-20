# Security Policy

## Supported Versions

Senda is currently in early release. Security fixes are applied to the latest version only.

| Version | Supported |
| ------- | --------- |
| latest  | Yes       |

## Reporting a Vulnerability

Report vulnerabilities privately through GitHub's
[private vulnerability reporting](https://github.com/this-senda/senda/security/advisories/new).
Please include:

- A description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fix (optional)

Please do not open public issues for security reports.

## Scope

Senda is a local desktop application. The primary attack surface is:

- Maliciously crafted collection YAML files
- Script sandbox escapes in the Goja pre/post-request scripting engine
- Path traversal in collection file operations

Out of scope: Senda does not run a network server, expose a web interface, or
handle authentication credentials beyond forwarding them in HTTP requests the
user explicitly configures.
