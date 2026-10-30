import * as z from 'zod'

export const DEFAULT_CSAT_TEMPLATE_LANGUAGE = 'en_US'
export const DEFAULT_CSAT_TEMPLATE_BODY =
  'Your conversation has been resolved. How did we do? Tap below to rate your experience.'
export const DEFAULT_CSAT_TEMPLATE_BUTTON_TEXT = 'Rate us'

export const createFormSchema = (i18n) =>
  z.object({
    name: z.string().min(1, i18n._('{name} Required')),
    enabled: z.boolean().optional(),
    csat_enabled: z.boolean().optional(),
    prompt_tags_on_reply: z.boolean().optional(),
    reopen_window_hours: z.coerce.number().int().min(0).optional(),
    config: z.object({
      phone_number_id: z.string().min(1, i18n._('{name} Required')),
      waba_id: z.string().min(1, i18n._('{name} Required')),
      access_token: z.string().min(1, i18n._('{name} Required')),
      app_secret: z.string().min(1, i18n._('{name} Required')),
      webhook_verify_token: z.string().min(1, i18n._('{name} Required')),
      api_version: z.string().optional(),
      csat_template_language: z.string().optional(),
      csat_template_body: z.string().optional(),
      csat_template_button_text: z.string().optional()
    })
  })
