import { z } from 'zod'
import { isGoDuration } from '@shared-ui/utils/string'

const hexColorRegex = /^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/
const hexColor = (i18n) => z.string().regex(hexColorRegex, { message: i18n._('Invalid color') })
const optionalHexColor = (i18n) => hexColor(i18n).optional().or(z.literal(''))
const optionalUrl = (i18n) =>
  z
    .string()
    .url({ message: i18n._('Invalid URL') })
    .optional()
    .or(z.literal(''))
const rangeNumber = (i18n, min, max) => {
  const msg = i18n._('Must be between {min} and {max}', { min, max })
  return z.coerce
    .number({ invalid_type_error: msg })
    .min(min, { message: msg })
    .max(max, { message: msg })
}
const rangeInteger = (i18n, min, max) => {
  const msg = i18n._('Must be between {min} and {max}', { min, max })
  return rangeNumber(i18n, min, max).int({ message: msg })
}
const spacingNumber = (i18n) => rangeNumber(i18n, 0, 200)
const goDurationSeconds = (value) => {
  const match = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/.exec(value)
  if (!match) return 0
  return Number(match[1] || 0) * 3600 + Number(match[2] || 0) * 60 + Number(match[3] || 0)
}
const branding = (i18n) =>
  z.object({
    colors: z.object({
      primary: hexColor(i18n)
    }),
    logo_url: optionalUrl(i18n),
    launcher: z.object({
      logo_url: optionalUrl(i18n),
      color: hexColor(i18n)
    }),
    home_screen: z.object({
      header_text_color: z.enum(['black', 'white']),
      background: z
        .object({
          type: z.enum(['solid', 'gradient', 'image']),
          color: optionalHexColor(i18n),
          gradient_start: optionalHexColor(i18n),
          gradient_end: optionalHexColor(i18n),
          image_url: optionalUrl(i18n)
        })
        .superRefine((bg, ctx) => {
          // An empty solid color falls back to the page background, gradients and images render nothing.
          if (bg.type === 'gradient') {
            if (!bg.gradient_start)
              ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ['gradient_start'],
                message: i18n._('{name} Required')
              })
            if (!bg.gradient_end)
              ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ['gradient_end'],
                message: i18n._('{name} Required')
              })
          } else if (bg.type === 'image' && !bg.image_url) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ['image_url'],
              message: i18n._('{name} Required')
            })
          }
        }),
      fade_background: z.boolean()
    })
  })

const quickReplies = (i18n) =>
  z
    .string()
    .refine(
      (value) =>
        value
          .split('\n')
          .map((reply) => reply.trim())
          .filter(Boolean).length <= 6,
      {
        message: i18n._('Add up to 6 quick replies.')
      }
    )
    .refine((value) => value.split('\n').every((reply) => reply.trim().length <= 120), {
      message: i18n._('Must be at most {max} characters', { max: 120 })
    })
const prechatField = (i18n) =>
  z.object({
    key: z.string().min(1),
    type: z.enum(['text', 'email', 'number', 'checkbox', 'date', 'link', 'list', 'phone']),
    label: z.string().min(1, { message: i18n._('{name} Required') }),
    placeholder: z.string().optional(),
    required: z.boolean(),
    enabled: z.boolean(),
    order: z.number().min(1),
    is_default: z.boolean(),
    custom_attribute_id: z.number().optional()
  })
const prechatAudience = (i18n) =>
  z.object({
    enabled: z.boolean(),
    title: z.string().optional(),
    fields: z.array(prechatField(i18n))
  })

