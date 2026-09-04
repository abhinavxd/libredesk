<template>
  <Spinner v-if="loading" />
  <p v-else-if="loadFailed" class="text-sm text-muted-foreground">
    {{ t('globals.messages.somethingWentWrong') }}
  </p>
  <div v-else class="flex flex-col">
    <CustomBreadcrumb :links="breadcrumbLinks" class="mb-5" />

    <div class="flex gap-6">
      <div class="w-[420px] shrink-0 pr-2">
        <HelpCenterForm
          :help-center="helpCenter"
          :submit-form="handleSave"
          :is-loading="isSubmitting"
          @cancel="goBack"
          @change="onFormChange"
        />
      </div>

      <!-- The preview page pins its footer to the bottom of the frame; an unbounded frame strands it below the content. -->
      <div class="flex-1 min-w-0">
        <div class="sticky top-0 space-y-2">
          <div class="flex items-center gap-2">
            <Select v-model="previewPage">
              <SelectTrigger class="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="landing">{{ t('helpCenter.styling.landingPage') }}</SelectItem>
                <SelectItem value="article">{{ t('helpCenter.styling.articlePage') }}</SelectItem>
              </SelectContent>
            </Select>
            <Select v-model="previewDevice">
              <SelectTrigger class="w-32 ml-auto">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="desktop">{{ t('globals.terms.desktop') }}</SelectItem>
                <SelectItem value="tablet">{{ t('globals.terms.tablet') }}</SelectItem>
                <SelectItem value="mobile">{{ t('globals.terms.mobile') }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="rounded-md border overflow-hidden bg-muted">
            <div class="flex items-end gap-3 px-3 pt-2.5">
              <span class="flex items-center gap-1.5 pb-2.5">
                <span class="size-2.5 rounded-full bg-muted-foreground/25" />
                <span class="size-2.5 rounded-full bg-muted-foreground/25" />
                <span class="size-2.5 rounded-full bg-muted-foreground/25" />
              </span>
              <span
                class="flex min-w-0 max-w-56 items-center gap-2 rounded-t-md bg-background px-3 py-2 text-xs"
              >
                <img
                  v-if="previewFavicon"
                  :src="previewFavicon"
                  alt=""
                  class="size-3.5 shrink-0 rounded-sm"
                />
                <Globe v-else class="size-3.5 shrink-0 text-muted-foreground" />
                <span class="truncate">{{ previewTitle }}</span>
                <X class="size-3 shrink-0 text-muted-foreground" />
              </span>
            </div>
            <div class="flex items-center border-b bg-background px-3 py-2">
              <a
                :href="previewLiveURL"
                target="_blank"
                rel="noopener"
                class="flex min-w-0 flex-1 items-center gap-1.5 rounded-full bg-muted px-3 py-1 text-xs text-muted-foreground hover:text-foreground hover:underline"
              >
                <span class="min-w-0 truncate">{{ previewURL }}</span>
                <ExternalLink class="size-3 shrink-0" />
              </a>
            </div>
            <div ref="previewBox" class="h-[calc(100dvh-13rem)] overflow-hidden bg-muted">
              <iframe
                ref="previewFrame"
                class="border-0 bg-background block origin-top-left"
                :style="frameStyle"
                :title="t('globals.terms.helpCenter')"
                sandbox="allow-same-origin"
                @load="lockPreview"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ExternalLink, Globe, X } from 'lucide-vue-next'
import { Spinner } from '@shared-ui/components/ui/spinner'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import HelpCenterForm from '@main/features/admin/help-center/HelpCenterForm.vue'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useAppSettingsStore } from '@/stores/appSettings'
import api from '@/api'

const props = defineProps({
  id: {
    type: [String, Number],
    required: true
  }
})

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()
const appSettingsStore = useAppSettingsStore()

