import * as z from 'zod'

export const createCollectionFormSchema = (t) => z.object({
  name: z.string().min(1, t('globals.messages.required')),
  description: z.string().optional(),
  parent_id: z.number().nullable().optional(),
  is_published: z.boolean().default(true),
  sort_order: z.number().default(0),
})
