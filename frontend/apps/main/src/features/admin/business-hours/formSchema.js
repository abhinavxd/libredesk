import * as z from 'zod'

const timeRegex = /^([01]\d|2[0-3]):([0-5]\d)$/

export const createFormSchema = (i18n) => z.object({
    name: z.string().min(1, i18n._('{name} Required')),
    description: z.string().nullable().optional().transform(v => v ?? ''),
    is_always_open: z.boolean(),
    hours: z.record(
        z.object({
            open: z.string().regex(timeRegex, i18n._('Invalid time format (HH:mm)')),
            close: z.string().regex(timeRegex, i18n._('Invalid time format (HH:mm)')),
        })
    ).optional()
}).superRefine((data, ctx) => {
    if (data.is_always_open === false) {
        if (!data.hours || Object.keys(data.hours).length === 0) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: i18n._('{name} Required'),
                path: ['hours']
            })
        } else {
            for (const day in data.hours) {
                if (!data.hours[day].open || !data.hours[day].close) {
                    ctx.addIssue({
                        code: z.ZodIssueCode.custom,
                        message: i18n._('{name} Required'),
                        path: ['hours', day]
                    })
                }
            }
        }
    }
})