export const defaultWidgetHelp = () => ({
  help_center_id: 0,
  visitors: { tab: true },
  users: { tab: true },
  featured_ids: []
})
export const createWidgetConditionsSchema = (i18n) =>
  z.object({
    logical_op: z.enum(['AND', 'OR']),
    rules: z
      .array(
        z.object({
          field: z.string().min(1, i18n._('Required')),
          field_type: z.literal('contact_custom_attribute'),
          operator: z.string().min(1, i18n._('Required')),
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
  return {
    ...audienceConfig,
    quick_replies: Array.isArray(replies) ? replies.join('\n') : (replies ?? ''),
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
export const createCampaignSchema = (i18n) =>
  z
    .object({
      id: z.string().uuid(),
      name: z.string().trim().min(1, i18n._('Required')).max(128),
      enabled: z.boolean(),
      message: z.string().trim().min(1, i18n._('Message cannot be empty')).max(10000),
      sender_id: z.number().int().min(0),
      team_id: z.number().int().min(0),
      audience: z.enum(['all', 'visitors', 'users']),
      include_urls: z
        .array(z.string().max(2048))
        .transform((items) => items.map((item) => item.trim()).filter(Boolean))
        .refine((items) => items.length <= 20, i18n._('Add up to 20 page patterns.')),
      exclude_urls: z
        .array(z.string().max(2048))
        .transform((items) => items.map((item) => item.trim()).filter(Boolean))
        .refine((items) => items.length <= 20, i18n._('Add up to 20 page patterns.')),
      conditions: createWidgetConditionsSchema(i18n),
      event: z.string().max(128),
      delay_seconds: rangeInteger(i18n, 0, 86400),
      business_hours_id: z.number().int().min(0),
      business_hours: z.enum(['any', 'inside', 'outside']),
      desktop: z.boolean(),
      mobile: z.boolean(),
      repeat: z.enum(['once', 'session', 'interval']),
      repeat_hours: rangeInteger(i18n, 1, 8760)
    })
    .superRefine((campaign, ctx) => {
      if (!campaign.desktop && !campaign.mobile) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['desktop'],
          message: i18n._('Select at least one device. This proactive message cannot appear until desktop or mobile is enabled.')
        })
      }
      if (campaign.business_hours !== 'any' && campaign.business_hours_id === 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['business_hours_id'],
          message: i18n._('Required')
        })
      }
    })
const helpAudience = z.object({ tab: z.boolean() })

export const createFormSchema = (i18n) =>
  z.object({
    name: z.string().min(1, { message: i18n._('{name} Required') }),
    enabled: z.boolean(),
    csat_enabled: z.boolean(),
    prompt_tags_on_reply: z.boolean(),
    secret: z.string().nullable().optional(),
    linked_email_inbox_id: z.number().nullable().optional(),
    config: z.object({
      campaigns: z.array(createCampaignSchema(i18n)).max(50).default([]),
      campaign_cooldown: z
        .string()
        .min(1, { message: i18n._('{name} Required') })
        .refine(isGoDuration, { message: i18n._('Invalid duration format. Please use a valid format (e.g. 30m, 1h, 48h).') })
        .default('24h'),
      help: z
        .object({
          help_center_id: z.number().int().min(0),
          visitors: helpAudience,
          users: helpAudience,
          featured_ids: z.array(z.number().int().positive()).max(10)
        })
        .default(defaultWidgetHelp),
      brand_name: z.string().min(1, { message: i18n._('{name} Required') }),
      website_url: optionalUrl(i18n),
      theme: z.enum(['system', 'light', 'dark']),
      show_powered_by: z.boolean(),
      language: z.string().min(1, { message: i18n._('{name} Required') }),
      fallback_language: z.string().optional(),
      branding: z.object({
        light: branding(i18n),
        dark: branding(i18n)
      }),
      launcher: z.object({
        position: z.enum(['left', 'right']),
        icon_scale: rangeInteger(i18n, 40, 100),
        spacing: z.object({
          side: spacingNumber(i18n),
          bottom: spacingNumber(i18n)
        })
      }),
      greeting_message: z.string().optional(),
      introduction_message: z.string().optional(),
      chat_introduction: z.string(),
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
              message: i18n._('{name} Required')
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
        .min(1, { message: i18n._('{name} Required') })
        .refine(isGoDuration, { message: i18n._('Invalid duration format. Please use a valid format (e.g. 30m, 1h, 48h).') })
        .refine((value) => goDurationSeconds(value) >= 3600, {
          message: i18n._('{name} must be at least {min}.', {
            name: i18n._('Authenticated user session duration'),
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
          image_url: optionalUrl(i18n),
          url: optionalUrl(i18n),
          text: z.string().optional().or(z.literal(''))
        })
      ),
      visitors: z.object({
        start_conversation_button_text: z.string(),
        allow_start_conversation: z.boolean(),
        prevent_multiple_conversations: z.boolean(),
        prevent_reply_to_closed_conversation: z.boolean(),
        quick_replies: quickReplies(i18n).optional(),
        direct_to_conversation: z.boolean().optional()
      }),
      users: z.object({
        start_conversation_button_text: z.string(),
        allow_start_conversation: z.boolean(),
        prevent_multiple_conversations: z.boolean(),
        prevent_reply_to_closed_conversation: z.boolean(),
        quick_replies: quickReplies(i18n).optional(),
        direct_to_conversation: z.boolean().optional()
      }),
      prechat_form: z.object({
        enabled: z.boolean(),
        handoff_only: z.boolean().default(false),
        title: z.string().optional(),
        fields: z.array(prechatField(i18n)),
        visitors: prechatAudience(i18n).optional(),
        users: prechatAudience(i18n).optional()
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
            message: i18n._('{name} Required')
          })
        } else if (!isGoDuration(continuity[field])) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['config', 'continuity', field],
            message: i18n._('Invalid duration format. Please use a valid format (e.g. 30m, 1h, 48h).')
          })
        }
      }
      const max = Number(continuity.max_messages_per_email)
      if (!Number.isInteger(max) || max < 1 || max > 100) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['config', 'continuity', 'max_messages_per_email'],
          message: i18n._('Must be between {min} and {max}', { min: 1, max: 100 })
        })
      }
    })
