import * as z from 'zod'
import { isGoHourMinuteDuration } from '../../../utils/strings'

export const createFormSchema = (t) =>
    z
        .object({
            name: z
                .string()
                .min(1, { message: t('admin.sla.name.valid') })
                .max(255, { message: t('admin.sla.name.valid') }),
            description: z
                .string()
                .min(1, { message: t('admin.sla.description.valid') })
                .max(255, { message: t('admin.sla.description.valid') }),
            first_response_time: z.string().nullable().optional().refine(val => !val || isGoHourMinuteDuration(val), {
                message: t('globals.messages.goHourMinuteDuration'),
            }),
            resolution_time: z.string().nullable().optional().refine(val => !val || isGoHourMinuteDuration(val), {
                message: t('globals.messages.goHourMinuteDuration'),
            }),
            next_response_time: z.string().nullable().optional().refine(val => !val || isGoHourMinuteDuration(val), {
                message: t('globals.messages.goHourMinuteDuration'),
            }),
            notifications: z
                .array(
                    z
                        .object({
                            type: z.enum(['breach', 'warning']),
                            time_delay_type: z.enum(['immediately', 'after', 'before']),
                            time_delay: z.string().optional(),
                            metric: z.enum(['first_response', 'resolution', 'next_response', 'all']),
                            recipients: z
                                .array(z.string())
                                .min(1, {
                                    message: t('globals.messages.selectAtLeastOne', {
                                        name: t('globals.terms.recipient').toLowerCase(),
                                    })
                                }),
                        })
                        .superRefine((obj, ctx) => {
                            if (obj.time_delay_type !== 'immediately') {
                                if (!obj.time_delay || obj.time_delay === '') {
                                    ctx.addIssue({
                                        code: z.ZodIssueCode.custom,
                                        message: t('globals.messages.required'),
                                        path: ['time_delay'],
                                    });
                                } else if (!isGoHourMinuteDuration(obj.time_delay)) {
                                    ctx.addIssue({
                                        code: z.ZodIssueCode.custom,
                                        message: t('globals.messages.goHourMinuteDuration'),
                                        path: ['time_delay'],
                                    });
                                }
                            }
                        })
                )
                .optional()
                .default([]),
        })
        .superRefine((data, ctx) => {
            const { first_response_time, resolution_time, next_response_time } = data
            const isEmpty = !first_response_time && !resolution_time && !next_response_time

            if (isEmpty) {
                const msg = t('admin.sla.atleastOneSLATimeRequired')
                ctx.addIssue({
                    code: z.ZodIssueCode.custom,
                    path: ['first_response_time'],
                    message: msg,
                })
                ctx.addIssue({
                    code: z.ZodIssueCode.custom,
                    path: ['resolution_time'],
                    message: msg,
                })
                ctx.addIssue({
                    code: z.ZodIssueCode.custom,
                    path: ['next_response_time'],
                    message: msg,
                })
            }
        })
