import * as z from 'zod'
export const createFormSchema = (t) => z.object({
  name: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(1, {
      message: t('form.error.minmax', {
        min: 1,
        max: 25,
      })
    })
    .max(25, {
      message: t('form.error.minmax', {
        min: 1,
        max: 25,
      })
    })
})
