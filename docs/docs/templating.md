# Templating

Templating in outgoing emails allows you to personalize content by embedding dynamic expressions like `{{ .Recipient.FullName }}`. These expressions reference fields from the conversation, contact, recipient, and author objects.

## Outgoing Email Template Expressions

If you want to customize the look of outgoing emails, you can do so in the Admin > Templates -> Outgoing Email Templates section. This template will be used for all outgoing emails including replies to conversations, notifications, and other system-generated emails.

### Conversation Variables

| Variable | Value |
|---------------------------------|--------------------------------------------------------|
| {{ .Conversation.ReferenceNumber }} | The unique reference number of the conversation |
| {{ .Conversation.Subject }} | The subject of the conversation |
| {{ .Conversation.Priority }} | The priority level of the conversation |
| {{ .Conversation.UUID }} | The unique identifier of the conversation |

### Contact Variables

| Variable | Value |
|------------------------------|------------------------------------|
| {{ .Contact.FirstName }} | First name of the contact/customer |
| {{ .Contact.LastName }} | Last name of the contact/customer |
| {{ .Contact.FullName }} | Full name of the contact/customer |
| {{ .Contact.Email }} | Email address of the contact/customer |

### Recipient Variables

| Variable | Value |
|--------------------------------|-----------------------------------|
| {{ .Recipient.FirstName }} | First name of the recipient |
| {{ .Recipient.LastName }} | Last name of the recipient |
| {{ .Recipient.FullName }} | Full name of the recipient |
| {{ .Recipient.Email }} | Email address of the recipient |

### Author Variables

| Variable | Value |
|------------------------------|-----------------------------------|
| {{ .Author.FirstName }} | First name of the message author |
| {{ .Author.LastName }} | Last name of the message author |
| {{ .Author.FullName }} | Full name of the message author |
| {{ .Author.Email }} | Email address of the message author |

### Example outgoing email template

```
Dear {{ .Recipient.FirstName }},

{{ template "content" . }}

Best regards,
{{ .Author.FullName }}
---
Reference: {{ .Conversation.ReferenceNumber }}
```

#### Important: How templates work with your responses

When this template is set as default, it automatically wraps around your ticket responses. The `{{ template "content" . }}` placeholder is where your response text will be inserted.

#### Template expressions
The template expressions like `{{ .Recipient.FirstName }}` and `{{ .Author.FullName }}` dynamically insert the appropriate information when the email is sent.

#### What this means for you:
- **DO NOT** include greetings (like "Hello" or "Dear Customer") in your response
- **DO NOT** include sign-offs (like "Best regards" or your name) in your response  
- **ONLY** write the main body of your message

#### Example of CORRECT usage:
When you write in the response field:
```
Thank you for contacting us. I've reviewed your account and can confirm that your refund has been processed.
```

The customer receives:
```
Dear John,

Thank you for contacting us. I've reviewed your account and can confirm that your refund has been processed.

Best regards,
Sarah Smith
---
Reference: TKT-2024-001
```

#### Example of INCORRECT usage (creates duplication):
When you write in the response field:
```
Hello John,

Thank you for contacting us. I've reviewed your account and can confirm that your refund has been processed.

Best regards,
Sarah
```

The customer receives (with unwanted duplication):
```
Dear John,

Hello John,

Thank you for contacting us. I've reviewed your account and can confirm that your refund has been processed.

Best regards,
Sarah

Best regards,
Sarah Smith
---
Reference: TKT-2024-001
```

#### Working with different templates

If your organization uses a different template structure, adjust your response accordingly. For example:
- A template with no greeting would require you to include your own
- A template with only `{{ template "content" . }}` and nothing else would require you to write complete emails
- Always check your active template configuration to understand what's automatically included
