import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  first_name: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(2, {
      message: t('form.error.minmax', {
        min: 2,
        max: 100,
      })
    })
    .max(100, {
      message: t('form.error.minmax', {
        min: 2,
        max: 100,
      })
    }),

  last_name: z.string().optional(),

  avatar_url: z
    .string()
    .url({
      message: t('globals.messages.invalidUrl'),
    })
    .optional()
    .or(z.literal('')),

  product_name: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(2, {
      message: t('form.error.minmax', {
        min: 2,
        max: 255,
      })
    })
    .max(255, {
      message: t('form.error.minmax', {
        min: 2,
        max: 255,
      })
    }),

  product_description: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(10, {
      message: t('form.error.minmax', {
        min: 10,
        max: 1000,
      })
    })
    .max(1000, {
      message: t('form.error.minmax', {
        min: 10,
        max: 1000,
      })
    }),

  answer_length: z
    .enum(['concise', 'medium', 'long'], {
      required_error: t('globals.messages.required'),
      invalid_type_error: t('globals.messages.invalid', { name: t('ai.assistant.answerLength') })
    }),

  answer_tone: z
    .enum(['neutral', 'friendly', 'professional', 'humorous'], {
      required_error: t('globals.messages.required'),
      invalid_type_error: t('globals.messages.invalid', { name: t('ai.assistant.answerTone') })
    }),

  enabled: z.boolean().optional().default(true),

  hand_off: z.boolean().optional().default(false),

  hand_off_team: z
    .number()
    .int({
      message: t('globals.messages.invalid', { name: t('globals.terms.team') })
    })
    .optional()
    .nullable()
    .default(null),
})