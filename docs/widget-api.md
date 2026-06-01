# Widget API

This document summarizes the public endpoints and WebSocket messages used by
the Libredesk live chat widget. These APIs are intended for custom widget
frontends that talk to an existing Libredesk instance.

## Authentication and inbox selection

Widget HTTP requests identify the live chat inbox with either:

- `X-Libredesk-Inbox-ID: <inbox_uuid>` header
- `?inbox_id=<inbox_uuid>` query parameter

Authenticated widget requests also send:

- `Authorization: Bearer <session_token>`

When a verified contact is replacing a previous visitor session, the widget can
also send:

- `X-Libredesk-Visitor-Token: <visitor_session_token>`

If Libredesk merges the visitor into the verified contact, the response includes
`X-Libredesk-Clear-Visitor: true` so the custom frontend can discard the visitor
token.

API responses use Libredesk's standard envelope format. Successful responses put
the endpoint payload in `data`.

## Public settings endpoints

### `GET /api/v1/widget/chat/settings/launcher`

Returns launcher-only settings for the embeddable script before the widget
iframe is opened. The inbox can be passed as `?inbox_id=<inbox_uuid>`.

### `GET /api/v1/widget/chat/settings`

Returns the live chat widget settings, including public configuration and, when
enabled, business hours and pre-chat custom attribute metadata.

## Session endpoints

### `POST /api/v1/widget/chat/auth/exchange`

Exchanges a customer-generated JWT for a widget session token.

Request body:

```json
{
  "jwt": "<signed_customer_jwt>"
}
```

The JWT must include `external_user_id`, `email`, and `first_name`. It can also
include `last_name` and `contact_custom_attributes`.

Response `data` includes:

```json
{
  "session_token": "<session_token>",
  "user": {
    "user_id": 123,
    "is_visitor": false,
    "first_name": "Ada",
    "last_name": "Lovelace"
  }
}
```

### `GET /api/v1/widget/chat/auth/me`

Returns the current widget user's metadata for the bearer session token.

## Conversation endpoints

### `POST /api/v1/widget/chat/conversations/init`

Starts a new live chat conversation. If no bearer token is present, Libredesk
creates a visitor and returns a new session token.

Request body:

```json
{
  "message": "Hello, I need help",
  "form_data": {
    "company": "Example Co"
  }
}
```

Response `data` includes the created `conversation`, `messages`, optional
business hours fields, and, for a new visitor, `session_token` plus `user`.

### `GET /api/v1/widget/chat/conversations`

Returns the conversations visible to the current widget user for the selected
inbox.

### `GET /api/v1/widget/chat/conversations/{uuid}`

Returns one conversation with its messages and optional business hours metadata.

### `POST /api/v1/widget/chat/conversations/{uuid}/message`

Sends a text message to an existing conversation.

Request body:

```json
{
  "message": "Here are more details"
}
```

### `POST /api/v1/widget/chat/conversations/{uuid}/update-last-seen`

Marks the conversation as seen by the widget user.

## Upload endpoint

### `POST /api/v1/widget/media/upload`

Uploads one or more files to an existing conversation. This endpoint requires
`multipart/form-data`.

Form fields:

- `conversation_uuid`: target conversation UUID
- `files`: one or more file parts

File uploads are rejected when the inbox has file upload disabled, the file is
empty, the file is larger than the configured limit, or the extension is not
allowed.

## WebSocket endpoint

### `GET /widget/ws`

The widget uses this WebSocket endpoint for realtime conversation events.

After opening the socket, send a `join` message with the inbox UUID and session
token:

```json
{
  "type": "join",
  "token": "<session_token>",
  "data": {
    "inbox_id": "<inbox_uuid>"
  }
}
```

The server replies with:

```json
{
  "type": "joined",
  "data": {
    "message": "namaste!"
  }
}
```

### Client-to-server messages

`typing` broadcasts the visitor typing state to agents:

```json
{
  "type": "typing",
  "data": {
    "conversation_uuid": "<conversation_uuid>",
    "is_typing": true
  }
}
```

`page_visit` stores the visitor's current page and broadcasts recent page visits
to agents:

```json
{
  "type": "page_visit",
  "data": {
    "url": "https://example.com/pricing",
    "title": "Pricing"
  }
}
```

`ping` keeps the session active and should be sent periodically:

```json
{
  "type": "ping"
}
```

The server replies with `pong`.

### Server-to-client messages

The widget client handles these message types:

- `joined`: the socket joined the live chat inbox.
- `pong`: response to `ping`.
- `new_message`: a new chat message; `data` is a chat message payload.
- `typing`: agent typing status with `conversation_uuid` and `is_typing`.
- `conversation_update`: partial conversation update.
- `error`: socket-level error payload.
