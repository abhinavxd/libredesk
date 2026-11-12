import * as z from 'zod'
export const createFormSchema = (i18n) => z.object({
    id: z.number().optional(),
    applies_to: z.enum(['contact', 'conversation'], {
        required_error: i18n._('{name} Required'),
    }),
    name: z
        .string({
            required_error: i18n._('{name} Required'),
        })
        .min(3, {
            message: i18n._('Must be between {min} and {max} characters', {
                min: 3,
                max: 140,
            })
        })
        .max(140, {
            message: i18n._('Must be between {min} and {max} characters', {
                min: 3,
                max: 140,
            })
        }),
    key: z
        .string({
            required_error: i18n._('{name} Required'),
        })
        .min(3, {
            message: i18n._('Must be between {min} and {max} characters', {
                min: 3,
                max: 140,
            })
        })
        .max(140, {
            message: i18n._('Must be between {min} and {max} characters', {
                min: 3,
                max: 140,
            })
        })
        .regex(/^[a-z0-9_]+$/, {
            message: i18n._('Invalid key'),
        }),
    description: z
        .string({
            required_error: i18n._('{name} Required'),
        })
        .min(3, {
            message: i18n._('Must be between {min} and {max} characters', {
                min: 3,
                max: 300,
            })
        })
        .max(300, {
            message: i18n._('Must be between {min} and {max} characters', {
                min: 3,
                max: 300,
            })
        }),
    data_type: z.enum(['text', 'number', 'checkbox', 'date', 'link', 'list'], {
        required_error: i18n._('{name} Required'),
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
                    message: i18n._('{name} Required'),
                    path: ['values'],
                });
            }
        }
    });
