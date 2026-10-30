import * as z from 'zod'

export const createFormSchema = (i18n) => z.object({
  name: z
    .string({
      required_error: i18n._('{name} Required'),
    })
    .min(3, {
      message: i18n._('Tag name should be at least 3 characters'),
    })
})
