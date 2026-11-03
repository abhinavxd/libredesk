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
  allowed_file_upload_extensions: z.array(z.string()).nullable().optional().default([]),
  conversation_retention_days: z
    .number({
      required_error: t('globals.messages.required')
    })
    .int({
      message: t('admin.general.conversationRetentionDays.valid')
    })
    .min(0, {
      message: t('admin.general.conversationRetentionDays.valid')
    })
    .max(3650, {
      message: t('admin.general.conversationRetentionDays.valid')
    })
    .optional()
    .default(0),
  conversation_retention_delete_contacts: z.boolean().optional(),
  show_conversation_subject: z.boolean().optional()
})
