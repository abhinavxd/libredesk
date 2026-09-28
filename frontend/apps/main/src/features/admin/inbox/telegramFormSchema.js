import * as z from 'zod'

export const createFormSchema = (t) =>
  z.object({
    name: z
      .string()
      .trim()
      .min(1, t('globals.messages.required', { name: t('globals.terms.name') }))
      .max(140, t('globals.messages.maxLength', { max: 140 })),
    enabled: z.boolean(),
    csat_enabled: z.boolean(),
    prompt_tags_on_reply: z.boolean(),
    reopen_window_hours: z.preprocess(
      (value) => (value === '' ? undefined : value),
      z.coerce
        .number({ invalid_type_error: t('admin.inbox.telegram.error.reopenWindow') })
        .int(t('admin.inbox.telegram.error.reopenWindow'))
        .min(0, t('admin.inbox.telegram.error.reopenWindow'))
        .max(2147483647, t('admin.inbox.telegram.error.reopenWindow'))
    ),
    config: z
      .object({
        greeting_message: z
          .string()
          .trim()
          .max(4096, t('globals.messages.maxLength', { max: 4096 }))
          .default(''),
        away_message: z
          .string()
          .trim()
          .max(4096, t('globals.messages.maxLength', { max: 4096 }))
          .default(''),
        csat_message: z
          .string()
          .trim()
          .max(4096, t('globals.messages.maxLength', { max: 4096 }))
          .default(''),
        business_hours_id: z.coerce.number().int().min(0).default(0),
        timezone: z.string().default('UTC'),
        bot_token: z
          .string()
          .trim()
          .min(1, t('globals.messages.required', { name: t('globals.terms.botToken') }))
      })
      .superRefine((config, context) => {
        if (config.away_message && !config.business_hours_id)
          context.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['business_hours_id'],
            message: t('admin.inbox.telegram.error.businessHours')
          })
      })
  })
