import * as z from 'zod'

// An uploaded logo is saved as the app-relative URL the upload returns.
const UPLOADED_LOGO_RE =
  /^\/uploads\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const isHTTPURL = (v) => /^https?:\/\//i.test(v) && z.string().url().safeParse(v).success

export const createFormSchema = (t) =>
  z.object({
    // Blank title and logo fall back to the Fernmail defaults.
    site_name: z.string().optional().default(''),
    lang: z.string().optional(),
    timezone: z.string().optional(),
    logo_url: z
      .string()
      .refine((v) => v === '' || UPLOADED_LOGO_RE.test(v) || isHTTPURL(v), {
        message: t('admin.general.logoURL.valid')
      })
      .optional()
      .default(''),
    root_url: z
      .string({
        required_error: t('globals.messages.required')
      })
      .url({
        message: t('admin.general.rootURL.valid')
      })
      .url(),
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
    show_conversation_subject: z.boolean().optional()
  })
