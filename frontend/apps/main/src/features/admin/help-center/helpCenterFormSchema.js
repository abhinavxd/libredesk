import * as z from 'zod'

// Mirrors the length caps on the backend.
const MAX_NAME = 200
const MAX_PAGE_TITLE = 200
const MAX_META_DESCRIPTION = 500

// The backend trims then counts runes, so trim here too and emoji must not cost two characters.
const withinLength = (max) => (v) => Array.from((v || '').trim()).length <= max

const localeRe = /^[a-z]{2,3}(-[A-Za-z0-9]{2,8})?$/

// Mirrors assetURLRe on the backend, which also discards protocol-relative URLs.
const urlRe = /^(?:https?:\/\/[^"'()\s\\<>;{}]+|\/(?!\/)[^"'()\s\\<>;{}]*)$/

// Mirrors hexColorRe on the backend, which discards anything else on save.
const hexColorRe = /^#(?:[0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/

export const createHelpCenterBasicsSchema = (t) =>
  createBaseSchema(t).pick({ name: true, slug: true, page_title: true, template: true })

export const createHelpCenterFormSchema = (t) =>
  createBaseSchema(t)
    .refine((v) => v.allowed_locales.includes(v.default_locale), {
      message: t('helpCenter.defaultLocaleNotAllowed'),
      path: ['default_locale']
    })
    // The banner is dropped on save when its text is blank, taking any link with it.
    .refine(
      (v) => {
        const a = v.theme?.announcement
        if (!a || a.text?.trim()) return true
        return !a.link_label?.trim() && !a.link_url?.trim()
      },
      {
        message: t('helpCenter.announcementTextRequired'),
        path: ['theme', 'announcement', 'text']
      }
    )

const createBaseSchema = (t) => {
  const optionalURL = z
    .string()
    .refine((v) => !v || urlRe.test(v), t('helpCenter.invalidURL'))
    .optional()

  const optionalHexColor = z
    .string()
    .refine((v) => !v || hexColorRe.test(v), t('validation.invalidColor'))
    .optional()

  const linkArray = z
    .array(
      z.object({
        label: z.string().min(1, t('globals.messages.required')),
        url: z
          .string()
          .min(1, t('globals.messages.required'))
          .regex(urlRe, t('helpCenter.invalidURL'))
      })
    )
    .optional()

  return z.object({
    name: z
      .string()
      .min(1, t('globals.messages.required'))
      .refine(withinLength(MAX_NAME), t('globals.messages.maxLength', { max: MAX_NAME })),
    slug: z
      .string()
      .min(1, t('globals.messages.required'))
      .max(200, t('helpCenter.invalidSlug'))
      .regex(/^[a-z0-9_-]+$/, t('helpCenter.invalidSlug')),
    page_title: z
      .string()
      .min(1, t('globals.messages.required'))
      .refine(
        withinLength(MAX_PAGE_TITLE),
        t('globals.messages.maxLength', { max: MAX_PAGE_TITLE })
      ),
    template: z.enum(['docs', 'classic']).default('classic'),
    meta_description: z
      .string()
      .refine(
        withinLength(MAX_META_DESCRIPTION),
        t('globals.messages.maxLength', { max: MAX_META_DESCRIPTION })
      )
      .optional(),
    custom_domain: z
      .string()
      .refine(
        (v) => !v || /^https?:\/\/[^"'()\s\\<>;{}/]+$/.test(v),
        t('helpCenter.invalidCustomDomain')
      )
      .optional(),
    custom_css: z.string().optional(),
    custom_js: z.string().optional(),
    default_locale: z
      .string()
      .min(1, t('globals.messages.required'))
      .regex(localeRe, t('helpCenter.invalidLocale'))
      .default('en'),
    allowed_locales: z
      .array(
        z
          .string()
          .min(1, t('globals.messages.required'))
          .regex(localeRe, t('helpCenter.invalidLocale'))
      )
      .min(1, t('globals.messages.required'))
      .default(['en']),
    theme: z
      .object({
        color: z.string().optional(),
        logo_url: optionalURL,
        nav_links: linkArray,
        favicon: optionalURL,
        tagline: z.string().optional(),
        header: z
          .object({
            heading: z.string().optional(),
            background_type: z.string().optional(),
            background_color: optionalHexColor,
            gradient_from: optionalHexColor,
            gradient_to: optionalHexColor,
            background_image: optionalURL,
            text_color: optionalHexColor
          })
          .optional(),
        layout: z
          .object({
            collections: z.string().optional(),
            columns: z.coerce.number().optional(),
            show_popular_articles: z.boolean().optional(),
            popular_articles_label: z.string().optional()
          })
          .optional(),
        cards: z
          .object({
            hide_description: z.boolean().optional(),
            hide_count: z.boolean().optional(),
            show_authors: z.boolean().optional(),
            show_icon_tile: z.boolean().optional(),
            icon_position: z.enum(['inline', 'top', 'center']).optional()
          })
          .optional(),
        announcement: z
          .object({
            text: z.string().optional(),
            link_label: z.string().optional(),
            link_url: optionalURL
          })
          .optional(),
        footer: z
          .object({
            background_color: optionalHexColor,
            text_color: optionalHexColor,
            tagline: z.string().optional()
          })
          .optional(),
        footer_links: linkArray,
        social_links: z
          .array(
            z.object({
              platform: z.string().min(1, t('globals.messages.required')),
              url: z
                .string()
                .min(1, t('globals.messages.required'))
                .regex(urlRe, t('helpCenter.invalidURL'))
            })
          )
          .optional(),
        article: z
          .object({
            hide_toc: z.boolean().optional(),
            hide_related: z.boolean().optional(),
            show_author: z.boolean().optional()
          })
          .optional()
      })
      .optional()
  })
}
