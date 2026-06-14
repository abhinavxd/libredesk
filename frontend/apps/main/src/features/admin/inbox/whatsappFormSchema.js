import * as z from 'zod'

export const createFormSchema = (t) =>
  z.object({
    name: z.string().min(1, t('globals.messages.required')),
    enabled: z.boolean().optional(),
    csat_enabled: z.boolean().optional(),
    prompt_tags_on_reply: z.boolean().optional(),
    config: z.object({
      phone_number_id: z.string().min(1, t('globals.messages.required')),
      waba_id: z.string().min(1, t('globals.messages.required')),
      access_token: z.string().min(1, t('globals.messages.required')),
      app_secret: z.string().min(1, t('globals.messages.required')),
      webhook_verify_token: z.string().min(1, t('globals.messages.required')),
      api_version: z.string().optional(),
      reopen_window_hours: z.coerce.number().int().min(0).optional()
    })
  })
