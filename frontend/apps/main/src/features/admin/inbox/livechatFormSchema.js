import { z } from 'zod'
import { isGoDuration } from '@shared-ui/utils/string'

const hexColorRegex = /^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/
const hexColor = (t) => z.string().regex(hexColorRegex, { message: t('validation.invalidColor') })
const optionalHexColor = (t) => hexColor(t).optional().or(z.literal(''))
const optionalUrl = (t) =>
  z
    .string()
    .url({ message: t('validation.invalidUrl') })
    .optional()
    .or(z.literal(''))
const rangeNumber = (t, min, max) => {
  const msg = t('validation.minmaxNumber', { min, max })
  return z.coerce
    .number({ invalid_type_error: msg })
    .min(min, { message: msg })
    .max(max, { message: msg })
}
const rangeInteger = (t, min, max) => {
  const msg = t('validation.minmaxNumber', { min, max })
  return rangeNumber(t, min, max).int({ message: msg })
}
const spacingNumber = (t) => rangeNumber(t, 0, 200)
const goDurationSeconds = (value) => {
  const match = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/.exec(value)
  if (!match) return 0
  return Number(match[1] || 0) * 3600 + Number(match[2] || 0) * 60 + Number(match[3] || 0)
}
const branding = (t) =>
  z.object({
    colors: z.object({
      primary: hexColor(t)
    }),
    logo_url: optionalUrl(t),
    launcher: z.object({
      logo_url: optionalUrl(t),
      color: hexColor(t)
    }),
    home_screen: z.object({
      header_text_color: z.enum(['black', 'white']),
      background: z
        .object({
          type: z.enum(['solid', 'gradient', 'image']),
          color: optionalHexColor(t),
          gradient_start: optionalHexColor(t),
          gradient_end: optionalHexColor(t),
          image_url: optionalUrl(t)
        })
        .superRefine((bg, ctx) => {
          // An empty solid color falls back to the page background, gradients and images render nothing.
          if (bg.type === 'gradient') {
            if (!bg.gradient_start)
              ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ['gradient_start'],
                message: t('globals.messages.required')
              })
            if (!bg.gradient_end)
              ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ['gradient_end'],
                message: t('globals.messages.required')
              })
          } else if (bg.type === 'image' && !bg.image_url) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['image_url'],
              message: t('globals.messages.required')
            })
          }
        }),
      fade_background: z.boolean()
    })
  })

const quickReplies = (t) =>
  z
    .string()
    .refine(
      (value) =>
        value
          .split('\n')
          .map((reply) => reply.trim())
          .filter(Boolean).length <= 6,
      {
        message: t('admin.inbox.livechat.quickReplies.limit')
      }
    )
    .refine((value) => value.split('\n').every((reply) => reply.trim().length <= 120), {
      message: t('globals.messages.maxLength', { max: 120 })
    })
const prechatField = (t) =>
  z.object({
    key: z.string().min(1),
    type: z.enum(['text', 'email', 'number', 'checkbox', 'date', 'link', 'list', 'phone']),
    label: z.string().min(1, { message: t('globals.messages.required') }),
    placeholder: z.string().optional(),
    required: z.boolean(),
    enabled: z.boolean(),
    order: z.number().min(1),
    is_default: z.boolean(),
    custom_attribute_id: z.number().optional()
  })
const prechatAudience = (t) =>
  z.object({
    enabled: z.boolean(),
    title: z.string().optional(),
    fields: z.array(prechatField(t))
  })

