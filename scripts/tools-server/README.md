# AI assistant custom tools test server

A small HTTP server for trying out custom tools without wiring up a real backend.
It holds two fake accounts and looks them up by the contact's external user id,
which libredesk sends in the `X-Libredesk-Contact-External-Id` header. Add your
own test users by editing the `users` map in `main.go`.

Custom tools are only called by AI assistants. Copilot gets the built-in tools
and never touches these.

## Run

```
go run ./scripts/tools-server
```

It listens on `:7070`, or pass `-addr` to change that. Every request logs the contact
headers and the raw args JSON the model sent, which is usually enough to work out
why a tool call did not do what you expected.

## Headers libredesk sends

Libredesk injects the contact and conversation context on every custom tool call.
This happens server-side, so the model cannot see or change any of it:

| Header                            | Value                                              |
| --------------------------------- | -------------------------------------------------- |
| `X-Libredesk-Contact-Id`          | internal contact id                                |
| `X-Libredesk-Contact-External-Id` | external user id (from the livechat JWT), if any    |
| `X-Libredesk-Contact-Type`        | `contact` or `visitor`                             |
| `X-Libredesk-Contact-Email`       | contact email, if known                            |
| `X-Libredesk-Contact-Verified`    | `true` only after OTP/JWT identity verification    |
| `X-Libredesk-Conversation-UUID`   | conversation uuid                                  |
| `X-Libredesk-Inbox-Id`            | inbox id                                           |

A visitor can type any email address into a chat, so only trust
`X-Libredesk-Contact-Email` for account data when `X-Libredesk-Contact-Verified`
is `true`. This server returns a 403 otherwise.

Work out who the customer is from these headers and nothing else. The request body
holds the model's arguments, written from whatever the customer typed, so it must
never decide which account you return.

## Endpoints

| Endpoint   | Returns                       |
| ---------- | ----------------------------- |
| `/account` | name, email, plan, KYC status |
| `/orders`  | recent orders with status     |
| `/balance` | account balance               |

## Tool setup (Admin -> AI -> Tools)

Create one tool per endpoint:

- Method: `POST`
- Auth header: `X-Api-Key`
- Auth value: `test-secret-token`
- Parameters: leave empty

For example, name `get_account`, URL `http://localhost:7070/account`, description
"Get the customer's account details: name, email, plan, and KYC status."

Grant the tools to an assistant under **Admin -> AI -> Assistants**, then assign a
conversation to that assistant. Nothing gets called until you do both.

## Test contacts

Set the contact's external user id and email to one of:

- `USR1001` / alice@example.com - pro plan, KYC verified, has orders
- `USR1002` / bob@example.com - free plan, KYC pending, no orders

Error paths: bad API key -> 401, unverified contact -> 403, missing external id -> 400,
unknown id -> 404, email mismatch -> 403.
