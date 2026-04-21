import { z } from 'zod'
import { isGoDuration } from '@shared-ui/utils/string'

const hexColorRegex = /^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/
const hexColor = (t) => z.string().regex(hexColorRegex, { message: t('validation.invalidColor') })
const optionalHexColor = (t) => hexColor(t).optional().or(z.literal(''))
const optionalUrl = (t) => z.string().url({ message: t('validation.invalidUrl') }).optional().or(z.literal(''))
const spacingNumber = (t) => {
  const msg = t('validation.minmaxNumber', { min: 0, max: 200 })
  return z.coerce.number({ invalid_type_error: msg }).min(0, { message: msg }).max(200, { message: msg })
}

export const createFormSchema = (t) => z.object({
  name: z.string().min(1, { message: t('globals.messages.required') }),
  enabled: z.boolean(),
  csat_enabled: z.boolean(),
  prompt_tags_on_reply: z.boolean(),
  secret: z.string().nullable().optional(),
  linked_email_inbox_id: z.number().nullable().optional(),
  config: z.object({
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
      }),
      fade_background: z.boolean(),
    }),
    features: z.object({
      file_upload: z.boolean(),
      emoji: z.boolean(),
    }),
    continuity: z.object({
      offline_threshold: z.string().min(1, { message: t('globals.messages.required') }).refine(isGoDuration, { message: t('validation.invalidDuration') }),
      max_messages_per_email: z.number().min(1).max(100),
      min_email_interval: z.string().min(1, { message: t('globals.messages.required') }).refine(isGoDuration, { message: t('validation.invalidDuration') }),
    }).optional(),
    session_duration: z.string().min(1, { message: t('globals.messages.required') }).refine(isGoDuration, { message: t('validation.invalidDuration') }),
    direct_to_conversation: z.boolean().default(false),
    trusted_domains: z.string().optional(),
    blocked_ips: z.string().optional(),
    home_apps: z.array(z.object({
      type: z.enum(['announcement', 'external_link']),
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
        type: z.enum(['text', 'email', 'number', 'checkbox', 'date', 'link', 'list']),
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
