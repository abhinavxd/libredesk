import * as z from 'zod'
import { isGoHourMinuteDuration } from '@shared-ui/utils/string'

export const createFormSchema = (i18n) =>
    z
        .object({
            name: z
                .string()
                .min(1, { message: i18n._('SLA Policy name should be between 1 and 255 characters') })
                .max(255, { message: i18n._('SLA Policy name should be between 1 and 255 characters') }),
            description: z
                .string()
                .max(255, { message: i18n._('SLA Policy description should be between 1 and 255 characters') })
                .nullable()
                .optional()
                .transform(v => v ?? ''),
            first_response_time: z.string().nullable().optional().refine(val => !val || isGoHourMinuteDuration(val), {
                message: i18n._('Invalid duration format. Please use a valid format (e.g. 30m, 1h, 48h).'),
            }),
            resolution_time: z.string().nullable().optional().refine(val => !val || isGoHourMinuteDuration(val), {
                message: i18n._('Invalid duration format. Please use a valid format (e.g. 30m, 1h, 48h).'),
            }),
            next_response_time: z.string().nullable().optional().refine(val => !val || isGoHourMinuteDuration(val), {
                message: i18n._('Invalid duration format. Please use a valid format (e.g. 30m, 1h, 48h).'),
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
                                    message: i18n._('Please select at least one recipient')
                                }),
                        })
                        .superRefine((obj, ctx) => {
                            if (obj.time_delay_type !== 'immediately') {
                                if (!obj.time_delay || obj.time_delay === '') {
                                    ctx.addIssue({
                                        code: z.ZodIssueCode.custom,
                                        message: i18n._('{name} Required'),
                                        path: ['time_delay'],
                                    });
                                } else if (!isGoHourMinuteDuration(obj.time_delay)) {
                                    ctx.addIssue({
                                        code: z.ZodIssueCode.custom,
                                        message: i18n._('Invalid duration format. Please use a valid format (e.g. 30m, 1h, 48h).'),
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
                const msg = i18n._('At least one of First Response Time, Next Response Time, or Resolution Time is required.')
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
