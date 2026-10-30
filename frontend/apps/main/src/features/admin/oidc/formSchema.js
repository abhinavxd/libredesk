import * as z from 'zod'

export const createFormSchema = (i18n) => z.object({
  disabled: z.boolean().optional(),
  name: z
    .string({
      required_error: i18n._('{name} Required'),
    })
    .min(1, { message: i18n._('{name} Required') }),
  provider: z.string().optional(),
  provider_url: z
    .string({
      required_error: i18n._('{name} Required'),
    })
    .url({
      message: i18n._('Invalid URL'),
    }),
  logo_url: z.string().url({ message: i18n._('Invalid URL') }).optional().or(z.literal('')),
  client_id: z
    .string({
      required_error: i18n._('{name} Required'),
    })
    .min(1, { message: i18n._('{name} Required') }),
  client_secret: z
    .string({
      required_error: i18n._('{name} Required'),
    })
    .min(1, { message: i18n._('{name} Required') }),
  redirect_uri: z.string().readonly().optional(),
  enabled: z.boolean().optional().default(true),
})
