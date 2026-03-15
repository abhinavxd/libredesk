import * as z from 'zod'
import { isGoHourMinuteDuration } from '@shared-ui/utils/string'

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
                message: t('validation.invalidDuration'),
            }),
            resolution_time: z.string().nullable().optional().refine(val => !val || isGoHourMinuteDuration(val), {
                message: t('validation.invalidDuration'),
            }),
            next_response_time: z.string().nullable().optional().refine(val => !val || isGoHourMinuteDuration(val), {
                message: t('validation.invalidDuration'),
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
                                    message: t('validation.selectAtLeastOneRecipient')
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
                                        message: t('validation.invalidDuration'),
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
