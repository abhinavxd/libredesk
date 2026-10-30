import * as z from 'zod'
import { phoneNumberSchema } from '@shared-ui/utils/phone.js'

export const createFormSchema = (i18n) => z.object({
    first_name: z
        .string({
            required_error: i18n._('{name} Required'),
        })
        .min(2, {
            message: i18n._('Must be between {min} and {max} characters', {
                min: 2,
                max: 50,
            })
        })
        .max(50, {
            message: i18n._('Must be between {min} and {max} characters', {
                min: 2,
                max: 50,
            })
        }),
    enabled: z.boolean().optional(),
    last_name: z.string().optional(),
    phone_number: phoneNumberSchema(i18n).optional().nullable(),
    phone_number_country_code: z.string().optional().nullable(),
    country: z.string().optional().nullable(),
    avatar_url: z.string().optional().nullable(),
    email: z
        .string()
        .email({
            message: i18n._('Invalid email address'),
        })
        .or(z.literal(''))
        .optional()
        .nullable(),
}).superRefine((data, ctx) => {
    if (!data.email && !data.phone_number) {
        ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: i18n._('Add an email or a phone number. A contact needs at least one to be reached.'),
            path: ['email']
        })
    }
})
