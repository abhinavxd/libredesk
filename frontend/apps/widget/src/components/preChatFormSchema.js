import { z } from 'zod'
import { countryCodeKey, phoneNumberSchema, countryCodeSchema } from '@shared-ui/utils/phone.js'

export const createPreChatFormSchema = (i18n, fields = []) => {
  const schemaFields = {}
  
  fields
    .filter(field => field.enabled)
    .forEach(field => {
      let fieldSchema

      if (field.type === 'phone') {
        let numberSchema = phoneNumberSchema(i18n, field.required)
        let codeSchema = countryCodeSchema(i18n, field.required)

        if (!field.required) {
          numberSchema = numberSchema.optional()
          codeSchema = codeSchema.optional()
        }

        schemaFields[field.key] = numberSchema
        schemaFields[countryCodeKey(field.key)] = codeSchema
        return
      }

      switch (field.type) {
        case 'email':
          fieldSchema = z
            .string()
            .max(254, { message: i18n._('Must be at most {max} characters', { max: 254 }) })
            .email({ message: i18n._('Invalid email address') })
          break

        case 'number': {
          const emptyToUndefined = (v) => (v === '' || v === null ? undefined : v)
          if (field.required) {
            schemaFields[field.key] = z.preprocess(
              emptyToUndefined,
              z.number({
                required_error: i18n._('{name} Required'),
                invalid_type_error: i18n._('Invalid')
              })
            )
          } else {
            schemaFields[field.key] = z.preprocess(
              emptyToUndefined,
              z.number({ invalid_type_error: i18n._('Invalid') }).optional()
            )
          }
          return
        }

        case 'checkbox':
          schemaFields[field.key] = z.boolean().default(false)
          return

        case 'date':
          fieldSchema = z.string().regex(/^(\d{4}-\d{2}-\d{2}|)$/, {
            message: i18n._('Invalid')
          })
          break

        case 'link':
          if (field.required) {
            schemaFields[field.key] = z
              .string()
              .max(1000, { message: i18n._('Must be at most {max} characters', { max: 1000 }) })
              .min(1, { message: i18n._('{name} Required') })
              .url({ message: i18n._('Invalid URL') })
          } else {
            schemaFields[field.key] = z
              .string()
              .max(1000, { message: i18n._('Must be at most {max} characters', { max: 1000 }) })
              .refine((val) => val === '' || z.string().url().safeParse(val).success, {
                message: i18n._('Invalid URL')
              })
              .optional()
          }
          return

        case 'text':
        case 'list':
        default: {
          const max = field.key === 'name' ? 128 : 1000
          fieldSchema = z.string().max(max, {
            message: i18n._('Must be at most {max} characters', { max })
          })
        }
      }
      
      if (field.required && field.type !== 'checkbox') {
        fieldSchema = fieldSchema.min(1, {
          message: i18n._('{name} Required')
        })
      } else if (field.type !== 'checkbox') {
        fieldSchema = fieldSchema.or(z.literal('')).optional()
      }
      
      schemaFields[field.key] = fieldSchema
    })
  
  return z.object(schemaFields)
}
