import * as z from 'zod'

export const createFormSchema = (i18n) => z.object({
  site_name: z
    .string({
      required_error: i18n._('{name} Required'),
    })
    .min(1, {
      message: i18n._('Site name should be at least 1 character'),
    }),
  lang: z.string().optional(),
  timezone: z.string().optional(),
  business_hours_id: z.string().optional(),
  logo_url: z.string().url({
    message: i18n._('Logo URL should be a valid URL'),
  }).or(z.literal(''))
    .optional(),
  root_url: z
    .string({
      required_error: i18n._('{name} Required')
    })
    .url({
      message: i18n._('Root URL should be a valid URL')
    }).url(),
  favicon_url: z
    .string({
      required_error: i18n._('{name} Required')
    })
    .url({
      message: i18n._('Favicon URL should be a valid URL')
    }).url(),
  max_file_upload_size: z
    .number({
      required_error: i18n._('{name} Required')
    })
    .min(1, {
      message: i18n._('Max allowed file upload size should be between 1 and 500 MB')
    })
    .max(500, {
      message: i18n._('Max allowed file upload size should be between 1 and 500 MB')
    }),
  allowed_file_upload_extensions: z.array(z.string()).nullable().optional().default([]),
  show_conversation_subject: z.boolean().optional()
})
