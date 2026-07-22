# AI copilot tools test server

A minimal HTTP server for testing AI copilot custom tools locally. Maps a contact's
external user id (sent by libredesk in the `X-Libredesk-Contact-External-Id` header)
to fake account data. Edit the `users` map in `main.go` to add test users.

## Run

```
go run ./scripts/tools-server
```

Listens on `:7070` by default (`-addr` to change). Every request logs the contact
headers and the raw args JSON the model sent.

## Headers libredesk sends

On every custom tool call libredesk injects the contact and conversation context
server-side (the model never sees or controls these):

| Header                            | Value                                              |
| --------------------------------- | -------------------------------------------------- |
| `X-Libredesk-Contact-Id`          | internal contact id                                |
| `X-Libredesk-Contact-External-Id` | external user id (from the livechat JWT), if any    |
| `X-Libredesk-Contact-Type`        | `contact` or `visitor`                             |
| `X-Libredesk-Contact-Email`       | contact email, if known                            |
| `X-Libredesk-Contact-Verified`    | `true` only after OTP/JWT identity verification    |
| `X-Libredesk-Conversation-UUID`   | conversation uuid                                  |
| `X-Libredesk-Inbox-Id`            | inbox id                                           |

Trust `X-Libredesk-Contact-Email` for sensitive lookups only when
`X-Libredesk-Contact-Verified` is `true`; a `visitor` can self-claim any address.

## Endpoints

| Endpoint   | Returns                       |
| ---------- | ----------------------------- |
| `/account` | name, email, plan, KYC status |
| `/orders`  | recent orders with status     |
| `/balance` | account balance               |

## Tool setup (Admin -> AI -> Tools)

For each endpoint create a tool with:

- Method: `POST`
- Auth header: `X-Api-Key`
- Auth value: `test-secret-token`
- Parameters: leave empty

Example: name `get_account`, URL `http://localhost:7070/account`, description
"Get the customer's account details: name, email, plan, and KYC status."

## Test contacts

Set the contact's external user id and email to one of:

- `USR1001` / alice@example.com - pro plan, KYC verified, has orders
- `USR1002` / bob@example.com - free plan, KYC pending, no orders

Error paths: bad API key -> 401, missing external id -> 400, unknown id -> 404,
email mismatch -> 403.
