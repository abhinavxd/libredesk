import * as z from 'zod'

export const createFormSchema = (t) => z.object({
    first_name: z
        .string({
            required_error: t('globals.messages.required'),
        })
        .min(2, {
            message: t('form.error.minmax', {
                min: 2,
                max: 50,
            })
        })
        .max(50, {
            message: t('form.error.minmax', {
                min: 2,
                max: 50,
            })
        }),
    enabled: z.boolean().optional(),
    last_name: z.string().optional(),
    phone_number: z
        .string()
        .optional()
        .refine(val => !val || (/^\d{1,15}$/.test(val)), {
            message: t('form.error.minmax', {
                min: 1,
                max: 15,
            })
        })
        .nullable(),
    phone_number_calling_code: z.string().optional().nullable(),
    avatar_url: z.string().optional().nullable(),
    email: z
        .string({
            required_error: t('globals.messages.required'),
        })
        .email({
            message: t('globals.messages.invalidEmailAddress'),
        }),
})
