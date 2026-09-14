export const BUILT_IN_TEMPLATE_BODIES = {
  'Conversation assigned': `<p>A new conversation has been assigned to you:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>`,
  'New reply from contact': `<p>{{ .Author.FullName }} replied to a conversation assigned to you:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<blockquote>{{ .Message.Content }}</blockquote>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>`,
  'New reply on participating conversation': `<p>{{ .Author.FullName }} replied to a conversation you are participating in:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<blockquote>{{ .Message.Content }}</blockquote>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>`,
  'Conversation reopened': `<p>{{ .Author.FullName }} replied and reopened a conversation assigned to you:</p>

<div>
    Reference number: {{ .Conversation.ReferenceNumber }} <br>
    Subject: {{ .Conversation.Subject }}
</div>

<blockquote>{{ .Message.Content }}</blockquote>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<div>
    Best regards,<br>
    Libredesk
</div>`,
  'SLA breach warning': `<p>This is a notification that the SLA for conversation {{ .Conversation.ReferenceNumber }} is approaching the SLA deadline for {{ .SLA.Metric }}.</p>

<p>
  Details:<br>
  - Conversation reference number: {{ .Conversation.ReferenceNumber }}<br>
  - Metric: {{ .SLA.Metric }}<br>
  - Due in: {{ .SLA.DueIn }}
</p>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<p>
  Best regards,<br>
  Libredesk
</p>`,
  'SLA breached': `<p>This is an urgent alert that the SLA for conversation {{ .Conversation.ReferenceNumber }} has been breached for {{ .SLA.Metric }}. Please take immediate action.</p>

<p>
  Details:<br>
  - Conversation reference number: {{ .Conversation.ReferenceNumber }}<br>
  - Metric: {{ .SLA.Metric }}<br>
  - Overdue by: {{ .SLA.OverdueBy }}
</p>

<p>
    <a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .Conversation.UUID }}">View Conversation</a>
</p>

<p>
  Best regards,<br>
  Libredesk
</p>`,
  'Mentioned in conversation': `<p>{{ .MentionedBy.FullName }} mentioned you in a private note on conversation #{{ .Conversation.ReferenceNumber }}.</p>

<blockquote style="background-color: #f5f5f5; padding: 12px; margin: 16px 0; border-left: 4px solid #ddd;">
{{ .Message.Content }}
</blockquote>

<p>
<a href="{{ RootURL }}/inboxes/mentioned/conversation/{{ .Conversation.UUID }}?scrollTo={{ .Message.UUID }}">View Conversation</a>
</p>

<p>
Best regards,<br>
libredesk
</p>`,
  'CSAT request': `<p style="margin: 0 0 4px; font-size: 15px; color: #374151; text-align: center; line-height: 1.5;">
  Your conversation <strong style="color: #111827;">#{{ .Conversation.ReferenceNumber }}</strong> has been resolved.
</p>
<p style="margin: 0 0 28px; font-size: 13px; color: #9ca3af; text-align: center;">
  We would love to hear how it went.
</p>
<p style="margin: 0 0 20px; font-size: 14px; font-weight: 600; color: #374151; text-align: center;">
  How would you rate your experience?
</p>
<!-- Variable CSATUUID is also available -->
<div style="text-align: center; margin: 0 auto; max-width: 400px; font-size: 0;">
  <div style="display: inline-block; width: 72px; text-align: center; vertical-align: top; padding: 4px 0;">
    <a href="{{ .CSATLink }}?rating=1" style="text-decoration: none; display: block;">
      <span style="font-size: 34px; display: block; line-height: 1.4;">&#128546;</span>
      <span style="font-size: 10px; display: block; font-weight: 600; color: #b0b5bd; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 4px;">Poor</span>
    </a>
  </div>
  <div style="display: inline-block; width: 72px; text-align: center; vertical-align: top; padding: 4px 0;">
    <a href="{{ .CSATLink }}?rating=2" style="text-decoration: none; display: block;">
      <span style="font-size: 34px; display: block; line-height: 1.4;">&#128533;</span>
      <span style="font-size: 10px; display: block; font-weight: 600; color: #b0b5bd; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 4px;">Fair</span>
    </a>
  </div>
  <div style="display: inline-block; width: 72px; text-align: center; vertical-align: top; padding: 4px 0;">
    <a href="{{ .CSATLink }}?rating=3" style="text-decoration: none; display: block;">
      <span style="font-size: 34px; display: block; line-height: 1.4;">&#128522;</span>
      <span style="font-size: 10px; display: block; font-weight: 600; color: #b0b5bd; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 4px;">Good</span>
    </a>
  </div>
  <div style="display: inline-block; width: 72px; text-align: center; vertical-align: top; padding: 4px 0;">
    <a href="{{ .CSATLink }}?rating=4" style="text-decoration: none; display: block;">
      <span style="font-size: 34px; display: block; line-height: 1.4;">&#128515;</span>
      <span style="font-size: 10px; display: block; font-weight: 600; color: #b0b5bd; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 4px;">Great</span>
    </a>
  </div>
  <div style="display: inline-block; width: 72px; text-align: center; vertical-align: top; padding: 4px 0;">
    <a href="{{ .CSATLink }}?rating=5" style="text-decoration: none; display: block;">
      <span style="font-size: 34px; display: block; line-height: 1.4;">&#129321;</span>
      <span style="font-size: 10px; display: block; font-weight: 600; color: #b0b5bd; text-transform: uppercase; letter-spacing: 0.05em; margin-top: 4px;">Excellent</span>
    </a>
  </div>
</div>`
}
