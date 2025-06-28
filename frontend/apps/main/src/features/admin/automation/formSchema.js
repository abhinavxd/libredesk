import * as z from 'zod';

export const createFormSchema = (t) => z
    .object({
        name: z.string({
            required_error: t('globals.messages.required'),
        }),
        description: z.string({
            required_error: t('globals.messages.required'),
        }),
        enabled: z.boolean().default(true),
        type: z.string({
            required_error: t('globals.messages.required'),
        }),
        events: z.array(z.string()).optional(),
    })
    .superRefine((data, ctx) => {
        if (data.type === 'conversation_update' && (!data.events || data.events.length === 0)) {
            ctx.addIssue({
                path: ['events'],
                message: t('globals.messages.selectAtLeastOne', {
                    name: t('globals.terms.event')
                }),
                code: z.ZodIssueCode.custom,
            });
        }
    });
