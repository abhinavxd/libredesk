# Mobile Device Tokens Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the four backend changes the Libredesk Flutter mobile app needs to authenticate: per-device tokens, bearer middleware, an OIDC one-time code exchange, and token-authenticated websocket upgrades.

**Architecture:** A new `user_device_tokens` table holds many long-lived tokens per agent, separate from the existing single-pair API key. Lookup is one indexed hit on a public `selector` followed by a constant-time compare of a hashed `verifier`, so there is no bcrypt cost per request. The auth middleware gains a `Bearer` branch beside the existing Basic api-key branch. OIDC gains a PKCE-bound one-time code so a native app can complete SSO without access to the browser cookie jar.

**Tech Stack:** Go, fastglue, PostgreSQL (goyesql), Redis, Casbin.

## Why this exists

The mobile app (separate repo, `~/projects/libredesk-mobile`) is already written and tested against these endpoints. It is blocked on this PR: nothing past the server-address screen works without it. The app's design doc is `docs/superpowers/specs/2026-07-31-libredesk-mobile-design.md` in that repo.

## Global Constraints

Copied from the repo's standing conventions. Every task's requirements implicitly include this section.

- **Go file layout, strict order, every file:** (1) consts, then vars; (2) type aliases / named non-struct types; (3) structs; (4) the `New`/init constructor; (5) all methods of the struct; (6) public package-level functions; (7) private functions last. Never declare a regex, const, or var inline next to the function that uses it - hoist it to the top. Private helpers go last, not first.
- **Comments:** only a non-obvious WHY, one line maximum. No comment that restates the next line. No rationale/design narration on functions, structs or SQL. Phrases like "so that", "because", "in order to", "ensures that" in a code comment are a smell - delete the comment.
- **No em dashes** anywhere: code, comments, commit messages, PR text.
- **Commit messages** lead with a plain imperative verb. No `feat:`/`fix:`/`chore:` prefixes.
- SQL lives in the package's `queries.sql` as goyesql `-- name: x` blocks, never inline in Go.
- API responses use fastglue envelopes: `{"data": ...}` on success, `{"error": {"type","message"}}` on failure. Error types are in `internal/envelope/envelope.go`.
- Deployment is **single-instance, single-tenant, one process**. Do not add distributed locks, pub/sub, or leader election. Redis is a cache/session store only.
- After any schema change, run `/audit-schema-sync` to verify `schema.sql` and the newest migration agree.
- User-facing error copy follows **status, then implication, then action**.

## Verified starting points

Read these before starting; the line numbers were correct as of this plan's date but confirm them.

| Fact | Location |
|---|---|
| Auth middleware, api-key branch then session branch | `cmd/middlewares.go`, `authenticateUser` |
| Existing API key validation (bcrypt, no cache) | `internal/user/user.go`, `ValidateAPIKey` |
| Existing api_key/api_secret columns on users | `schema.sql`, users table |
| Websocket origin check that rejects native clients | `cmd/websocket.go`, `agentUpgrader.CheckOrigin` |
| OIDC login and callback handlers | `cmd/auth.go`, `handleOIDCLogin` / `handleOIDCCallback` |
| Route registration | `cmd/handlers.go` |
| Migration registry | `cmd/upgrade.go` (latest entry is `v2.6.0`) |
| Migration file shape | `internal/migrations/v2.6.0.go` |

---

### Task B1: Device tokens

**Model:** Opus

**Files:**
- Create: `internal/user/device_token.go`
- Create: `internal/user/device_token_test.go`
- Modify: `internal/user/queries.sql`
- Modify: `internal/user/models/models.go`
- Modify: `internal/user/user.go` (add the new prepared queries to the queries struct)
- Modify: `schema.sql`
- Create: `internal/migrations/v2.7.0.go`
- Modify: `cmd/upgrade.go`
- Modify: `cmd/handlers.go`
- Create: `cmd/device_token.go`

