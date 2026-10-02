# Authentication hardening discovery

Discovery date: 21 September 2026. Base commit: `173c0a512445d101685ea29e1e62e67fd3ca1719`.
Branch: `codex/auth-portal-hardening`.

This is an upstream PR proposal covering the staff authentication portal. It contains no application changes. The proposed scope is TOTP and optional successful-login alerts, with the supporting controls needed to make them safe. Customer/widget authentication is a separate system.

Scope update: the current PR is **TOTP only**. The [TOTP implementation plan](totp-implementation-plan.md) supersedes the implementation sequence below. Login alerts and broader authentication hardening remain discovery notes for future work.

**The System account and alert addresses**

LibreDesk has a built-in agent whose login identifier is literally `System`, created with the Admin role. Its `email` value is not a deliverable address. Ordinary agents can also have the Admin role. See [account identity](../internal/user/models/models.go), [system account creation](../internal/user/user.go), and the explicit System exception in [login validation](../frontend/apps/main/src/views/auth/UserLoginView.vue).

The requested settings should therefore be independent of the System account's login identifier:

| Setting | Proposed behavior |
| --- | --- |
| Enable login alerts | Off by default for existing installations; administrator-controlled. |
| Alerts Receipt Address | One valid recipient mailbox, internal or external. Suggested UI wording: “Alert recipient address.” Required when alerts are enabled. |
| Alerts Sending Address | A distinct sender for security alerts. Suggested UI wording: “Alert sender address.” Default to the existing notification sender, with an optional override. |
| Send test alert | Explicit user action to check the configured recipient, sender, and SMTP connection; rate limited. |

Use the existing notification SMTP transport. A sender address is not another SMTP account: the configured relay must permit sending from it. Validate a single mailbox and reject header injection in both fields. Do not constrain the recipient to the installation's domain. Require SMTP to be configured/enabled before enabling alerts, and surface delivery failures to administrators.

The remaining product decision is which logins trigger the alert: System only, all administrator accounts including System, or every staff account. Recommendation: all administrator accounts including System. The addresses are independent of that choice; supporting every staff login can be a separate scope option if desired.

**What already exists**

| Capability | Evidence and implication |
| --- | --- |
| Password login | [cmd/login.go](../cmd/login.go) verifies a bcrypt password and immediately creates an authenticated session. TOTP needs an intermediate challenge before that session is created. |
| OIDC sign-in | [cmd/auth.go](../cmd/auth.go) handles provider callbacks. Password and OIDC login duplicate session creation, last-login updates, and audit recording. A shared login-completion function should own these actions and alert creation. |
| Sessions | [internal/auth/auth.go](../internal/auth/auth.go) uses Redis, a fresh random session ID on login, HttpOnly cookies, Secure cookies by default, SameSite=Lax, and a configurable lifetime defaulting to nine hours. Redis expiry is not extended on reads. |
| Throttling | [internal/ratelimit/ratelimit.go](../internal/ratelimit/ratelimit.go) and [cmd/init.go](../cmd/init.go) provide a shared auth limit of 30 requests/minute/IP by default. |
| Audit records | [internal/activity_log/activity_log.go](../internal/activity_log/activity_log.go) records successful login/logout with user and IP. No dedicated MFA lifecycle events were found. |
| Email | [notification service](../internal/notification/notification.go), [SMTP provider](../internal/notification/providers/email/email.go), templates, SMTP settings, and a [persistent email queue](../internal/notification/email_queue.go) already exist. |
| Encryption | [internal/crypto/crypto.go](../internal/crypto/crypto.go) supplies AES-GCM using the configured application key. A TOTP wrapper must require encrypted input; the general decrypt helper intentionally accepts plaintext for legacy callers. |
| Return redirects | The pinned fastglue `RedirectURI` helper strips schemes/hosts and normalizes leading slashes. An apparent unvalidated `next` parameter is not sufficient evidence of an open redirect here. Keep regression coverage. |

**Supporting fixes identified by source review**

These are source findings, not a claim that a deployed instance was penetration tested.

