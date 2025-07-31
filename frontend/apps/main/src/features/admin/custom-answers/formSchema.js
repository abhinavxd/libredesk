import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  question: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(10, {
      message: t('form.error.minmax', {
        min: 10,
        max: 500,
      })
    })
    .max(500, {
      message: t('form.error.minmax', {
        min: 10,
        max: 500,
      })
    }),

  answer: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(10, {
      message: t('form.error.minmax', {
        min: 10,
        max: 2000,
      })
    })
    .max(2000, {
      message: t('form.error.minmax', {
        min: 10,
        max: 2000,
      })
    }),

  enabled: z.boolean().optional().default(true),
})