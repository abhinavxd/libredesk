import * as z from 'zod'

export const createArticleFormSchema = (t) => z.object({
  title: z.string().min(1, t('globals.messages.required')),
  content: z.string().min(1, t('globals.messages.required')),
  status: z.enum(['draft', 'published']).default('draft'),
  collection_id: z.number().min(1, t('globals.messages.required')),
  sort_order: z.number().default(0),
  ai_enabled: z.boolean().default(false),
})
