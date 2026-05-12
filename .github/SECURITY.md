# Security Policy

## Supported Versions

Security fixes are accepted for the currently maintained development branch,
`develop`, and any actively deployed release branch.

Historical branches, archived experiments, and local development snapshots are
not supported unless a maintainer explicitly marks them as supported.

## Reporting A Vulnerability

Please report security vulnerabilities privately. Do not create a public issue,
pull request, discussion, or social media post that includes exploit details,
proof-of-concept code, secrets, private keys, token abuse paths, or personally
identifiable information.

Preferred reporting path:

1. Open the repository's Security tab on GitHub.
2. Choose "Report a vulnerability" if private vulnerability reporting is
   enabled.
3. Include a concise description, affected surface, reproduction steps, impact,
   and any suggested mitigation.

If private vulnerability reporting is not available, open a minimal public issue
asking for a private security contact. Do not include vulnerability details in
that issue.

## Scope

Security reports are especially useful for issues involving:

- authentication or authorization bypass
- Twitch extension or OAuth token exposure
- wallet signature verification, replay protection, or chain ownership checks
- TACHI token accounting, mint, burn, claim, spend, balance, reward, raffle, or
  airdrop logic
- smart contract vulnerabilities
- leaked secrets, signer keys, deployment credentials, or production
  configuration
- dependency vulnerabilities that are reachable in production behavior

Routine dependency update requests, general bugs without a security impact, and
feature requests should use the normal issue or pull request process instead.

## Response Expectations

Maintainers will try to acknowledge valid private reports within three business
days. The timeline for a fix depends on severity, exploitability, affected
surface, and release coordination needs.

When a report is accepted, maintainers may coordinate a fix privately before
public disclosure. Please avoid publishing details until a fix or mitigation has
been released and maintainers have had a reasonable opportunity to notify users.

## Bounty Policy

This project does not currently operate a paid bug bounty program. Reports are
still welcome and appreciated, but submitting a report does not create an
entitlement to payment, rewards, tokens, or other compensation.
