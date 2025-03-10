import * as z from 'zod'

export const oidcLoginFormSchema = z.object({
  disabled: z.boolean().optional(),
  name: z.string({
    required_error: 'Name is required.'
  }),
  provider: z.string().optional(),
  provider_url: z
    .string({
      required_error: 'Provider URL is required.'
    })
    .url({
      message: 'Provider URL must be a valid URL.'
    }),
  client_id: z.string(),
  client_secret: z.string(),
  redirect_uri: z.string().readonly().optional(),
  enabled: z.boolean().default(true).optional(),
})
