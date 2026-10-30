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
    url_template: z
      .string({
        required_error: i18n._('{name} Required')
      })
      .min(1, {
        message: i18n._('{name} Required')
      }),
    secret: z.string().optional(),
    token_expiry_seconds: z.coerce.number().int().min(1).default(1200),
    is_active: z.boolean().optional().default(true)
  })
