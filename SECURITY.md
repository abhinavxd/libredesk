# Security Reports

Report vulnerabilities privately via GitHub Security Advisories: https://github.com/abhinavxd/libredesk/security/advisories

## Threat model

Libredesk is **self-hosted and single-tenant**. Agents and admins are trusted internal staff. The only untrusted surfaces are the **livechat widget** (anonymous contacts) and **inbound email**.
The permission system is a policy layer over already-trusted users, not a boundary between mutually distrusting parties. Admin permissions (`*:manage`, `*:read_all`) grant full control over their scope and are not granted by default.

## Out of scope

- An admin exercising a documented admin capability (configuring webhooks, OIDC providers, automations, templates, inboxes, etc.). It is up to the operator to grant these capabilities only to trusted users. Defects in the admin code paths (sqli, RCE, auth bypass etc.) remain in scope.
- SSRF via admin-configured outbound URLs (webhooks, OIDC provider discovery, AI provider and custom tool calls, etc.). In single-tenant self-hosted deployments the admin already controls the host, so this is not a privilege boundary. Deployments where these URLs come from untrusted parties (e.g. multi-tenant/hosted) can enable the `[ssrf]` guard in the config to block requests to private/reserved IP ranges.

Anything else is in scope.
