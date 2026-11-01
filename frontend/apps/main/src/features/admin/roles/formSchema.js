import * as z from 'zod'

export const createFormSchema = (i18n) => z.object({
  name: z
    .string({
      required_error: i18n._('{name} Required'),
    })
    .min(2, {
      message: i18n._('Must be between {min} and {max} characters', { min: 2, max: 50 })
    })
    .max(50, {
      message: i18n._('Must be between {min} and {max} characters', { min: 2, max: 50 })
    }),

  description: z
    .string({
      required_error: i18n._('{name} Required'),
    })
    .min(2, {
      message: i18n._('Must be between {min} and {max} characters', { min: 2, max: 300 })
    })
    .max(300, {
      message: i18n._('Must be between {min} and {max} characters', { min: 2, max: 300 })
    }),
  permissions: z.array(z.string()).optional()
})
