import * as z from 'zod'

export const createHelpCenterFormSchema = (t) => {
  const linkArray = z
    .array(
      z.object({
        label: z.string().min(1, t('globals.messages.required')),
        url: z.string().min(1, t('globals.messages.required'))
      })
    )
    .optional()

  return z.object({
    name: z.string().min(1, t('globals.messages.required')),
    slug: z
      .string()
      .min(1, t('globals.messages.required'))
      .regex(/^[a-z0-9-]+$/, t('helpCenter.invalidSlug')),
    page_title: z.string().min(1, t('globals.messages.required')),
    header_text: z.string().optional(),
    logo_url: z.string().optional(),
    color: z.string().optional(),
    nav_links: linkArray,
    custom_css: z.string().optional(),
    custom_js: z.string().optional(),
    default_locale: z.string().min(1, t('globals.messages.required')).default('en'),
    allowed_locales: z
      .array(z.string().min(1, t('globals.messages.required')))
      .min(1, t('globals.messages.required'))
      .default(['en']),
    theme: z
      .object({
        favicon: z.string().optional(),
        tagline: z.string().optional(),
        header: z
          .object({
            background_type: z.string().optional(),
            background_color: z.string().optional(),
            gradient_from: z.string().optional(),
            gradient_to: z.string().optional(),
            text_color: z.string().optional()
          })
          .optional(),
        footer: z
          .object({
            background_color: z.string().optional(),
            text_color: z.string().optional(),
            tagline: z.string().optional()
          })
          .optional(),
        footer_links: linkArray,
        social_links: z
          .array(
            z.object({
              platform: z.string().min(1, t('globals.messages.required')),
              url: z.string().min(1, t('globals.messages.required'))
            })
          )
          .optional(),
        article: z
          .object({
            hide_toc: z.boolean().optional(),
            hide_related: z.boolean().optional()
          })
          .optional()
      })
      .optional()
  })
}