**Interfaces produced:**
- `models.DeviceToken` struct with `ID`, `CreatedAt`, `UserID`, `Name`, `LastUsedAt`, `ExpiresAt`, `RevokedAt`. The verifier hash and selector are never serialised to JSON.
- `(*Manager) MintDeviceToken(userID int, name string) (string, models.DeviceToken, error)` - returns the plaintext token **once**.
- `(*Manager) ValidateDeviceToken(token string) (models.User, error)`
- `(*Manager) ListDeviceTokens(userID int) ([]models.DeviceToken, error)`
- `(*Manager) RevokeDeviceToken(userID, tokenID int) error`
- Endpoints: `POST /api/v1/auth/device-token`, `GET /api/v1/agents/me/device-tokens`, `DELETE /api/v1/agents/me/device-tokens/{id}`.

**Design constraints that are not negotiable:**

1. **Do not reuse the API key mechanism.** `ValidateAPIKey` bcrypt-compares on every request with no cache, which is 60-100ms of CPU per API call. Today's API key is also a single `api_key`/`api_secret` column pair on the user, so re-minting would destroy the agent's existing integration. Device tokens are many-per-user and separate.
2. **Minting requires the password**, and is blocked when the caller authenticated with a device token. A stolen token must not be able to mint siblings or extend itself.
3. Token format is `ld_<selector>_<verifier>`. `selector` is looked up by index; `verifier` is compared in constant time against `sha256(verifier)`. SHA-256 is correct here and bcrypt is not: the verifier is 32 bytes of `crypto/rand`, not a human-chosen password, so there is nothing to brute force and per-request bcrypt cost buys nothing.
4. `expires_at` is 90 days out and pushed forward on use, but **written only when more than a day stale**, so validation is not a DB write per request.

- [ ] **Step 1: Write the schema into schema.sql**

```sql
CREATE TABLE user_device_tokens (
    id              SERIAL PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id         INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    selector        TEXT NOT NULL UNIQUE,
    verifier_hash   BYTEA NOT NULL,
    last_used_at    TIMESTAMPTZ NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ NULL
);
CREATE INDEX index_user_device_tokens_on_user_id ON user_device_tokens(user_id);
```

- [ ] **Step 2: Write the migration**

`internal/migrations/v2.7.0.go`, following the shape of `v2.6.0.go`:

```go
package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_device_tokens (
			id              SERIAL PRIMARY KEY,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			user_id         INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name            TEXT NOT NULL,
			selector        TEXT NOT NULL UNIQUE,
			verifier_hash   BYTEA NOT NULL,
			last_used_at    TIMESTAMPTZ NULL,
			expires_at      TIMESTAMPTZ NOT NULL,
			revoked_at      TIMESTAMPTZ NULL
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_user_device_tokens_on_user_id ON user_device_tokens(user_id);`); err != nil {
		return err
	}
	return nil
}
```

Register it in `cmd/upgrade.go` directly after the `v2.6.0` entry:

```go
	{"v2.7.0", migrations.V2_7_0},
```

- [ ] **Step 3: Add the queries**

In `internal/user/queries.sql`:

```sql
-- name: insert-device-token
INSERT INTO user_device_tokens (user_id, name, selector, verifier_hash, expires_at)
VALUES ($1, $2, $3, $4, NOW() + INTERVAL '90 days')
RETURNING id, created_at, user_id, name, last_used_at, expires_at, revoked_at;

-- name: get-device-token-by-selector
SELECT id, user_id, verifier_hash, expires_at, revoked_at, last_used_at
FROM user_device_tokens
WHERE selector = $1;

-- name: touch-device-token
UPDATE user_device_tokens
SET last_used_at = NOW(), expires_at = NOW() + INTERVAL '90 days'
WHERE id = $1;

-- name: get-device-tokens
SELECT id, created_at, user_id, name, last_used_at, expires_at, revoked_at
FROM user_device_tokens
WHERE user_id = $1 AND revoked_at IS NULL
ORDER BY created_at DESC;

