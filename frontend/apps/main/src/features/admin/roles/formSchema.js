import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  name: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(2, {
      message: t('form.error.minmax', { min: 2, max: 50 })
    })
    .max(50, {
      message: t('form.error.minmax', { min: 2, max: 50 })
    }),

  description: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(2, {
      message: t('form.error.minmax', { min: 2, max: 300 })
    })
    .max(300, {
      message: t('form.error.minmax', { min: 2, max: 300 })
    }),
  permissions: z.array(z.string()).optional()
})
