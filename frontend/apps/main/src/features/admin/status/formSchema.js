import * as z from 'zod'
export const createFormSchema = (i18n) => z.object({
  name: z
    .string({
      required_error: i18n._('{name} Required'),
    })
    .min(1, {
      message: i18n._('Must be between {min} and {max} characters', {
        min: 1,
        max: 25,
      })
    })
    .max(25, {
      message: i18n._('Must be between {min} and {max} characters', {
        min: 1,
        max: 25,
      })
    }),
  category: z.enum(['open', 'waiting', 'resolved'], {
    required_error: i18n._('{name} Required'),
  })
})
