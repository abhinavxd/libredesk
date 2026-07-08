import { z } from 'zod'
import { countryCodeKey, phoneNumberSchema, countryCodeSchema } from '@shared-ui/utils/phone.js'

export const createPreChatFormSchema = (t, fields = []) => {
  const schemaFields = {}
  
  fields
    .filter(field => field.enabled)
    .forEach(field => {
      let fieldSchema

      if (field.type === 'phone') {
        let numberSchema = phoneNumberSchema(t, field.required)
        let codeSchema = countryCodeSchema(t, field.required)

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
            .max(254, { message: t('globals.messages.maxLength', { max: 254 }) })
            .email({ message: t('validation.invalidEmail') })
          break

        case 'number':
          fieldSchema = z.coerce.number({
            invalid_type_error: t('validation.invalid')
          })
          break

        case 'checkbox':
          fieldSchema = z.boolean().default(false)
          break

        case 'date':
          fieldSchema = z.string().regex(/^(\d{4}-\d{2}-\d{2}|)$/, {
            message: t('validation.invalid')
          })
          break

        case 'link':
          fieldSchema = z
            .string()
            .max(1000, { message: t('globals.messages.maxLength', { max: 1000 }) })
            .refine((val) => val === '' || z.string().url().safeParse(val).success, {
              message: t('validation.invalidUrl')
            })
          break

        case 'text':
        case 'list':
        default: {
          const max = field.key === 'name' ? 128 : 1000
          fieldSchema = z.string().max(max, {
            message: t('globals.messages.maxLength', { max })
          })
        }
      }
      
      if (field.required && field.type !== 'checkbox') {
        fieldSchema = fieldSchema.min(1, {
          message: t('globals.messages.required')
        })
      } else if (field.type !== 'checkbox') {
        fieldSchema = fieldSchema.optional()
      }
      
      schemaFields[field.key] = fieldSchema
    })
  
  return z.object(schemaFields)
}