const loading = ref(true)
const loadFailed = ref(false)
const isSubmitting = ref(false)
const helpCenter = ref(null)
const previewFrame = ref(null)
const origin = computed(() =>
  (appSettingsStore.settings?.['app.root_url'] || window.location.origin).replace(/\/$/, '')
)
const previewPage = ref('landing')
const previewDevice = ref('desktop')
const previewBox = ref(null)
const previewTitle = computed(() => {
  const values = lastFormValues.value || helpCenter.value || {}
  if (previewPage.value === 'article') return t('helpCenter.preview.sampleArticleTitle')
  return values.page_title || values.name || ''
})

const previewFavicon = computed(
  () => (lastFormValues.value || helpCenter.value || {})?.theme?.favicon || ''
)

const helpCenterPath = (values) => `/hc/${values.slug || ''}/${values.default_locale || 'en'}`

const previewURL = computed(() => {
  const path = helpCenterPath(lastFormValues.value || helpCenter.value || {})
  return previewPage.value === 'article'
    ? `${origin.value}${path}/articles/sample-article`
    : `${origin.value}${path}`
})
// Uses the saved slug, not the form: an unsaved slug has no live page yet.
const previewLiveURL = computed(() => `${origin.value}${helpCenterPath(helpCenter.value || {})}`)
const boxSize = ref({ width: 0, height: 0 })

// The preview pane is narrower than a real desktop viewport, so the frame is rendered at the
// device width and scaled down to fit. Otherwise "desktop" sits below the page's own breakpoints.
const DEVICE_WIDTH = { desktop: 1440, tablet: 768, mobile: 390 }

const frameStyle = computed(() => {
  const width = DEVICE_WIDTH[previewDevice.value]
  const { width: boxWidth, height: boxHeight } = boxSize.value
  if (!boxWidth || !boxHeight) return { width: `${width}px` }
  const scale = Math.min(1, boxWidth / width)
  return {
    width: `${width}px`,
    height: `${boxHeight / scale}px`,
    transform: `scale(${scale}) translateX(${(boxWidth / scale - width) / 2}px)`
  }
})
const lastFormValues = ref(null)

let previewTimer = null

const breadcrumbLinks = computed(() => [
  { path: 'help-center-list', label: t('globals.terms.helpCenter', 2) },
  { path: '', label: helpCenter.value?.name || '' }
])

const goBack = () => router.push({ name: 'help-center-list' })

// Links resolve against the admin origin; following one would replace the preview.
const lockPreview = () => {
  const body = previewFrame.value?.contentDocument?.body
  if (body) body.inert = true
}

// A slower earlier render can land after a newer one, so only the latest request may paint.
let previewRequest = 0

const renderPreview = async (values) => {
  const request = ++previewRequest
  try {
    const { data } = await api.previewHelpCenter(props.id, values, previewPage.value)
    if (request === previewRequest && previewFrame.value) previewFrame.value.srcdoc = data
  } catch {
    // A half-filled form can fail validation while typing; the last good preview stays up.
  }
}

const onFormChange = (values) => {
  lastFormValues.value = values
  clearTimeout(previewTimer)
  previewTimer = setTimeout(() => renderPreview(values), 300)
}

watch(previewPage, () => {
  if (lastFormValues.value) renderPreview(lastFormValues.value)
})

const handleSave = async (formData) => {
  isSubmitting.value = true
  try {
    const { data } = await api.updateHelpCenter(props.id, formData)
    helpCenter.value = data.data
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSubmitting.value = false
  }
}

onMounted(async () => {
  try {
    const { data } = await api.getHelpCenter(props.id)
    helpCenter.value = data.data
  } catch (error) {
    loadFailed.value = true
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
})

// The pane only exists once the help center has loaded, so observe it when the ref fills in.
let boxObserver = null

watch(previewBox, (box) => {
  boxObserver?.disconnect()
  if (!box) return
  boxObserver = new ResizeObserver(([entry]) => {
    boxSize.value = { width: entry.contentRect.width, height: entry.contentRect.height }
  })
  boxObserver.observe(box)
})

onBeforeUnmount(() => {
  clearTimeout(previewTimer)
  boxObserver?.disconnect()
})
</script>
