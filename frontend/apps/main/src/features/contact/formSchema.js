import * as z from 'zod'
import { phoneNumberSchema } from '@shared-ui/utils/phone.js'

export const createFormSchema = (t) => z.object({
    first_name: z
        .string({
            required_error: t('globals.messages.required'),
        })
        .min(2, {
            message: t('validation.minmax', {
                min: 2,
                max: 50,
            })
        })
        .max(50, {
            message: t('validation.minmax', {
                min: 2,
                max: 50,
            })
        }),
    enabled: z.boolean().optional(),
    last_name: z.string().optional(),
    phone_number: phoneNumberSchema(t).optional().nullable(),
    phone_number_country_code: z.string().optional().nullable(),
    country: z.string().optional().nullable(),
    avatar_url: z.string().optional().nullable(),
    email: z
        .string({
            required_error: t('globals.messages.required'),
        })
        .email({
            message: t('validation.invalidEmail'),
        }),
})