1. **Revoke browser sessions on security changes.** [Password reset](../cmd/users.go) invalidates an agent cache and kicks WebSockets; password updates also lack an account-wide session revocation mechanism. [Session validation](../internal/auth/auth.go) reads identity without a credential/security version. Add revocation on password reset/change, MFA replacement/removal/recovery, and emergency recovery. Ensure old sessions cannot reconnect through HTTP or WebSockets. A durable per-user security version is one option; check it across processes without relying on the ten-minute process-local agent cache. Define how pre-upgrade sessions are invalidated.
2. **Use trusted client IPs and account-based limits.** The pinned `fast-realip` dependency prioritizes request headers, including `X-Client-IP`, without a trusted-proxy check. Both the limiter and login audit use it. Header spoofing can evade the IP bucket when the app is directly reachable or a proxy preserves those headers. Introduce configured trusted proxies, use the socket peer otherwise, and share that resolver between limits and audit/alerts. Add independent account/IP/challenge limits and bounded cooldowns. The existing limiter returns success on Redis errors; MFA verification must not become unrestricted when its state or limits are unavailable.
3. **Tighten OIDC identity assurance.** The ID token is verified, but the callback selects an account by email without enforcing `EmailVerified`; the parsed subject is not used for account binding. Establish a verified-email/provider-trust policy, with a compatibility plan for providers that omit the claim. Longer term, bind identity to issuer/subject. State exists, but the application does not explicitly consume it, give it a short authentication-specific expiry, or bind it to the selected provider. Add those protections and assess PKCE/nonce support as a focused OIDC change.
4. **Protect security operations and their logs.** [Authentication middleware](../cmd/middlewares.go) accepts API credentials as well as sessions and logs CSRF cookie/header values on mismatch. Remove token values from logs. New MFA management and alert-settings routes should require an interactive session, CSRF protection, and recent authentication; an API key must not silently replace a second factor. The login endpoint itself is only rate-limit wrapped, and logout is a GET: review login-origin/CSRF protections and move logout to a protected POST with frontend updates.
5. **Harden password recovery as a related change.** Reset tokens are currently stored in plaintext, expire after one day, and are cleared on use ([queries](../internal/user/queries.sql)). Store token hashes, consider a shorter reset-specific lifetime separately from invitation links, and keep generic responses. Unknown login identifiers currently skip bcrypt, leaving a timing distinction despite generic error text. Password reset must never clear TOTP enrollment or bypass its challenge.

Items 1–2 and the protections for new endpoints belong with the MFA work. OIDC and existing recovery changes should be separately reviewable so their compatibility implications are visible.

**TOTP implementation requirements**

- Add an account Security page with enrollment, QR code/manual setup key, confirmation using a valid code, recovery-code download/copy, disable, and recovery-code regeneration. Include System and ordinary agents; do not tie availability to having a deliverable email address.
- Generate the QR locally. Use a maintained RFC 6238 implementation, standard authenticator-compatible parameters, an injected clock for tests, and a small documented clock-skew allowance. Never send the secret to a third-party QR service. Dependency selection remains implementation work.
- Store encrypted TOTP secrets in a dedicated security record, along with enrollment state and the last accepted time step. Keep this data out of generic user responses and logs. Provide key backup/rotation guidance; missing or incorrect keys must cause a controlled failure, never bypass MFA.
- After a valid password, issue an opaque, short-lived challenge (proposed lifetime: five minutes), bound to the browser and user. It must not contain an authenticated user session. Apply CSRF/origin protections to the challenge flow, limit attempts, and erase the password from frontend state.
- Complete authentication only after a valid TOTP or recovery code. Consume the challenge once, reject reused TOTP time steps atomically across concurrent requests, create a fresh authenticated session, and then record login and queue any enabled alert. Recheck account eligibility/security version before completion.
- Issue a small set of high-entropy, single-use recovery codes (proposed: ten). Show once, store hashes, consume atomically, and invalidate the previous set on regeneration. Require recent authentication and an existing factor/recovery code before replacing or disabling enrolled MFA. Enrollment itself requires fresh primary authentication.
- Keep password recovery separate from MFA recovery. Add an explicit, audited operator recovery command for System, following the existing password-reset CLI pattern. A normal password-reset command must not silently remove MFA.
- For an account enrolled in LibreDesk TOTP, the simplest safe initial rule is to challenge after both password and OIDC primary authentication. Otherwise an alternate login route bypasses enrollment. Trusting identity-provider MFA instead is a separate explicit policy requiring validated assurance claims; it can avoid double prompts but needs provider-specific design.
- Start with opt-in enrollment for upstream compatibility. “Require MFA for administrators/all staff” can follow, with a restricted enrollment session and recovery plan so enforcement cannot lock everyone out. Existing sessions must not bypass newly enabled MFA.
- Browser MFA does not automatically change API-key authentication. Document that boundary and separately review credential creation/revocation; new MFA administration endpoints must not accept API-key-only authorization.

