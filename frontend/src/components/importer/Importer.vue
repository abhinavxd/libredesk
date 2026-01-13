<template>
  <div>
    <Button variant="secondary" @click="openDialog">
      {{ $t('globals.terms.import') }}
    </Button>

    <Dialog v-model:open="showDialog">
      <DialogContent class="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{{
            $t('globals.messages.import', {
              name: $t(entityKey, 2).toLowerCase()
            })
          }}</DialogTitle>
        </DialogHeader>

        <div class="space-y-4 py-4">
          <div v-if="!loading && !status" class="space-y-4">
            <div
              @click="$refs.fileInput.click()"
              class="flex items-center h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground"
            >
              <span class="flex-1 truncate" :class="!file && 'text-muted-foreground'">
                {{
                  file
                    ? file.name
                    : $t('globals.messages.select', { name: $t('globals.terms.csvFile') })
                }}
              </span>
              <Upload class="h-4 w-4 text-muted-foreground flex-shrink-0" />
            </div>
            <input
              type="file"
              accept=".csv"
              @change="onFileSelect"
              ref="fileInput"
              class="hidden"
            />

            <Alert>
              <AlertTitle>{{ $t('importer.requiredCSVFormat') }}</AlertTitle>
              <AlertDescription class="mt-2">
                <div class="bg-muted p-3 rounded text-xs font-mono overflow-x-auto leading-relaxed">
                  <div>first_name,last_name,email,roles,teams</div>
                  <div>John,Doe,john@example.com,Agent,Sales</div>
                  <div>Jane,Smith,jane@example.com,Admin,Support</div>
                  <div>Bob,Test,bob@example.com,"Agent,Admin",Support</div>
                </div>
                <p class="text-xs mt-2 text-muted-foreground">
                  {{ $t('importer.agentCaseSensitiveNote') }}
                </p>
              </AlertDescription>
            </Alert>
          </div>

          <!-- Loading spinner -->
          <div v-if="loading" class="flex justify-center py-8">
            <Spinner />
          </div>

          <!-- Logs -->
          <div v-if="status?.logs?.some((l) => l && l.trim())" class="space-y-4">
            <div class="space-y-2">
              <p class="text-sm font-medium">{{ $t('globals.terms.log', 2) }}</p>
              <Card class="p-0 overflow-hidden">
                <div class="relative">
                  <CopyButton :text="status.logs.join('\n')" class="absolute top-2 right-2 z-10" />
                  <div
                    class="bg-black text-white p-4 text-xs font-mono min-h-24 max-h-60 overflow-y-auto space-y-1 logs-scroll-container"
                  >
                    <div v-for="(log, idx) in status.logs.filter((l) => l && l.trim())" :key="idx">
                      {{ log }}
                    </div>
                  </div>
                </div>
              </Card>
            </div>
          </div>

          <Alert v-if="error" variant="destructive">
            <AlertTitle>{{ $t('globals.terms.error') }}</AlertTitle>
            <AlertDescription>{{ error }}</AlertDescription>
          </Alert>
        </div>

        <DialogFooter>
          <Button v-if="complete" @click="resetAndClose">
            {{ $t('globals.messages.close') }}
          </Button>
          <template v-else>
            <Button variant="outline" @click="closeDialog" :disabled="loading || status?.running">
              {{ $t('globals.messages.cancel') }}
            </Button>
            <Button @click="startImport" :disabled="!file || loading || status?.running">
              {{ $t('globals.terms.import') }}
            </Button>
          </template>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter
} from '@/components/ui/dialog'
import { Upload } from 'lucide-vue-next'
import { Spinner } from '@/components/ui/spinner'
import CopyButton from '@/components/button/CopyButton.vue'
import { handleHTTPError } from '@/utils/http'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'

const { t } = useI18n()
const emitter = useEmitter()

const props = defineProps({
  entityKey: { type: String, required: true },
  uploadFn: { type: Function, required: true },
  getStatusFn: { type: Function, required: true }
})

const showDialog = ref(false)
const file = ref(null)
const loading = ref(false)
const status = ref(null)
const error = ref('')
const pollInterval = ref(null)

const complete = computed(() => status.value && !status.value.running)

const emit = defineEmits(['import-complete'])

const onFileSelect = (e) => {
  file.value = e.target.files[0]
  error.value = ''
}

const openDialog = async () => {
  showDialog.value = true
  loading.value = true

  // Check if import already running
  try {
    const res = await props.getStatusFn()
    if (res.data.data?.running) {
      startPolling(res.data.data)
      return
    }
  } catch {
    // no existing import
  }
  loading.value = false
}

const startImport = async () => {
  if (!file.value) return

  error.value = ''
  loading.value = true

  const formData = new FormData()
  formData.append('file', file.value)
  try {
    await props.uploadFn(formData)
    startPolling()
  } catch (err) {
    error.value = handleHTTPError(err).message
    loading.value = false
    status.value = null
  }
}

const fetchStatus = async () => {
  try {
    const res = await props.getStatusFn()
    status.value = res.data.data

    // Auto-scroll logs to bottom
    scrollLogsToBottom()

    if (!status.value.running) {
      stopPolling()
    }
  } catch (err) {
    if (err.response?.status !== 404) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(err).message
      })
    }
  }
}

const scrollLogsToBottom = () => {
  nextTick(() => {
    const logsContainer = document.querySelector('.logs-scroll-container')
    if (logsContainer) {
      logsContainer.scrollTop = logsContainer.scrollHeight
    }
  })
}

const startPolling = async (initialStatus = null) => {
  if (initialStatus) {
    status.value = initialStatus
  } else {
    await fetchStatus()
  }
  loading.value = false
  pollInterval.value = setInterval(fetchStatus, 1000)
}

const stopPolling = () => {
  if (pollInterval.value) {
    clearInterval(pollInterval.value)
    pollInterval.value = null
  }
}

const closeDialog = () => {
  showDialog.value = false
}

const resetAndClose = () => {
  stopPolling()
  resetState()
  showDialog.value = false
  emit('import-complete')
}

const resetState = () => {
  file.value = null
  loading.value = false
  status.value = null
  error.value = ''
}

onBeforeUnmount(() => {
  stopPolling()
})

// Reset state when dialog is closed
watch(showDialog, (open) => {
  if (!open) {
    stopPolling()
    resetState()
  }
})
</script>