-- name: revoke-device-token
UPDATE user_device_tokens
SET revoked_at = NOW()
WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL;
```

Add the matching fields to the package's prepared-queries struct in `internal/user/user.go`, following how the existing queries are declared there.

- [ ] **Step 4: Add the model**

In `internal/user/models/models.go`, placed with the other structs:

```go
type DeviceToken struct {
	ID           int         `db:"id" json:"id"`
	CreatedAt    time.Time   `db:"created_at" json:"created_at"`
	UserID       int         `db:"user_id" json:"user_id"`
	Name         string      `db:"name" json:"name"`
	LastUsedAt   null.Time   `db:"last_used_at" json:"last_used_at"`
	ExpiresAt    time.Time   `db:"expires_at" json:"expires_at"`
	RevokedAt    null.Time   `db:"revoked_at" json:"revoked_at"`
	SelectorID   int         `db:"-" json:"-"`
	VerifierHash []byte      `db:"verifier_hash" json:"-"`
}
```

- [ ] **Step 5: Write the failing tests**

`internal/user/device_token_test.go`. These must cover, at minimum:

1. A minted token round-trips through `ValidateDeviceToken` and returns the right user.
2. Right selector, wrong verifier fails.
3. A revoked token fails.
4. An expired token fails.
5. An unknown selector fails.
6. A malformed token string (`"garbage"`, `"ld_only_two"` with a missing part, wrong prefix) fails **without a database round trip**.
7. Two mints for the same user produce different selectors and different tokens.
8. The plaintext token is returned only from `MintDeviceToken` and never appears in the `DeviceToken` JSON.

Use the package's existing test setup for the DB-backed cases. Keep the parsing tests pure so they run without Postgres.

- [ ] **Step 6: Run the tests and confirm they fail**

```bash
cd ~/projects/libredesk-main-3 && go test ./internal/user/...
```

Expected: FAIL, `MintDeviceToken` / `ValidateDeviceToken` undefined.

- [ ] **Step 7: Implement internal/user/device_token.go**

Respect the file layout order: consts and vars first, then the parse helper's types, then methods on the existing manager, then private funcs last.

```go
package user

const (
	deviceTokenPrefix   = "ld"
	deviceTokenParts    = 3
	deviceTokenTTL      = 90 * 24 * time.Hour
	deviceTokenTouchGap = 24 * time.Hour
)

var errMalformedDeviceToken = errors.New("malformed device token")
```

`MintDeviceToken` generates 16 random bytes for the selector and 32 for the verifier via `crypto/rand`, hex-encodes both, stores `sha256.Sum256([]byte(verifier))` and returns `"ld_" + selector + "_" + verifier`.

`ValidateDeviceToken` splits on `_`, rejects anything that is not three parts or does not start with `ld` before touching the database, looks up by selector, compares with `subtle.ConstantTimeCompare`, then rejects when `revoked_at IS NOT NULL` or `expires_at < NOW()`. Only if `last_used_at` is null or older than `deviceTokenTouchGap` does it run `touch-device-token`. It then loads the user through `GetAgentCachedOrLoad` so the existing agent cache is reused.

- [ ] **Step 8: Run the tests**

```bash
go test ./internal/user/...
```

Expected: PASS.

- [ ] **Step 9: Add the handlers**

`cmd/device_token.go`:

- `handleCreateDeviceToken` - body is `{"email", "password", "name"}`. Validates the password through the same path `handleLogin` uses (`cmd/login.go`, note the field is `email`, not username). **Rejects the request when `r.RequestCtx.UserValue("auth_method") == "device_token"`.** Returns `{"id", "token", "name"}` once.
- `handleGetDeviceTokens` - lists the caller's own tokens.
- `handleDeleteDeviceToken` - revokes one of the caller's own tokens by id, scoped by `user_id` in the SQL so an agent cannot revoke someone else's.

- [ ] **Step 10: Register the routes**

In `cmd/handlers.go`, beside the other auth routes:

```go
	g.POST("/api/v1/auth/device-token", rateLimit(handleCreateDeviceToken, "auth"))
