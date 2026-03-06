import * as z from 'zod'

export const formSchema = z.object({
  endpoint_url: z
    .string()
    .refine((val) => val === '' || /^https?:\/\/.+/.test(val), {
      message: 'Must be a valid URL (http:// or https://)'
    })
    .optional()
    .default(''),
  model: z.string().optional().default(''),
  api_key: z.string().optional().default('')
})