Proposed data additions: a per-user MFA record, recovery-code hashes, a security version for revocation, and private alert settings. Short-lived challenges and attempt counters can use Redis. Update both fresh-install schema and versioned migrations. The exact table and route names are not settled by this discovery.

TOTP is useful protection against stolen passwords, but it is not phishing resistant. Recovery and factor replacement must be designed with the same care as login. See [OWASP's MFA guidance](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html) and [RFC 6238](https://datatracker.ietf.org/doc/html/rfc6238), which also requires rejecting an OTP after successful use. Passkeys are a reasonable later addition.

**Login-alert implementation requirements**

The current SMTP provider always uses one configured `From` address. `notifier.Message` and queued emails have no explicit sender override. Supporting the requested Alerts Sending Address therefore needs a typed sender field carried from the security event through persistent storage to SMTP delivery. Default other notification messages to the existing sender. Do not try to implement this with an arbitrary `From` header alone.

Store settings under a private security/notification namespace with permission-checked endpoints. Do not add recipient addresses to the public `app.*` settings response in [cmd/settings.go](../cmd/settings.go). Show the new fields in an admin Security or Security alerts panel and add translations and server-side validation.

Create one logical alert event for a completed login, after all required factors. Include account, timestamp/timezone, trusted source IP, a bounded browser/user-agent description, and authentication method. Do not include passwords, tokens, setup secrets, recovery codes, or raw session IDs. Escape untrusted values: the existing template renderer uses `text/template`, so HTML autoescaping must not be assumed. Use the configured canonical site URL for any links.

The immediate notification path is an in-memory channel, while delayed messages have persistent queueing and retries. Reuse/extend the persistent queue for security mail, with a unique login-event ID, sender/recipient captured for delivery, and no conversation “seen” suppression or coalescing. SMTP failure must not undo a successful login. Record queue/delivery failures and retain useful failure status after retry exhaustion. Deduplicate event creation; SMTP retries can still produce duplicate deliveries after ambiguous transport outcomes, so do not promise exactly-once email.

Audit enabling/disabling alerts and changing addresses. Sending a test email is a deliberate settings action. No emails were sent during discovery.

**Reviewable implementation sequence**

1. Session revocation, shared authentication completion, trusted client IP handling, account/challenge limits, and focused regression coverage.
2. Optional TOTP, enrollment/recovery UI, encrypted storage, recovery command, schema/migrations, and local/OIDC completion through the shared flow.
3. Optional login alerts with the two addresses, sender override, persistent delivery, settings UI, and tests.

These can be separate commits on this branch or small dependent PRs. Broader OIDC identity binding, passkeys, administrator-enforced MFA, idle timeouts, and a sessions/device-management screen should have explicit follow-up scope.

Acceptance coverage must include: no API/WebSocket access before MFA; expired/exhausted challenges; concurrent TOTP and recovery-code replay; disable/re-enroll races; password reset and old-session revocation; System enrollment/recovery; the selected OIDC policy; spoofed forwarding headers; absent/unavailable Redis; correct SMTP sender/recipient; alerts off; no alert after password-only success; persistent retries and deduplication; settings permissions; secret redaction; and fresh-install/upgrade behavior. A live SMTP service is unnecessary for CI: use a capture/fake sender.

**Discovery validation and limits**

The isolated branch was tested with:

```sh
go test -json ./internal/auth/... ./internal/authz/... ./internal/user/... \
  ./internal/ratelimit/... ./internal/crypto/... ./internal/notification/... ./cmd
```

The command succeeded: 412 passing test/subtest results and 16 skipped results. Database integration tests skipped because test Postgres was unavailable. `internal/auth`, `internal/ratelimit`, `internal/crypto`, and the SMTP provider currently have no tests of their own. This was a source/dependency review and targeted baseline run, not a browser exercise, migration test, live email delivery test, or full security audit.

The worktree is `/home/uron/projects/libredesk-auth-hardening`; it was isolated after another task changed the original checkout's branch. Other work was left untouched.
