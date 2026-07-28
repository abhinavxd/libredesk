import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import api from '@main/api'
import { PLACEHOLDER_PATTERN, extractPlaceholders } from './whatsappTemplate.js'

// Media-header templates need a media ID the dashboard can't supply, AUTHENTICATION templates need OTP button params, and libredesk_csat_* names are reserved for surveys.
const SENDABLE_HEADER_TYPES = ['', 'NONE', 'TEXT']
const RESERVED_NAME_PREFIX = 'libredesk_csat_'

export function useWhatsAppTemplatePicker() {
  const { t } = useI18n()
  const emitter = useEmitter()
  const templates = ref([])
  const selectedTemplate = ref(null)
  const templateParams = reactive({})

  const approvedTemplates = computed(() =>
    templates.value.filter(
      (tmpl) =>
        tmpl.status === 'APPROVED' &&
        tmpl.category !== 'AUTHENTICATION' &&
        !tmpl.name.startsWith(RESERVED_NAME_PREFIX) &&
        SENDABLE_HEADER_TYPES.includes(tmpl.header_type || '')
    )
  )

  const placeholders = computed(() => {
    if (!selectedTemplate.value) return []
    const out = []
    for (const name of extractPlaceholders([selectedTemplate.value.body_content])) {
      out.push({ key: `body:${name}`, name, partLabel: t('globals.terms.body') })
    }
    if (selectedTemplate.value.header_type === 'TEXT' && selectedTemplate.value.header_content) {
      for (const name of extractPlaceholders([selectedTemplate.value.header_content])) {
        out.push({ key: `header:${name}`, name, partLabel: t('admin.whatsappTemplates.header') })
      }
    }
    return out
  })

  const urlButtonParams = computed(() => {
    const buttons = selectedTemplate.value?.buttons || []
    const out = []
    buttons.forEach((btn, idx) => {
      if ((btn.type || '').toUpperCase() !== 'URL') return
      if (!(btn.url || '').match(PLACEHOLDER_PATTERN)) return
      out.push({ key: `button_url_${idx}`, label: btn.text || btn.url, url: btn.url })
    })
    return out
  })

  const allParamKeys = computed(() => [
    ...placeholders.value.map((ph) => ph.key),
    ...urlButtonParams.value.map((btn) => btn.key)
  ])

  const allParamsFilled = computed(() =>
    allParamKeys.value.every((key) => (templateParams[key] || '').trim() !== '')
  )

  const renderedPreview = computed(() => {
    if (!selectedTemplate.value) return ''
    let body = selectedTemplate.value.body_content || ''
    for (const ph of placeholders.value) {
      if (ph.key.startsWith('body:')) {
        const v = templateParams[ph.key]
        if (v) body = body.split(`{{${ph.name}}}`).join(v)
      }
    }
    return body
  })

  const isFetchingTemplates = ref(false)

  const fetchTemplates = async (inboxID) => {
    reset()
    if (!inboxID) return
    try {
      isFetchingTemplates.value = true
      const resp = await api.getWhatsAppTemplates(inboxID)
      templates.value = resp.data.data || []
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      isFetchingTemplates.value = false
    }
  }

  const pickTemplate = (tmpl) => {
    selectedTemplate.value = tmpl
    for (const key of allParamKeys.value) {
      if (templateParams[key] === undefined) templateParams[key] = ''
    }
  }

  const reset = () => {
    selectedTemplate.value = null
    templates.value = []
    Object.keys(templateParams).forEach((k) => delete templateParams[k])
  }

  watch(selectedTemplate, () => {
    const valid = new Set(allParamKeys.value)
    for (const k of Object.keys(templateParams)) {
      if (!valid.has(k)) delete templateParams[k]
    }
  })

  return {
    templates,
    selectedTemplate,
    templateParams,
    approvedTemplates,
    placeholders,
    urlButtonParams,
    allParamsFilled,
    renderedPreview,
    isFetchingTemplates,
    pickTemplate,
    fetchTemplates,
    reset
  }
}
