import * as z from 'zod'

export const createFormSchema = (t) =>
  z.object({
    title: z
      .string({
        required_error: t('globals.messages.required')
      })
      .min(1, {
        message: t('globals.messages.empty', { name: t('globals.terms.title') })
      })
      .max(255, {
        message: t('form.error.minmax', { min: 1, max: 255 })
      }),

    section_id: z
      .string({
        required_error: t('globals.messages.required')
      })
      .or(z.number())
      .transform((val) => (typeof val === 'string' ? parseInt(val, 10) : val))
      .refine((val) => !isNaN(val) && val > 0, {
        message: t('globals.messages.invalid', { name: t('globals.terms.section') })
      }),

    content: z
      .string({
        required_error: t('globals.messages.required')
      })
      .min(1, {
        message: t('globals.messages.empty', { name: t('globals.terms.content') })
      })
  })
