import * as z from 'zod'
export const createFormSchema = (t) => z.object({
    id: z.number().optional(),
    applies_to: z.enum(['contact', 'conversation'], {
        required_error: t('globals.messages.required'),
    }),
    name: z
        .string({
            required_error: t('globals.messages.required'),
        })
        .min(3, {
            message: t('form.error.minmax', {
                min: 3,
                max: 140,
            })
        })
        .max(140, {
            message: t('form.error.minmax', {
                min: 3,
                max: 140,
            })
        }),
    key: z
        .string({
            required_error: t('globals.messages.required'),
        })
        .min(3, {
            message: t('form.error.minmax', {
                min: 3,
                max: 140,
            })
        })
        .max(140, {
            message: t('form.error.minmax', {
                min: 3,
                max: 140,
            })
        })
        .regex(/^[a-z0-9_]+$/, {
            message: t('globals.messages.invalid', {
                name: t('globals.terms.key'),
            }),
        }),
    description: z
        .string({
            required_error: t('globals.messages.required'),
        })
        .min(3, {
            message: t('form.error.minmax', {
                min: 3,
                max: 300,
            })
        })
        .max(300, {
            message: t('form.error.minmax', {
                min: 3,
                max: 300,
            })
        }),
    data_type: z.enum(['text', 'number', 'checkbox', 'date', 'link', 'list'], {
        required_error: t('globals.messages.required'),
    }),
    regex: z.string().optional(),
    regex_hint: z.string().optional(),
    values: z.array(z.string())
        .default([])
})
    .superRefine((data, ctx) => {
        if (data.data_type === 'list') {
            // If data_type is 'list', values should be defined and have at least one item.
            if (!data.values || data.values.length === 0) {
                ctx.addIssue({
                    code: z.ZodIssueCode.too_small,
                    minimum: 1,
                    type: "array",
                    inclusive: true,
                    message: t('globals.messages.required'),
                    path: ['values'],
                });
            }
        }
    });
