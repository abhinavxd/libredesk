import * as z from 'zod'

export const TEMPLATE_CATEGORIES = ['MARKETING', 'UTILITY', 'AUTHENTICATION']
export const HEADER_TYPES = ['NONE', 'TEXT']
export const BUTTON_TYPES = ['URL', 'PHONE_NUMBER', 'QUICK_REPLY']

export const createFormSchema = (t) =>
  z.object({
    inbox_id: z.number().int().positive(t('globals.messages.required')),
    name: z
      .string()
      .min(1, t('globals.messages.required'))
      .max(512)
      .regex(/^[a-z0-9_]+$/, t('admin.whatsappTemplates.nameInvalid')),
    language: z.string().min(1, t('globals.messages.required')).max(20),
    category: z.enum(TEMPLATE_CATEGORIES),
    header_type: z.enum(HEADER_TYPES).optional(),
    header_content: z.string().optional(),
    body_content: z.string().min(1, t('globals.messages.required')),
    footer_content: z.string().max(60).optional(),
    buttons: z
      .array(
        z.object({
          type: z.enum(BUTTON_TYPES),
          text: z.string().min(1, t('globals.messages.required')),
          url: z.string().optional(),
          phone_number: z.string().optional()
        })
      )
      .max(3)
      .optional(),
    sample_values: z.record(z.string(), z.string()).optional()
  })
    .superRefine((data, ctx) => {
      if (data.header_type === 'TEXT' && !data.header_content?.trim()) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['header_content'],
          message: t('globals.messages.required')
        })
      }
    })