```

and beside the other `agents/me` routes:

```go
	g.GET("/api/v1/agents/me/device-tokens", auth(handleGetDeviceTokens))
	g.DELETE("/api/v1/agents/me/device-tokens/{id}", auth(handleDeleteDeviceToken))
```

Minting sits in the `auth` rate-limit group. Listing and revoking are under plain `auth` deliberately, so an agent can revoke their own lost phone without an admin. The existing `users:manage` admin path stays for revoking other people's.

- [ ] **Step 11: Verify schema sync**

Run `/audit-schema-sync` and confirm `schema.sql` and `internal/migrations/v2.7.0.go` agree.

- [ ] **Step 12: Commit**

```bash
git add internal/user internal/migrations schema.sql cmd
git commit -m "add per-device tokens for mobile authentication"
```

---

### Task B2: Bearer middleware

**Model:** Opus

**Files:**
- Modify: `cmd/middlewares.go`

**Interfaces consumed:** `ValidateDeviceToken` (B1).

The new branch goes **before** the session path, beside the existing Basic api-key branch. Note that `authenticateUser` currently calls `r.ParseAuthHeader(fastglue.AuthBasic | fastglue.AuthToken)`; a `Bearer` header will not be handled by that call, so read and parse the header explicitly.

- [ ] **Step 1: Add the bearer branch**

Directly after the api-key branch in `authenticateUser`:

```go
	if hdr := string(r.RequestCtx.Request.Header.Peek("Authorization")); strings.HasPrefix(hdr, "Bearer ") {
		user, err = app.user.ValidateDeviceToken(strings.TrimPrefix(hdr, "Bearer "))
		if err != nil {
			return user, err
		}
		r.RequestCtx.SetUserValue("auth_method", "device_token")
		return user, nil
	}
```

- [ ] **Step 2: Confirm the disabled-user check still applies**

The session branch destroys the session and returns a 403 for a disabled user. The device-token branch must reach the same disabled check. If that check lives inside the session branch only, hoist it so all three auth methods run it. A disabled agent holding a valid token must not get through.

- [ ] **Step 3: Verify the failure statuses by hand**

- An invalid or revoked token returns **401**.
- A disabled user with a valid token returns **403** with a message readable to a human.
- CSRF is **not** checked on the device-token path. It is browser-cookie defence and does not apply to a bearer client.

- [ ] **Step 4: Commit**

```bash
git commit -am "accept bearer device tokens in the auth middleware"
```

---

### Task B3: OIDC one-time code exchange

**Model:** Opus

**Files:**
- Modify: `cmd/auth.go`
- Modify: `cmd/handlers.go`

This is required, not optional. The system browser deliberately does not expose its cookie jar to a native app, so there is no client-side way for the app to capture an OIDC session. A one-time code bound to a PKCE challenge is the only route.

It also closes an **existing open redirect**: `next` is currently taken verbatim from a query parameter and used as the redirect target in `handleOIDCLogin` / `handleOIDCCallback`. The fix is a whitelist, not a sanitiser.

The mobile app already implements its half: it generates a 64-character verifier from the RFC 7636 unreserved set, sends `code_challenge = base64url(sha256(verifier))` unpadded plus `client=mobile`, and expects the callback on `libredesk://auth?code=...`.

- [ ] **Step 1: Whitelist the redirect targets**

Replace the verbatim `next` usage with an allowlist: the configured app root URL and its subpaths, plus the literal `libredesk://auth` when the request is marked mobile. Anything else is rejected outright, not rewritten.

- [ ] **Step 2: Accept the challenge on the login route**

`GET /api/v1/oidc/{id}/login` reads `code_challenge` and `client`. When `client=mobile`, carry both through the OIDC `state` so they survive the round trip to the identity provider.

- [ ] **Step 3: Mint the one-time code on callback**

