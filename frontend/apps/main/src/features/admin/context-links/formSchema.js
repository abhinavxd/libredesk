import * as z from 'zod'

export const createFormSchema = (t) =>
  z.object({
    name: z
      .string({
        required_error: t('globals.messages.required')
      })
      .min(1, {
        message: t('globals.messages.required')
      }),
    url_template: z
      .string({
        required_error: t('globals.messages.required')
      })
      .min(1, {
        message: t('globals.messages.required')
      }),
    secret: z.string().optional(),
    token_expiry_seconds: z.coerce.number().int().min(1).default(1200),
    is_active: z.boolean().optional().default(true)
  })
