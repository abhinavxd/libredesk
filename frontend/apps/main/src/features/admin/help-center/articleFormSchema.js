import * as z from 'zod'

// An untouched editor serializes to '<p></p>', which passes a length check.
const hasArticleContent = (html) =>
  html.replace(/<[^>]*>/g, '').replace(/&nbsp;/g, ' ').trim().length > 0 ||
  /<(img|table|iframe|video|hr|blockquote|details)\b/i.test(html)

export const createArticleFormSchema = (t) =>
  z.object({
    title: z.string().min(1, t('globals.messages.required')),
    content: z.string().refine(hasArticleContent, t('globals.messages.required')),
    status: z.enum(['draft', 'published']).default('draft'),
    collection_id: z.coerce.number().min(1, t('globals.messages.required')),
    sort_order: z.number().default(0),
    ai_enabled: z.boolean().default(false),
    locale: z.string().default('en'),
    author_id: z.string().optional(),
    excerpt: z.string().default(''),
    meta_title: z.string().default(''),
    meta_description: z.string().default(''),
    meta_image_url: z.string().default('')
  })
