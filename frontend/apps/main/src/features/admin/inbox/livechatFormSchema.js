import { z } from 'zod'
import { isGoDuration } from '@shared-ui/utils/string'

const hexColorRegex = /^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/
const hexColor = (t) => z.string().regex(hexColorRegex, { message: t('validation.invalidColor') })
const optionalHexColor = (t) => hexColor(t).optional().or(z.literal(''))
const optionalUrl = (t) => z.string().url({ message: t('validation.invalidUrl') }).optional().or(z.literal(''))
const rangeNumber = (t, min, max) => {
  const msg = t('validation.minmaxNumber', { min, max })
  return z.coerce.number({ invalid_type_error: msg }).min(min, { message: msg }).max(max, { message: msg })
}
const spacingNumber = (t) => rangeNumber(t, 0, 200)

export const defaultWidgetHelp = () => ({
  help_center_id: 0,
  visitors: { tab: true },
  users: { tab: true },
  featured_ids: []
})
export const defaultWidgetPreviews = () => ({ desktop: false, mobile: false, content: 'message', auto_hide_seconds: 0 })
export const widgetConditionsSchema = z.object({
  logical_op: z.enum(['AND', 'OR']),
  rules: z.array(z.object({ field: z.string().min(1), field_type: z.literal('contact_custom_attribute'), operator: z.string().min(1), value: z.string(), case_sensitive_match: z.boolean() })).max(20)
})
export const newCampaignId = () => {
  if (typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}
export const defaultCampaign = () => ({
  id: newCampaignId(), name: '', enabled: false, message: '', sender_id: 0, team_id: 0,
  audience: 'all', include_urls: [], exclude_urls: [], conditions: { logical_op: 'AND', rules: [] },
  event: '', delay_seconds: 10,
  business_hours_id: 0, business_hours: 'any', desktop: true, mobile: true,
  repeat: 'once', repeat_hours: 24,
})
export const campaignSchema = z.object({
  id: z.string().uuid(), name: z.string().trim().min(1).max(128), enabled: z.boolean(), message: z.string().trim().min(1).max(10000),
  sender_id: z.number().int().min(0), team_id: z.number().int().min(0),
  audience: z.enum(['all', 'visitors', 'users']), include_urls: z.array(z.string().max(2048)).max(20).transform(items => items.map(item => item.trim()).filter(Boolean)),
  exclude_urls: z.array(z.string().max(2048)).max(20).transform(items => items.map(item => item.trim()).filter(Boolean)), conditions: widgetConditionsSchema,
  event: z.string().max(128), delay_seconds: z.number().int().min(0).max(86400),
  business_hours_id: z.number().int().min(0), business_hours: z.enum(['any', 'inside', 'outside']),
  desktop: z.boolean(), mobile: z.boolean(),
  repeat: z.enum(['once', 'session', 'interval']), repeat_hours: z.number().int().min(1).max(8760),
})
const helpAudience = z.object({ tab: z.boolean() })

export const createFormSchema = (t) => z.object({
  name: z.string().min(1, { message: t('globals.messages.required') }),
  enabled: z.boolean(),
  csat_enabled: z.boolean(),
  prompt_tags_on_reply: z.boolean(),
  secret: z.string().nullable().optional(),
  linked_email_inbox_id: z.number().nullable().optional(),
  config: z.object({
    campaigns: z.array(campaignSchema).max(50).default([]),
    campaign_cooldown_hours: z.number().int().min(1).max(8760).default(24),
    help: z.object({
      help_center_id: z.number().int().min(0),
      visitors: helpAudience, users: helpAudience,
      featured_ids: z.array(z.number().int().positive()).max(10),
    }).default(defaultWidgetHelp),
    previews: z.object({ desktop: z.boolean(), mobile: z.boolean(), content: z.enum(['message', 'generic']), auto_hide_seconds: z.number().int().min(0).max(300) }).default(defaultWidgetPreviews),
    brand_name: z.string().min(1, { message: t('globals.messages.required') }),
    website_url: optionalUrl(t),
    dark_mode: z.boolean(),
    show_powered_by: z.boolean(),
    language: z.string().min(1, { message: t('globals.messages.required') }),
    fallback_language: z.string().optional(),
    logo_url: optionalUrl(t),
    launcher: z.object({
      position: z.enum(['left', 'right']),
      logo_url: optionalUrl(t),
      color: hexColor(t),
      spacing: z.object({
        side: spacingNumber(t),
        bottom: spacingNumber(t),
      })
    }),
    greeting_message: z.string().optional(),
    introduction_message: z.string().optional(),
    chat_introduction: z.string(),
    show_office_hours_in_chat: z.boolean(),
    show_office_hours_after_assignment: z.boolean(),
    chat_reply_expectation_message: z.string().optional(),
    notice_banner: z.object({
      enabled: z.boolean(),
      text: z.string().optional()
    }).superRefine((nb, ctx) => {
      if (nb.enabled && !nb.text?.trim()) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['text'], message: t('globals.messages.required') })
      }
    }),
    colors: z.object({
      primary: hexColor(t)
    }),
    home_screen: z.object({
      header_text_color: z.enum(['black', 'white']),
      background: z.object({
        type: z.enum(['solid', 'gradient', 'image']),
        color: optionalHexColor(t),
        gradient_start: optionalHexColor(t),
        gradient_end: optionalHexColor(t),
        image_url: optionalUrl(t),
      }).superRefine((bg, ctx) => {
        // An empty solid color is intentional (the widget falls back to the page background),
        // but a gradient/image with no value renders nothing, so require those.
        if (bg.type === 'gradient') {
          if (!bg.gradient_start) ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['gradient_start'], message: t('globals.messages.required') })
          if (!bg.gradient_end) ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['gradient_end'], message: t('globals.messages.required') })
        } else if (bg.type === 'image' && !bg.image_url) {
          ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['image_url'], message: t('globals.messages.required') })
        }
      }),
      fade_background: z.boolean(),
    }),
    features: z.object({
      file_upload: z.boolean(),
      emoji: z.boolean(),
      transcript: z.boolean().default(false),
    }),
    continuity: z.object({
      offline_threshold: z.string().min(1, { message: t('globals.messages.required') }).refine(isGoDuration, { message: t('validation.invalidDuration') }),
      max_messages_per_email: rangeNumber(t, 1, 100),
      min_email_interval: z.string().min(1, { message: t('globals.messages.required') }).refine(isGoDuration, { message: t('validation.invalidDuration') }),
    }).optional(),
    session_duration: z.string().min(1, { message: t('globals.messages.required') }).refine(isGoDuration, { message: t('validation.invalidDuration') }),
    direct_to_conversation: z.boolean().default(false),
    trusted_domains: z.string().optional(),
    blocked_ips: z.string().optional(),
    home_apps: z.array(z.object({
      type: z.enum(['announcement', 'external_link', 'help']),
      title: z.string().optional().or(z.literal('')),
      description: z.string().optional().or(z.literal('')),
      image_url: optionalUrl(t),
      url: optionalUrl(t),
      text: z.string().optional().or(z.literal('')),
    })),
    visitors: z.object({
      start_conversation_button_text: z.string(),
      allow_start_conversation: z.boolean(),
      prevent_multiple_conversations: z.boolean(),
      prevent_reply_to_closed_conversation: z.boolean(),
    }),
    users: z.object({
      start_conversation_button_text: z.string(),
      allow_start_conversation: z.boolean(),
      prevent_multiple_conversations: z.boolean(),
      prevent_reply_to_closed_conversation: z.boolean(),
    }),
    prechat_form: z.object({
      enabled: z.boolean(),
      title: z.string().optional(),
      fields: z.array(z.object({
        key: z.string().min(1),
        type: z.enum(['text', 'email', 'number', 'checkbox', 'date', 'link', 'list', 'phone']),
        label: z.string().min(1, { message: t('globals.messages.required') }),
        placeholder: z.string().optional(),
        required: z.boolean(),
        enabled: z.boolean(),
        order: z.number().min(1),
        is_default: z.boolean(),
        custom_attribute_id: z.number().optional()
      }))
    })
  })
})
