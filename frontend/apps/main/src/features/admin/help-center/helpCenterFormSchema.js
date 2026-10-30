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

export const createHelpCenterBasicsSchema = (i18n) =>
  createBaseSchema(i18n).pick({ name: true, slug: true, page_title: true, template: true })

export const createHelpCenterFormSchema = (i18n) =>
  createBaseSchema(i18n)
    .refine((v) => v.allowed_locales.includes(v.default_locale), {
      message: i18n._('The default language is not in the help center\'\'s languages. Add it to the list or pick another default.'),
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
        message: i18n._('The announcement needs text. A link on its own is not shown.'),
        path: ['theme', 'announcement', 'text']
      }
    )

const createBaseSchema = (i18n) => {
  const optionalURL = z
    .string()
    .refine((v) => !v || urlRe.test(v), i18n._('Enter a full URL starting with https:// or a path starting with /.'))
    .optional()

  const optionalHexColor = z
    .string()
    .refine((v) => !v || hexColorRe.test(v), i18n._('Invalid color'))
    .optional()

  const linkArray = z
    .array(
      z.object({
        label: z.string().min(1, i18n._('{name} Required')),
        url: z
          .string()
          .min(1, i18n._('{name} Required'))
          .regex(urlRe, i18n._('Enter a full URL starting with https:// or a path starting with /.'))
      })
    )
    .optional()

  return z.object({
    name: z
      .string()
      .trim()
      .min(1, i18n._('{name} Required'))
      .refine(withinLength(MAX_NAME), i18n._('Must be at most {max} characters', { max: MAX_NAME })),
    slug: z
      .string()
      .trim()
      .min(1, i18n._('{name} Required'))
      .max(200, i18n._('This slug is not allowed. Use a different slug.'))
      .regex(/^[a-z0-9_-]+$/, i18n._('This slug is not allowed. Use a different slug.')),
    page_title: z
      .string()
      .trim()
      .min(1, i18n._('{name} Required'))
      .refine(
        withinLength(MAX_PAGE_TITLE),
        i18n._('Must be at most {max} characters', { max: MAX_PAGE_TITLE })
      ),
    template: z.enum(['docs', 'classic']).default('classic'),
    meta_description: z
      .string()
      .refine(
        withinLength(MAX_META_DESCRIPTION),
        i18n._('Must be at most {max} characters', { max: MAX_META_DESCRIPTION })
      )
      .optional(),
    custom_domain: z
      .string()
      .refine(
        (v) => !v || /^https?:\/\/[^"'()\s\\<>;{}/]+$/.test(v),
        i18n._('Enter the domain\'\'s root URL starting with https://, with no path or trailing slash, e.g. https://help.example.com.')
      )
      .optional(),
    livechat_inbox_id: z.string().optional(),
    custom_css: z.string().optional(),
    custom_js: z.string().optional(),
    default_locale: z
      .string()
      .min(1, i18n._('{name} Required'))
      .regex(localeRe, i18n._('Invalid language code. Use a code like en or en-US.'))
      .default('en'),
    allowed_locales: z
      .array(
        z
          .string()
          .min(1, i18n._('{name} Required'))
          .regex(localeRe, i18n._('Invalid language code. Use a code like en or en-US.'))
      )
      .min(1, i18n._('{name} Required'))
      .default(['en']),
    theme: z
      .object({
        color_scheme: z.enum(['system', 'light', 'dark']).optional(),
        color: z.string().optional(),
        color_dark: optionalHexColor,
        logo_url: optionalURL,
        logo_url_dark: optionalURL,
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
            background_color_dark: optionalHexColor,
            text_color_dark: optionalHexColor,
            tagline: z.string().optional()
          })
          .optional(),
        footer_links: linkArray,
        social_links: z
          .array(
            z.object({
              platform: z.string().min(1, i18n._('{name} Required')),
              url: z
                .string()
                .min(1, i18n._('{name} Required'))
                .regex(urlRe, i18n._('Enter a full URL starting with https:// or a path starting with /.'))
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
