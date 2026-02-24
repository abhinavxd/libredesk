import { z } from 'zod'

export const createPreChatFormSchema = (t, fields = []) => {
  const schemaFields = {}
  
  fields
    .filter(field => field.enabled)
    .forEach(field => {
      let fieldSchema
      
      switch (field.type) {
        case 'email':
          fieldSchema = z.string().email({
            message: t('validation.invalidEmail')
          })
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
          fieldSchema = z.string().refine((val) => val === '' || z.string().url().safeParse(val).success, {
            message: t('validation.invalidUrl')
          })
          break
        
        case 'text':
        case 'list':
        default:
          fieldSchema = z.string().max(1000, {
            message: t('globals.messages.maxLength', { max: 1000 })
          })
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

export const createVisitorInfoSchema = (t, requireContactInfo) => {
  const baseFields = [
    {
      key: 'name',
      type: 'text',
      label: t('globals.terms.name'),
      required: requireContactInfo === 'required',
      enabled: true
    },
    {
      key: 'email',
      type: 'email',
      label: t('globals.terms.email'),
      required: requireContactInfo === 'required',
      enabled: true
    }
  ]
  
  return createPreChatFormSchema(t, baseFields)
}