export const defaultWidgetHelp = () => ({
  help_center_id: 0,
  visitors: { tab: true },
  users: { tab: true },
  featured_ids: []
})
export const createWidgetConditionsSchema = (t) =>
  z.object({
    logical_op: z.enum(['AND', 'OR']),
    rules: z
      .array(
        z.object({
          field: z.string().min(1, t('globals.terms.required')),
          field_type: z.literal('contact_custom_attribute'),
          operator: z.string().min(1, t('globals.terms.required')),
          value: z.string(),
          case_sensitive_match: z.boolean()
        })
      )
      .max(20)
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
  id: newCampaignId(),
  name: '',
  enabled: false,
  message: '',
  sender_id: 0,
  team_id: 0,
  audience: 'all',
  include_urls: [],
  exclude_urls: [],
  conditions: { logical_op: 'AND', rules: [] },
  event: '',
  delay_seconds: 10,
  business_hours_id: 0,
  business_hours: 'any',
  desktop: true,
  mobile: true,
  repeat: 'once',
  repeat_hours: 24
})
export const normalizeAudienceConfig = (config, audience) => {
  const audienceConfig = { ...config?.[audience] }
  const replies = audienceConfig.quick_replies
  const legacyReplies = Array.isArray(config?.quick_replies)
    ? config.quick_replies.join('\n')
    : (config?.quick_replies ?? '')
  return {
    ...audienceConfig,
    quick_replies: Array.isArray(replies) ? replies.join('\n') : (replies ?? legacyReplies),
    direct_to_conversation:
      audienceConfig.direct_to_conversation ?? config?.direct_to_conversation ?? false
  }
}
export const normalizePrechatConfig = (config = {}) => {
  const normalizeAudience = (value) => {
    const audience = value && typeof value === 'object' && !Array.isArray(value) ? value : {}
    return {
      enabled: typeof value === 'boolean' ? value : (audience.enabled ?? true),
      title: audience.title ?? config.title ?? '',
      fields: (audience.fields ?? config.fields ?? []).map((field) => ({ ...field }))
    }
  }
  return {
    ...config,
    enabled: config.enabled ?? false,
    fields: config.fields ?? [],
    handoff_only: config.handoff_only ?? false,
    visitors: normalizeAudience(config.visitors),
    users: normalizeAudience(config.users)
  }
}
export const createCampaignSchema = (t) =>
  z
    .object({
      id: z.string().uuid(),
      name: z.string().trim().min(1, t('globals.terms.required')).max(128),
      enabled: z.boolean(),
      message: z.string().trim().min(1, t('validation.messageCannotBeEmpty')).max(10000),
      sender_id: z.number().int().min(0),
      team_id: z.number().int().min(0),
      audience: z.enum(['all', 'visitors', 'users']),
      include_urls: z
        .array(z.string().max(2048))
        .transform((items) => items.map((item) => item.trim()).filter(Boolean))
        .refine((items) => items.length <= 20, t('widget.campaignUrlLimit')),
      exclude_urls: z
        .array(z.string().max(2048))
        .transform((items) => items.map((item) => item.trim()).filter(Boolean))
        .refine((items) => items.length <= 20, t('widget.campaignUrlLimit')),
      conditions: createWidgetConditionsSchema(t),
      event: z.string().max(128),
      delay_seconds: rangeInteger(t, 0, 86400),
      business_hours_id: z.number().int().min(0),
      business_hours: z.enum(['any', 'inside', 'outside']),
      desktop: z.boolean(),
      mobile: z.boolean(),
      repeat: z.enum(['once', 'session', 'interval']),
      repeat_hours: rangeInteger(t, 1, 8760)
    })
    .superRefine((campaign, ctx) => {
      if (!campaign.desktop && !campaign.mobile) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['desktop'],
          message: t('widget.campaignDeviceRequired')
        })
      }
      if (campaign.business_hours !== 'any' && campaign.business_hours_id === 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['business_hours_id'],
          message: t('globals.terms.required')
        })
      }
    })
const helpAudience = z.object({ tab: z.boolean() })

