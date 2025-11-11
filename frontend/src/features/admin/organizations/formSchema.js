import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  name: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(2, {
      message: t('form.error.minmax', {
        min: 2,
        max: 100,
      })
    })
    .max(100, {
      message: t('form.error.minmax', {
        min: 2,
        max: 100,
      })
    }),
  website: z
    .string()
    .url({
      message: t('globals.messages.invalidURL'),
    })
    .optional()
    .or(z.literal(''))
    .nullable(),
  email_domain: z
    .string()
    .regex(/^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]?\.([a-zA-Z]{2,}\.?)+$/, {
      message: t('globals.messages.invalidDomain'),
    })
    .optional()
    .or(z.literal(''))
    .nullable(),
  phone: z
    .string()
    .optional()
    .or(z.literal(''))
    .nullable(),
})
