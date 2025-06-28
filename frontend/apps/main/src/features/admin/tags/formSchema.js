import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  name: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(3, {
      message: t('admin.conversationTags.name.valid'),
    })
})
