import * as z from 'zod'

export const createHelpCenterFormSchema = (t) => z.object({
  name: z.string().min(1, t('globals.messages.required')),
  slug: z
    .string()
    .min(1, t('globals.messages.required'))
    .regex(/^[a-z0-9-]+$/, 'Slug can only contain lowercase letters, numbers, and hyphens'),
  page_title: z.string().min(1, t('globals.messages.required')),
  default_locale: z.string().min(1, t('globals.messages.required')),
})
