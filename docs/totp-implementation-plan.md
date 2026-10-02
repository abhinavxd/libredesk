# TOTP implementation plan

Branch: `codex/auth-portal-hardening`. Worktree: `/home/uron/projects/libredesk-auth-hardening`.
Status: proposed implementation plan and interactive frontend mock; no production authentication changes yet.

**PR outcome and boundary**

An agent can enable authenticator-app two-factor authentication, save recovery codes, sign in with their password or SSO plus a second factor, and safely manage that factor. This includes the built-in System account. Existing installations and unenrolled accounts retain their current login behavior. Enrollment is opt-in; no global enforcement policy is added.

The same account-level TOTP requirement applies after local password and OIDC authentication. SSO is not itself proof that a second factor was used. Trusting an identity provider's MFA claims is outside this PR.

The PR includes only protections necessary for this feature: short-lived challenges, attempt limits, replay protection, recent authentication for factor management, invalidation of sessions when MFA security state changes, and MFA audit events. It does not include login-alert email/settings, a global session-management screen, passkeys, remembered devices, forced enrollment, a general password-reset rewrite, proxy configuration changes, or a broader OIDC identity/linking rewrite. API-key authentication remains separate; API keys cannot administer MFA through the new endpoints.

**Backend changes**

| Work | Files / area | Required behavior |
| --- | --- | --- |
| MFA storage | New `internal/mfa/` service, models, SQL; `schema.sql`; new registered version migration in `internal/migrations/` and `cmd/upgrade.go` | Per-user encrypted secret, enabled timestamp, last accepted TOTP step, persistent MFA version; separate hashed recovery codes. Never expose credentials through normal user serialization. |
| Cryptographic operations | `internal/mfa/`, `internal/crypto/`, `go.mod` / `go.sum` | Use a vetted RFC 6238 library, cryptographic randomness, six digits, 30-second steps, and a documented small skew window. Encrypt secrets using existing AES-GCM, with strict encrypted-only reads. Generate ten high-entropy recovery codes and store only hashes. Dependency selection/license review is an implementation step. |
| Pending challenges | New MFA challenge store using Redis | Five-minute expiry, opaque browser-bound token in a separate HttpOnly/Secure cookie, purpose/user/version binding, no authenticated user ID in the normal session. Server-side attempt and account limits apply to both TOTP and recovery codes. |
| Complete login | `cmd/login.go`, `cmd/auth.go`, `internal/auth/auth.go` | Check enrollment after primary authentication. For enrolled accounts, issue a challenge; create the full session, update last login, and write the existing login activity only after the second factor succeeds. Extract a small shared completion helper for the two login paths. |
| Verification | New `cmd/mfa.go` handlers and service methods | Verify TOTP and consume the accepted step atomically; consume each recovery code once in a database transaction. Allow only one request to complete a challenge. Recheck user eligibility and MFA version before granting access. |
| Account management | `cmd/mfa.go`, `cmd/handlers.go`, session-only authorization helper | Status, fresh authentication, enrollment start/confirm, recovery-code regeneration, and disable. Require session authentication and CSRF; reject API-key-only requests. Bind every action to the signed-in account. |
| MFA session invalidation | `internal/auth/`, `cmd/middlewares.go`, agent WebSocket authorization/hub | Store the MFA version in new sessions and check the authoritative version. Enabling, disabling, operator reset, and recovery-code regeneration increment it. Rotate the current session after successful management; reject older browser sessions and terminate stale sockets across instances. Do not rely on the process-local ten-minute user cache. |
| Recovery and audit | CLI setup in `cmd/`, `internal/mfa/`, `internal/activity_log/`, activity enum migration | Provide an explicit operator-only MFA reset command for a selected existing agent, including System. Clear factor/codes/pending state, bump MFA version, and record the action without secrets. Password-reset commands must not silently clear MFA. |

Keep an MFA-version record even after disabling the factor. Otherwise deleting the record could make old sessions valid again. Missing versions in legacy sessions count as zero: they remain compatible for accounts that have never changed MFA state, while enabling MFA invalidates them. Authoritative checks must fail closed on storage errors. Reuse any existing cross-instance socket signaling; if none is available, add narrowly scoped MFA invalidation signaling and verify inbound socket actions against the version.

Enrollment secrets are pending until the user submits a valid authenticator code. Keep pending setup encrypted with a short expiry and bind it to a freshly authenticated session. Cancel/restart invalidates the previous pending setup. The confirmation transaction enables the factor, records its accepted step, creates recovery-code hashes, and bumps the MFA version. Return the plaintext recovery codes once with `Cache-Control: no-store`. If that response is lost, the user can authenticate with their app and regenerate the codes.

