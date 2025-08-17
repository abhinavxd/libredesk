import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  type: z.literal('snippet'), // Fixed value for now, expandable later

  content: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(10, {
      message: t('form.error.minmax', {
        min: 10,
        max: 5000,
      })
    })
    .max(5000, {
      message: t('form.error.minmax', {
        min: 10,
        max: 5000,
      })
    }),

  enabled: z.boolean().optional().default(true),
})