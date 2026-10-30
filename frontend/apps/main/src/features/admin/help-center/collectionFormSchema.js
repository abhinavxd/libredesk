import * as z from 'zod'

export const createCollectionFormSchema = (i18n) =>
  z.object({
    name: z.string().trim().min(1, i18n._('{name} Required')),
    description: z.string().optional(),
    icon: z.string().optional(),
    locale: z.string().default('en'),
    parent_id: z.coerce.number().nullable().optional(),
    is_published: z.boolean().default(true),
    sort_order: z.number().default(0)
  })
