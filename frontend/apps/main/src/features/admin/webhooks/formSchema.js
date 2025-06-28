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
    url: z
      .string({
        required_error: t('globals.messages.required')
      })
      .url({
        message: t('form.error.validUrl')
      }),
    events: z.array(z.string()).min(1, {
      message: t('globals.messages.required')
    }),
    secret: z.string().optional(),
    is_active: z.boolean().default(true).optional(),
    headers: z.string().optional()
  })
