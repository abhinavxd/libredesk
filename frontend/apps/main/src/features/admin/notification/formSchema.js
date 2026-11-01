import * as z from 'zod';
import { isGoDuration } from '@shared-ui/utils/string';

export const createFormSchema = (i18n) => z.object({
    enabled: z.boolean().default(false),
    username: z.string().default(''),
    host: z.string().nonempty({
        message: i18n._('{name} Required')
    }),
    port: z
        .number({
            invalid_type_error: i18n._('Invalid port value'),
            required_error: i18n._('{name} Required')
        })
        .min(1, { message: i18n._('Must be between {min} and {max}', { min: 1, max: 65535 }) })
        .max(65535, { message: i18n._('Must be between {min} and {max}', { min: 1, max: 65535 }) })
        .default(587),
    password: z.string().default(''),
    max_conns: z
        .number({
            invalid_type_error: i18n._('Must be a number'),
            required_error: i18n._('{name} Required')
        })
        .min(1, { message: i18n._('Must be between {min} and {max}', { min: 1, max: 1000 }) })
        .max(1000, { message: i18n._('Must be between {min} and {max}', { min: 1, max: 1000 }) }),
    idle_timeout: z
        .string()
        .refine(isGoDuration, {
            message: i18n._('Invalid duration format. Please use a valid format (e.g. 30m, 1h, 48h).')
        })
        .default('15s'),
    wait_timeout: z
        .string()
        .refine(isGoDuration, {
            message: i18n._('Invalid duration format. Please use a valid format (e.g. 30m, 1h, 48h).')
        })
        .default('5s'),
    auth_protocol: z.enum(['plain', 'login', 'cram', 'none']),
    email_address: z.string().nonempty({
        message: i18n._('{name} Required')
    }),
    max_msg_retries: z
        .number({
            invalid_type_error: i18n._('Must be a number'),
            required_error: i18n._('{name} Required')
        })
        .min(0, { message: i18n._('Must be between {min} and {max}', { min: 0, max: 1000 }) })
        .max(1000, { message: i18n._('Must be between {min} and {max}', { min: 0, max: 1000 }) })
        .default(2),
    hello_hostname: z.string().optional(),
    tls_type: z.enum(['none', 'starttls', 'tls']),
    tls_skip_verify: z.boolean().optional(),
}).superRefine((smtp, ctx) => {
    if (smtp.auth_protocol === 'none') return
    for (const field of ['username', 'password']) {
        if (!smtp[field]) {
            ctx.addIssue({ code: 'custom', path: [field], message: i18n._('{name} Required') })
        }
    }
});