Recovery-code login consumes one code but keeps TOTP enabled. Preserve a short-lived, session-bound proof of that successful second-factor verification so a user who just used their last recovery code can still disable/replace the lost authenticator after confirming their primary identity. Bind that proof to one management action and consume it; otherwise require a new TOTP/recovery code. There is no email-based MFA reset in this PR. A person who loses all factors and has no fresh verification proof needs the documented operator reset path.

Recent authentication lasts at most five minutes and is server-side, purpose-bound proof. Local accounts can confirm their password. SSO-only accounts must be able to return through their configured provider for explicit fresh authentication; request and validate fresh authentication (`max_age` / `auth_time`) rather than treating a silent SSO round trip as reauthentication. The returned identity must be the same account and provider associated with the flow. Providers unable to supply the required assurance must show a clear error, not silently waive the check. Existing enabled MFA also requires TOTP, a recovery code, or the fresh second-factor proof described above for disable/regeneration. Never require an unknown local password from an SSO-only user.

Proposed initial verification limits: five failed attempts per challenge and ten per account in ten minutes, shared across new challenges and both verification methods. Use atomic Redis operations and a bounded cooldown, with clear retry behavior. Keep the current IP limiter as an additional layer; MFA safety must not depend on the existing forwarded-IP behavior. Redis failure or lost challenge state requires restarting sign-in, not unrestricted verification. Exact thresholds can be tuned without changing the API.

Do not log request bodies, authenticator secrets, OTPs, recovery codes, cookies, or reauthentication proofs. Audit enrollment, disable, recovery-code use/regeneration, and operator reset using the existing activity-log mechanism. Store enough actor/target context to review changes without introducing a new monitoring subsystem.

**Proposed API contract**

Route names below are implementation targets, not existing endpoints. All responses use the existing envelope format. Sensitive responses use `Cache-Control: no-store`.

| Endpoint | Authorization / result |
| --- | --- |
| Existing `POST /api/v1/auth/login` | Unenrolled success keeps the current user response. Enrolled success of the password step returns `data: { mfa_required: true }` and a pending challenge cookie, never a full user session. |
| Existing OIDC callback | Unenrolled users continue normally. Enrolled users get the pending challenge and redirect to `/login/two-factor`; preserve the safe post-login destination server-side. |
| `GET /api/v1/auth/mfa/challenge` | Bound pending cookie; returns validity, expiry, and permitted methods, without credentials or unrelated user data. |
| `POST /api/v1/auth/mfa/verify` | Pending cookie plus CSRF/origin protection; body `{ method: "totp" | "recovery_code", code }`. Success returns the user and safe destination after rotating into a full session. |
| `POST /api/v1/auth/mfa/cancel` | Bound challenge and CSRF/origin protection; consumes the pending challenge and returns to sign-in. |
| `GET /api/v1/agents/me/mfa` | Full browser session; returns enabled state, enabled timestamp, remaining recovery-code count, and available reauthentication methods. No secret or code hashes. |
| `POST /api/v1/agents/me/mfa/reauthenticate` | Full session plus CSRF; confirms password or begins the bound OIDC reauthentication flow. Issues a short-lived management proof. |
| `POST /api/v1/agents/me/mfa/enrollment` | Recent authentication; creates pending enrollment and returns the setup URI/manual key only to that user. |
| `POST /api/v1/agents/me/mfa/enrollment/confirm` | Recent authentication, pending enrollment, and valid TOTP; enables MFA and returns recovery codes once. |
| `POST /api/v1/agents/me/mfa/recovery-codes` | Recent authentication plus current TOTP/recovery code; atomically replaces the code set and returns the new codes once. |
| `DELETE /api/v1/agents/me/mfa` | Recent authentication plus current TOTP/recovery code; disables MFA, clears secrets/codes, invalidates pending challenges and old sessions. |

Enrollment and management proofs must be single-purpose and rejected after security-version changes. Use a narrowly scoped CSRF/session middleware for these endpoints rather than changing the behavior of unrelated APIs. Account-wide limits prevent restarting enrollment or reauthentication to obtain unlimited attempts. Challenge consumption, MFA version checks, and session issuance need explicit race tests: a concurrent reset/disable must not leave an accepted session with stale authority.

**Frontend changes**

