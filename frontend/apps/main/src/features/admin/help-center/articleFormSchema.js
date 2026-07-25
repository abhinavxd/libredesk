import * as z from 'zod'

export const createArticleFormSchema = (t) =>
  z.object({
    title: z.string().min(1, t('globals.messages.required')),
    content: z.string().min(1, t('globals.messages.required')),
    status: z.enum(['draft', 'published', 'archived']).default('draft'),
    collection_id: z.coerce.number().min(1, t('globals.messages.required')),
    sort_order: z.number().default(0),
    ai_enabled: z.boolean().default(false),
    locale: z.string().default('en'),
    excerpt: z.string().default(''),
    meta_title: z.string().default(''),
    meta_description: z.string().default(''),
    meta_image_url: z.string().default('')
  })
