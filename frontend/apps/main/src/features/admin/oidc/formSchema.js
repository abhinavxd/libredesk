import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  disabled: z.boolean().optional(),
  name: z.string({
    required_error: t('globals.messages.required'),
  }),
  provider: z.string().optional(),
  provider_url: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .url({
      message: t('form.error.validUrl'),
    }),
  client_id: z.string({
    required_error: t('globals.messages.required'),
  }),
  client_secret: z.string({
    required_error: t('globals.messages.required'),
  }),
  redirect_uri: z.string().readonly().optional(),
  enabled: z.boolean().default(true).optional(),
})
