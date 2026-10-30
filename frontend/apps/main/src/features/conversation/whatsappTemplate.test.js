import { describe, expect, it } from 'vitest'
import { canEditWhatsAppTemplate, supportsWhatsAppTemplateContent } from './whatsappTemplate.js'

const template = {
  name: 'order_update',
  status: 'APPROVED',
  category: 'UTILITY',
  component_types: ['BODY'],
  buttons: []
}

describe('supported WhatsApp template content', () => {
  it.each([
    ['plain text', {}, true],
    ['carousel', { component_types: ['BODY', 'CAROUSEL'] }, false],
    ['unknown component', { component_types: ['BODY', 'UNKNOWN'] }, false],
    ['unknown legacy template', { component_types: null }, false],
    ['empty component list', { component_types: [] }, false],
    ['repeated supported types', { component_types: ['BODY', 'BODY'] }, true],
    ['media header', { header_type: 'IMAGE' }, false],
    ['authentication', { category: 'AUTHENTICATION' }, false],
    ['flow button', { buttons: [{ type: 'FLOW' }] }, false],
    ['URL button', { buttons: [{ type: 'URL' }] }, true]
  ])('%s', (_name, changes, supported) => {
    expect(Boolean(supportsWhatsAppTemplateContent({ ...template, ...changes }))).toBe(supported)
  })
})

describe('editable WhatsApp templates', () => {
  it.each(['APPROVED', 'REJECTED', 'PAUSED'])('allows %s text templates', (status) => {
    expect(canEditWhatsAppTemplate({ ...template, status })).toBe(true)
  })
  it.each(['PENDING', 'DISABLED', 'PENDING_DELETION', 'IN_APPEAL'])(
    'blocks %s templates',
    (status) => {
      expect(canEditWhatsAppTemplate({ ...template, status })).toBe(false)
    }
  )
  it('protects reserved surveys', () => {
    expect(canEditWhatsAppTemplate({ ...template, name: 'libredesk_csat_2' })).toBe(false)
  })
  it('blocks carousel editing', () => {
    expect(canEditWhatsAppTemplate({ ...template, component_types: ['BODY', 'CAROUSEL'] })).toBe(
      false
    )
  })
})
