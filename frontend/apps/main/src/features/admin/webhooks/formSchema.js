import * as z from 'zod'

export const createFormSchema = (i18n) =>
  z.object({
    name: z
      .string({
        required_error: i18n._('{name} Required')
      })
      .min(1, {
        message: i18n._('{name} Required')
      }),
    url: z
      .string({
        required_error: i18n._('{name} Required')
      })
      .url({
        message: i18n._('Invalid URL')
      }),
    events: z.array(z.string()).min(1, {
      message: i18n._('{name} Required')
    }),
    secret: z.string().optional(),
    is_active: z.boolean().optional().default(true),
    headers: z.string().optional()
  })
