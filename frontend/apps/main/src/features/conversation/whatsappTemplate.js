export const WHATSAPP_CHANNEL = 'whatsapp'
export const WHATSAPP_WINDOW_MS = 24 * 60 * 60 * 1000
export const PLACEHOLDER_PATTERN = /\{\{([A-Za-z0-9_]+)\}\}/g

export const supportsWhatsAppTemplateContent = (template) =>
  template.component_types?.includes('BODY') &&
  template.component_types.every((type) =>
    ['HEADER', 'BODY', 'FOOTER', 'BUTTONS'].includes(type)
  ) &&
  template.category?.toUpperCase() !== 'AUTHENTICATION' &&
  ['', 'NONE', 'TEXT'].includes((template.header_type || '').toUpperCase()) &&
  (template.buttons || []).every((button) =>
    ['URL', 'PHONE_NUMBER', 'QUICK_REPLY'].includes((button.type || '').toUpperCase())
  )

export const canEditWhatsAppTemplate = (template) =>
  !template.name.startsWith('libredesk_csat_') &&
  supportsWhatsAppTemplateContent(template) &&
  (template.buttons || []).length <= 3 &&
  ['APPROVED', 'REJECTED', 'PAUSED'].includes(template.status)

export const whatsAppWindowInboundAt = (conversation) => {
  const ts = conversation?.contact_last_inbound_at
  return ts ? new Date(ts).getTime() : 0
}

export const isWhatsAppWindowOpen = (conversation, now = Date.now()) => {
  const ts = whatsAppWindowInboundAt(conversation)
  return ts > 0 && now - ts < WHATSAPP_WINDOW_MS
}

export const extractPlaceholders = (sources) => {
  const seen = new Set()
  const out = []
  for (const src of sources) {
    if (!src) continue
    for (const match of src.matchAll(PLACEHOLDER_PATTERN)) {
      if (!seen.has(match[1])) {
        seen.add(match[1])
        out.push(match[1])
      }
    }
  }
  return out
}

export const placeholderLabel = (key) => `{{${key}}}`
