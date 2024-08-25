import * as z from 'zod'

export const formSchema = z.object({
  name: z
    .string({
      required_error: 'Tag name is required.'
    })
    .min(1, {
      message: 'Tag name must be at least 1 character.'
    })
})
