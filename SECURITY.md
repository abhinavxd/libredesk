# Security Reports

Report vulnerabilities privately via GitHub Security Advisories: https://github.com/abhinavxd/libredesk/security/advisories

## Threat model

Libredesk is **self-hosted and single-tenant**. Agents and admins are trusted internal staff. The only untrusted surfaces are the **livechat widget** (anonymous contacts) and **inbound email**.
The permission system is a policy layer over already-trusted users, not a boundary between mutually distrusting parties. Admin permissions (`*:manage`, `*:read_all`) grant full control over their scope and are not granted by default.

## Out of scope

- An admin exercising a documented admin capability (configuring webhooks, OIDC providers, automations, templates, inboxes, etc.). It is up to the operator to grant these capabilities only to trusted users. Defects in the admin code paths (sqli, RCE, auth bypass etc.) remain in scope.
- SSRF via admin-configured outbound URLs (webhooks, OIDC provider discovery, etc.).

Anything else is in scope.
