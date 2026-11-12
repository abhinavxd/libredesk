import * as z from 'zod';

export const createFormSchema = (i18n) => z
    .object({
        name: z.string({
            required_error: i18n._('{name} Required'),
        }),
        description: z.string().optional().default(''),
        enabled: z.boolean().default(true),
        type: z.string({
            required_error: i18n._('{name} Required'),
        }),
        events: z.array(z.string()).optional(),
    })
    .superRefine((data, ctx) => {
        if (data.type === 'conversation_update' && (!data.events || data.events.length === 0)) {
            ctx.addIssue({
                path: ['events'],
                message: i18n._('Please select at least one event'),
                code: z.ZodIssueCode.custom,
            });
        }
    });
