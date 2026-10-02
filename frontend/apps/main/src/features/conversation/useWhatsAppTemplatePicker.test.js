import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick } from 'vue'
import { useWhatsAppTemplatePicker } from './useWhatsAppTemplatePicker'
import api from '@main/api'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))
vi.mock('@main/api', () => ({ default: { getWhatsAppTemplates: vi.fn() } }))

const template = {
  id: 7,
  name: 'draft_template',
  status: 'APPROVED',
  component_types: ['HEADER', 'BODY', 'BUTTONS'],
  body_content: 'Hello {{1}}',
  header_type: 'TEXT',
  header_content: 'Order {{1}}',
  buttons: [{ type: 'URL', text: 'Track', url: 'https://example.com/{{1}}' }]
}
const params = { 'body:1': 'Customer A', 'header:1': '123', button_url_0: 'order-123' }
let scope
let picker
beforeEach(() => {
  vi.clearAllMocks()
  scope = effectScope()
  picker = scope.run(() => useWhatsAppTemplatePicker())
  api.getWhatsAppTemplates.mockResolvedValue({ data: { data: [template] } })
})
afterEach(() => scope.stop())

describe('restoring a WhatsApp template draft', () => {
  it('restores body, header and URL parameters after fetching templates', async () => {
    await picker.fetchTemplates('1', { templateId: 7, params })
    await nextTick()
    expect(picker.selectedTemplate.value.id).toBe(7)
    expect(picker.templateParams).toEqual(params)
    expect(picker.renderedPreview.value).toBe('Hello Customer A')
    expect(picker.allParamsFilled.value).toBe(true)
  })

  it('retains partial input and leaves newly required parameters empty', async () => {
    await picker.fetchTemplates('1', { templateId: 7, params: { 'body:1': 'Customer A', obsolete: 'old' } })
    await nextTick()
    expect(picker.templateParams).toEqual({ 'body:1': 'Customer A', 'header:1': '', button_url_0: '' })
    expect(picker.allParamsFilled.value).toBe(false)
  })

  it.each([
    [],
    [{ ...template, status: 'REJECTED' }],
    [{ ...template, name: 'libredesk_csat_reserved' }],
    [{ ...template, header_type: 'IMAGE' }]
  ])('does not restore an unavailable or unsupported template (%j)', async (...templates) => {
    api.getWhatsAppTemplates.mockResolvedValue({ data: { data: templates } })
    await picker.fetchTemplates('1', { templateId: 7, params })
    expect(picker.selectedTemplate.value).toBeNull()
    expect(picker.templateParams).toEqual({})
  })

  it('clears the selection and parameters when switching inboxes', async () => {
    await picker.fetchTemplates('1', { templateId: 7, params })
    await picker.fetchTemplates('2')
    expect(picker.selectedTemplate.value).toBeNull()
    expect(picker.templateParams).toEqual({})
  })

  it('ignores a delayed restoration from the previous inbox', async () => {
    let resolveFirst
    api.getWhatsAppTemplates.mockReturnValueOnce(new Promise((resolve) => { resolveFirst = resolve }))
    const first = picker.fetchTemplates('1', { templateId: 7, params })
    await picker.fetchTemplates('2')
    resolveFirst({ data: { data: [template] } })
    await first
    expect(picker.selectedTemplate.value).toBeNull()
    expect(picker.templateParams).toEqual({})
  })
})
