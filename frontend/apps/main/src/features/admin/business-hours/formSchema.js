import * as z from 'zod'

const timeRegex = /^([01]\d|2[0-3]):([0-5]\d)$/

export const createFormSchema = (t) => z.object({
    name: z.string().min(1, t('globals.messages.required')),
    description: z.string().min(1, t('globals.messages.required')),
    is_always_open: z.boolean(),
    hours: z.record(
        z.object({
            open: z.string().regex(timeRegex, t('form.error.time.invalid')),
            close: z.string().regex(timeRegex, t('form.error.time.invalid')),
        })
    ).optional()
}).superRefine((data, ctx) => {
    if (data.is_always_open === false) {
        if (!data.hours || Object.keys(data.hours).length === 0) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                message: t('globals.messages.required'),
                path: ['hours']
            })
        } else {
            for (const day in data.hours) {
                if (!data.hours[day].open || !data.hours[day].close) {
                    ctx.addIssue({
                        code: z.ZodIssueCode.custom,
                        message: t('globals.messages.required'),
                        path: ['hours', day]
                    })
                }
            }
        }
    }
})
