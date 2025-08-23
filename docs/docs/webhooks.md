# Webhooks

Webhooks allow you to receive real-time HTTP notifications when specific events occur in your Libredesk instance. This enables you to integrate Libredesk with external systems and automate workflows based on conversation and message events.

## Overview

When a configured event occurs in Libredesk, a HTTP POST request is sent to the webhook URL you specify. The request contains a JSON payload with event details and relevant data.

## Webhook Configuration

1. Navigate to **Admin > Integrations > Webhooks** in your Libredesk dashboard
2. Click **Create Webhook**
3. Configure the following:
   - **Name**: A descriptive name for your webhook
   - **URL**: The endpoint URL where webhook payloads will be sent
   - **Events**: Select which events you want to subscribe to
   - **Secret**: Optional secret key for signature verification
   - **Status**: Enable or disable the webhook

## Security

### Signature Verification

If you provide a secret key, webhook payloads will be signed using HMAC-SHA256. The signature is included in the `X-Signature-256` header in the format `sha256=<signature>`.

To verify the signature:

```python
import hmac
import hashlib

def verify_signature(payload, signature, secret):
    expected_signature = hmac.new(
        secret.encode('utf-8'),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected_signature}", signature)
```

### Headers

Each webhook request includes the following headers:

- `Content-Type`: `application/json`
- `User-Agent`: `Libredesk-Webhook/<libredesk_version_here>`
- `X-Signature-256`: HMAC signature (if secret is configured)

## Available Events

### Conversation Events

#### `conversation.created`
Triggered when a new conversation is created.

**Sample Payload:**
```json
{
  "event": "conversation.created",
  "timestamp": "2025-06-15T10:30:00Z",
  "payload": {
    "id": 123,
    "created_at": "2025-06-15T10:30:00Z",
    "updated_at": "2025-06-15T10:30:00Z",
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "contact_id": 456,
    "inbox_id": 1,
    "reference_number": "100",
    "priority": "Medium",
    "priority_id": 2,
    "status": "Open",
    "status_id": 1,
    "subject": "Help with account setup",
    "inbox_name": "Support",
    "inbox_channel": "email",
    "contact": {
      "id": 456,
      "first_name": "John",
      "last_name": "Doe",
      "email": "john.doe@example.com",
      "type": "contact"
    },
    "custom_attributes": {},
    "tags": []
  }
}
```

#### `conversation.status_changed`
Triggered when a conversation's status is updated.

**Sample Payload:**
```json
{
  "event": "conversation.status_changed",
  "timestamp": "2025-06-15T10:35:00Z",
  "payload": {
    "conversation_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "previous_status": "Open",
    "new_status": "Resolved",
    "snooze_until": "",
    "actor_id": 789
  }
}
```

#### `conversation.assigned`
Triggered when a conversation is assigned to a user.

**Sample Payload:**
```json
{
  "event": "conversation.assigned",
  "timestamp": "2025-06-15T10:32:00Z",
  "payload": {
    "conversation_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "assigned_to": 789,
    "actor_id": 789
  }
}
```

#### `conversation.unassigned`
Triggered when a conversation is unassigned from a user.

**Sample Payload:**
```json
{
  "event": "conversation.unassigned",
  "timestamp": "2025-06-15T10:40:00Z",
  "payload": {
    "conversation_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "actor_id": 789
  }
}
```

#### `conversation.tags_changed`
Triggered when tags are added or removed from a conversation.

**Sample Payload:**
```json
{
  "event": "conversation.tags_changed",
  "timestamp": "2025-06-15T10:45:00Z",
  "payload": {
    "conversation_uuid": "550e8400-e29b-41d4-a716-446655440000",
    "previous_tags": ["bug", "priority"],
    "new_tags": ["bug", "priority", "resolved"],
    "actor_id": 789
  }
}
```

### Message Events

#### `message.created`
Triggered when a new message is created in a conversation.

**Sample Payload:**
```json
{
  "event": "message.created",
  "timestamp": "2025-06-15T10:33:00Z",
  "payload": {
    "id": 987,
    "created_at": "2025-06-15T10:33:00Z",
    "updated_at": "2025-06-15T10:33:00Z",
    "uuid": "123e4567-e89b-12d3-a456-426614174000",
    "type": "outgoing",
    "status": "sent",
    "conversation_id": 123,
    "content": "<p>Hello! How can I help you today?</p>",
    "text_content": "Hello! How can I help you today?",
    "content_type": "html",
    "private": false,
    "sender_id": 789,
    "sender_type": "agent",
    "attachments": []
  }
}
```

#### `message.updated`
Triggered when an existing message is updated.

**Sample Payload:**
```json
{
  "event": "message.updated",
  "timestamp": "2025-06-15T10:34:00Z",
  "payload": {
    "id": 987,
    "created_at": "2025-06-15T10:33:00Z",
    "updated_at": "2025-06-15T10:34:00Z",
    "uuid": "123e4567-e89b-12d3-a456-426614174000",
    "type": "outgoing",
    "status": "sent",
    "conversation_id": 123,
    "content": "<p>Hello! How can I help you today? (Updated)</p>",
    "text_content": "Hello! How can I help you today? (Updated)",
    "content_type": "html",
    "private": false,
    "sender_id": 789,
    "sender_type": "agent",
    "attachments": []
  }
}
```

## Delivery and Retries

- Webhooks requests timeout can be configured in the `config.toml` file
- Failed deliveries are not automatically retried
- Webhook delivery runs in a background worker pool for better performance
- If the webhook queue is full (configurable in config.toml file), new events may be dropped

## Testing Webhooks

You can test your webhook configuration using tools like:

- [Webhook.site](https://webhook.site) - Generate a temporary URL to inspect webhook payloads