| Work | Files / area | User experience |
| --- | --- | --- |
| Account navigation | `frontend/apps/main/src/router/index.js`, `layouts/account/AccountLayout.vue`, new `views/account/security/SecurityView.vue` | Add Account → Security. Show two-factor status and an Enable button; once enabled show recovery-code count, regenerate, and disable actions. |
| Setup | New small components under `features/account/security/` | Confirm identity → scan QR or copy manual key → verify a six-digit code → save recovery codes. Require acknowledgement before leaving the recovery-code screen. |
| Login challenge | `views/auth/UserLoginView.vue`, new `views/auth/TwoFactorView.vue`, router | Check `mfa_required` before updating the user store or navigating into the app. Show authenticator entry, “Use a recovery code,” and “Back to sign in.” OIDC lands on the same challenge view. |
| Recovery and management | Security page components | Recovery-code entry during login; explicit confirmation before disable/regeneration; fresh-authentication UI that supports password or SSO. Explain that regeneration invalidates old codes. |
| API and translations | `frontend/apps/main/src/api/index.js`, `i18n/en-US.json` and translation workflow | Typed-by-convention API helpers and translated labels/errors. Keep credentials, setup keys, and recovery codes out of persistent Pinia/localStorage state. |
| Tests | Vitest component/behavior tests; focused Cypress auth flow | Exercise partial authentication, enrollment, recovery, management, expired/invalid codes, loading states, reload/back navigation, and accessibility. |

Use existing Vue/Radix UI components and authentication layout. Prefer one properly labeled numeric input with `inputmode="numeric"`, `autocomplete="one-time-code"`, and six-character validation. It should support paste, leading zeros, keyboard submission, and screen readers. Recovery codes use a normal text field and documented separator normalization. Avoid six separate focus targets. Generate QR codes locally from the returned setup URI; no remote QR service. Do not expose sample/mock credentials in production code.

Errors distinguish retryable invalid code, expired challenge (restart login), exhausted attempts (retry after cooldown), and service unavailability. Never fall through into a signed-in state on a malformed or failed response. A reload of the challenge page uses its bound cookie, not a token in a URL or browser storage. Navigating back cancels the challenge and clears transient credentials.

The mock demonstrates the Security page, setup and recovery-code save screens, enabled state, disable/regenerate confirmation, authenticator/recovery login, and invalid/expired challenge states. It is sample data and local interaction only; it makes no authentication requests. For the initial implementation, match the existing app typography and neutral controls rather than redesigning the portal.

**Suggested commits and acceptance gates**

1. Backend storage/cryptography and migrations, with RFC test vectors and atomic recovery/TOTP replay tests.
2. Backend challenge/login/session integration, management, operator recovery, and audit tests.
3. Frontend Security/enrollment/management screens and translations.
4. Frontend login challenge integration and end-to-end regressions; operator/user documentation.

The API contract above is the boundary between backend and frontend work. Both are required before merging this single TOTP PR; backend support alone must not present MFA as ready for users. Feature availability remains off until enrollment.

Required checks before the eventual PR:

- Existing local and OIDC login behavior is unchanged for unenrolled accounts.
- Password/SSO success alone gives enrolled accounts no protected HTTP or WebSocket access and creates no successful-login record.
- A valid code completes login exactly once; wrong/expired/replayed codes fail, including concurrent requests on different instances.
- TOTP boundary/skew tests use a fake clock and RFC vectors; codes with leading zeros work.
- Recovery codes are hashed, single-use under concurrency, and obsolete after regeneration or disable. Recovery-code login keeps MFA enabled. A user who just consumed the last code can recover their factor using the bounded verification proof; expired or replayed proofs cannot authorize management.
- Setup secrets never appear in generic user responses/logs; encryption-key failures do not bypass the factor.
- Existing sessions are rejected after MFA state changes, including connected/reconnecting sockets and multi-instance caches. Missing legacy versions cannot bypass enrollment.
- Password resets and admin password changes do not remove enrollment. Operator reset is explicit and audited.
- Both password and SSO users can safely enroll/manage MFA; mismatched or insufficient OIDC reauthentication is rejected.
- API credentials cannot invoke MFA management; CSRF, cross-account requests, replayed management proofs, Redis outages, challenge restart abuse, and reset/verify races are covered.
- Fresh installation and upgrade migrations pass against Postgres; Go tests, frontend tests/build, and a focused browser flow pass.

The previous discovery baseline passed its runnable tests, but 16 database tests skipped without Postgres. Those skipped checks are not sufficient acceptance for this feature. This planning change introduces no runnable authentication implementation and does not claim the gates above have passed.