On a successful callback with a mobile marker, store in Redis:

- key: a fresh random code
- value: the user id plus the `code_challenge`
- TTL: 60 seconds
- single use: delete on read

Then redirect to `libredesk://auth?code=<code>`.

- [ ] **Step 4: Add the exchange endpoint**

`POST /api/v1/auth/token/exchange`, in the `auth` rate-limit group, body `{"code", "code_verifier", "name"}`:

1. Read and **delete** the code from Redis in one step.
2. Verify `base64url_unpadded(sha256(code_verifier)) == stored challenge`.
3. Mint a device token via `MintDeviceToken` (B1) and return `{"id", "token", "email"}`.

- [ ] **Step 5: Test the failure paths**

A replayed code fails. A wrong verifier fails. An expired code fails. A code exchanged with no verifier fails. Each returns 401, not 500.

- [ ] **Step 6: Commit**

```bash
git commit -am "add OIDC one-time code exchange for native clients"
```

---

### Task B4: Websocket token auth

**Model:** Opus

**Files:**
- Modify: `cmd/websocket.go`

`agentUpgrader.CheckOrigin` returns `false` when `Origin` is empty. A native client sends no `Origin`, so it is rejected today. That check is browser CSRF defence and is exactly what must be skipped for a token-authenticated upgrade - and only for that case.

- [ ] **Step 1: Detect the token-authenticated upgrade**

The `/ws` route already runs through `auth(...)`, so by the time the upgrade happens the request context carries `auth_method`. Read it and skip the origin check when it is `device_token`.

- [ ] **Step 2: Keep the cookie path strict**

`CheckOrigin` must still return `false` for an empty `Origin` on a session-authenticated upgrade. Do not weaken the browser path while enabling the native one. If the upgrader is a package-level `var`, this likely means selecting between two upgraders inside the handler rather than mutating the shared one.

- [ ] **Step 3: Verify by hand**

- A browser session upgrade with a mismatched `Origin` is still rejected.
- A browser session upgrade with no `Origin` is still rejected.
- A `Bearer`-authenticated upgrade with no `Origin` succeeds.

- [ ] **Step 4: Commit**

```bash
git commit -am "allow token-authenticated websocket upgrades from native clients"
```

---

## What the mobile app expects, exactly

The app is already written against these shapes. Matching them means no client change is needed.

| Method | Path | Request | Response `data` |
|---|---|---|---|
| POST | `/api/v1/auth/device-token` | `{"email","password","name"}` | `{"id": int, "token": "ld_..._...", "name": string}` |
| POST | `/api/v1/auth/token/exchange` | `{"code","code_verifier","name"}` | `{"id": int, "token": string, "email": string}` |
| GET | `/api/v1/agents/me/device-tokens` | - | array of device tokens |
| DELETE | `/api/v1/agents/me/device-tokens/{id}` | - | any |
| GET | `/api/v1/oidc/{id}/login` | query `code_challenge`, `client=mobile` | redirect to `libredesk://auth?code=...` |
| GET | `/ws` | `Authorization: Bearer <token>`, no `Origin` | websocket upgrade |

The app keys **all** of its error handling off the HTTP status, not the envelope type, because session failures already return `GeneralException` with a 401. Keep the statuses right and the type string matters less.

## Out of scope

Push notifications. The Go backend has no push code at all today, and whether every operator supplies their own APNs/FCM credentials or Libredesk runs a hosted relay is undecided. The mobile app ships v1 without push, so nothing here depends on it.

## Definition of done

- [ ] `go test ./...` passes.
- [ ] `/audit-schema-sync` reports `schema.sql` and `internal/migrations/v2.7.0.go` in sync.
- [ ] A device token minted with a password authenticates an ordinary API call.
- [ ] That same token **cannot** mint another device token.
- [ ] A revoked token gets a 401 on the next request.
- [ ] `/ws` upgrades with a bearer token and no `Origin` header, while a browser upgrade with a bad `Origin` is still rejected.
