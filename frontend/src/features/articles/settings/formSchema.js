import * as z from 'zod'

export const createFormSchema = (t) => {
  return z.object({
    site_title: z
      .string({
        required_error: t('globals.messages.required')
      })
      .min(3, {
        message: t('form.error.minmax', {
          min: 3,
          max: 140
        })
      })
      .max(140, {
        message: t('form.error.minmax', {
          min: 3,
          max: 140
        })
      }),

    custom_script: z
      .string()
      .max(100_000, {
        message: t('form.error.max', { max: 100_000 })
      })
      .optional(),

    site_description: z
      .string()
      .max(255, {
        message: t('form.error.minmax', { min: 1, max: 255 })
      })
      .optional(),

    primary_font: z.string({
      required_error: t('globals.messages.required')
    }),
    logo_url: z
      .string()
      .url({ message: t('form.error.validUrl') })
      .optional()
      .or(z.literal(''))
  })
}