export const createFormSchema = (t) =>
  z.object({
    name: z.string().min(1, { message: t('globals.messages.required') }),
    enabled: z.boolean(),
    csat_enabled: z.boolean(),
    prompt_tags_on_reply: z.boolean(),
    secret: z.string().nullable().optional(),
    linked_email_inbox_id: z.number().nullable().optional(),
    config: z.object({
      campaigns: z.array(createCampaignSchema(t)).max(50).default([]),
      campaign_cooldown: z
        .string()
        .min(1, { message: t('globals.messages.required') })
        .refine(isGoDuration, { message: t('validation.invalidDuration') })
        .default('24h'),
      help: z
        .object({
          help_center_id: z.number().int().min(0),
          visitors: helpAudience,
          users: helpAudience,
          featured_ids: z.array(z.number().int().positive()).max(10)
        })
        .default(defaultWidgetHelp),
      brand_name: z.string().min(1, { message: t('globals.messages.required') }),
      website_url: optionalUrl(t),
      theme: z.enum(['system', 'light', 'dark']),
      show_powered_by: z.boolean(),
      language: z.string().min(1, { message: t('globals.messages.required') }),
      fallback_language: z.string().optional(),
      branding: z.object({
        light: branding(t),
        dark: branding(t)
      }),
      launcher: z.object({
        position: z.enum(['left', 'right']),
        icon_scale: rangeInteger(t, 40, 100),
        spacing: z.object({
          side: spacingNumber(t),
          bottom: spacingNumber(t)
        })
      }),
      greeting_message: z.string().optional(),
      introduction_message: z.string().optional(),
      chat_introduction: z.string(),
      quick_replies: quickReplies(t).optional(),
      show_office_hours_in_chat: z.boolean(),
      show_office_hours_after_assignment: z.boolean(),
      chat_reply_expectation_message: z.string().optional(),
      notice_banner: z
        .object({
          enabled: z.boolean(),
          text: z.string().optional()
        })
        .superRefine((nb, ctx) => {
          if (nb.enabled && !nb.text?.trim()) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['text'],
              message: t('globals.messages.required')
            })
          }
        }),
      features: z.object({
        file_upload: z.boolean(),
        emoji: z.boolean(),
        transcript: z.boolean().default(false)
      }),
      continuity: z
        .object({
          offline_threshold: z.string().optional().or(z.literal('')),
          max_messages_per_email: z.coerce.number().optional(),
          min_email_interval: z.string().optional().or(z.literal(''))
        })
        .optional(),
      session_duration: z
        .string()
        .min(1, { message: t('globals.messages.required') })
        .refine(isGoDuration, { message: t('validation.invalidDuration') })
        .refine((value) => goDurationSeconds(value) >= 3600, {
          message: t('validation.minDuration', {
            name: t('admin.inbox.livechat.sessionDuration.label'),
            min: '1h'
          })
        }),
      direct_to_conversation: z.boolean().default(false),
      trusted_domains: z.string().optional(),
      blocked_ips: z.string().optional(),
      home_apps: z.array(
        z.object({
          type: z.enum(['announcement', 'external_link', 'help']),
          title: z.string().optional().or(z.literal('')),
          description: z.string().optional().or(z.literal('')),
          image_url: optionalUrl(t),
          url: optionalUrl(t),
          text: z.string().optional().or(z.literal(''))
        })
      ),
      visitors: z.object({
        start_conversation_button_text: z.string(),
        allow_start_conversation: z.boolean(),
        prevent_multiple_conversations: z.boolean(),
        prevent_reply_to_closed_conversation: z.boolean(),
        quick_replies: quickReplies(t).optional(),
        direct_to_conversation: z.boolean().optional()
      }),
      users: z.object({
        start_conversation_button_text: z.string(),
        allow_start_conversation: z.boolean(),
        prevent_multiple_conversations: z.boolean(),
        prevent_reply_to_closed_conversation: z.boolean(),
        quick_replies: quickReplies(t).optional(),
        direct_to_conversation: z.boolean().optional()
      }),
      prechat_form: z.object({
        enabled: z.boolean(),
        handoff_only: z.boolean().default(false),
        title: z.string().optional(),
        fields: z.array(prechatField(t)),
        visitors: prechatAudience(t).optional(),
        users: prechatAudience(t).optional()
      })
    })
  })
    .superRefine((values, ctx) => {
      if (!values.linked_email_inbox_id) return
      const continuity = values.config?.continuity ?? {}
      for (const field of ['offline_threshold', 'min_email_interval']) {
        if (!continuity[field]) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['config', 'continuity', field],
            message: t('globals.messages.required')
          })
        } else if (!isGoDuration(continuity[field])) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['config', 'continuity', field],
            message: t('validation.invalidDuration')
          })
        }
      }
      const max = Number(continuity.max_messages_per_email)
      if (!Number.isInteger(max) || max < 1 || max > 100) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['config', 'continuity', 'max_messages_per_email'],
          message: t('validation.minmaxNumber', { min: 1, max: 100 })
        })
      }
    })
