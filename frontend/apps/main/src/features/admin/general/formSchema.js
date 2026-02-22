import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  site_name: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(1, {
      message: t('admin.general.siteName.min'),
    }),
  lang: z.string().optional(),
  timezone: z.string().optional(),
  business_hours_id: z.string().optional(),
  logo_url: z.string().url({
    message: t('admin.general.logoURL.valid'),
  }).or(z.literal(''))
    .optional(),
  root_url: z
    .string({
      required_error: t('globals.messages.required')
    })
    .url({
      message: t('admin.general.rootURL.valid')
    }).url(),
  favicon_url: z
    .string({
      required_error: t('globals.messages.required')
    })
    .url({
      message: t('admin.general.faviconURL.valid')
    }).url(),
  max_file_upload_size: z
    .number({
      required_error: t('globals.messages.required')
    })
    .min(1, {
      message: t('admin.general.maxAllowedFileUploadSize.valid')
    })
    .max(500, {
      message: t('admin.general.maxAllowedFileUploadSize.valid')
    }),
  allowed_file_upload_extensions: z.array(z.string()).nullable().default([]).optional()
})
