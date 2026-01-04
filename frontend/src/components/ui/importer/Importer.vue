<template>
  <div>
    <Button variant="secondary" @click="showDialog = true">
      Import Agents
    </Button>

    <Dialog v-model:open="showDialog">
      <DialogContent class="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Import Agents</DialogTitle>
        </DialogHeader>

        <div class="space-y-4 py-4">
          <!-- File Upload Section -->
          <div v-if="!importing && !complete" class="space-y-4">
            <div class="space-y-2">
              <label class="text-sm font-medium">Select CSV file</label>
              <div 
                @click="$refs.fileInput.click()"
                class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground"
              >
                <span class="flex-1 truncate" :class="!file && 'text-muted-foreground'">
                  {{ file ? file.name : 'Choose a CSV file...' }}
                </span>
                <svg 
                  class="h-5 w-5 text-muted-foreground" 
                  fill="none" 
                  stroke="currentColor" 
                  viewBox="0 0 24 24"
                >
                  <path 
                    stroke-linecap="round" ``
                    stroke-linejoin="round" 
                    stroke-width="2" 
                    d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                  />
                </svg>
              </div>
              <input 
                type="file" 
                accept=".csv"
                @change="onFileSelect"
                ref="fileInput"
                class="hidden"
              />
            </div>

            <AlertTitle>Required CSV format</AlertTitle>
            <Alert>
              <AlertDescription>
                <p class="text-xs mb-2">Example CSV:</p>
                <div class="bg-muted p-2 rounded text-xs font-mono overflow-x-auto">
                  <div>first_name,last_name,email,roles,teams</div>
                  <div>John,Doe,john@example.com,Agent,Sales</div>
                  <div>Jane,Smith,jane@example.com,Admin,Support</div>
                  <div>Bob,Test,bob@example.com,"Agent,Admin",Support</div>
                </div>
                <p class="text-xs mt-2 text-muted-foreground">
                  Roles and teams must match database values exactly (case-sensitive)
                </p>
              </AlertDescription>
            </Alert>

            <Button 
              @click="startImport" 
              :disabled="!file" 
              class="w-full"
            >
              Start Import
            </Button>
          </div>

          <!-- Progress Section -->
          <div v-if="status" class="space-y-4">
            <div v-if="importing" class="flex items-center gap-2">
              <Spinner class="h-4 w-4" />
              <span class="text-sm">Importing agents...</span>
            </div>

            <Alert 
              v-if="complete" 
              class="bg-green-50 dark:bg-green-950 border-green-200"
            >
              <AlertTitle class="text-green-600">Success!</AlertTitle>
              <AlertDescription class="text-green-600">
                Import completed: {{ status.success }} successful, 
                {{ status.errors }} failed out of {{ status.total }} total
              </AlertDescription>
            </Alert>

            <!-- Logs -->
           <div>
              <p class="text-sm font-medium mb-2">Import logs</p>
              <Card class="p-3">
                <div class="bg-black text-white p-3 rounded-md text-xs font-mono max-h-60 overflow-y-auto space-y-1 logs-scroll-container">
                  <div v-for="(log, idx) in status.logs" :key="idx">
                    {{ log }}
                  </div>
                </div>
              </Card>
            </div>
          </div>

          <Alert v-if="error" variant="destructive">
            <AlertTitle>Error</AlertTitle>
            <AlertDescription>{{ error }}</AlertDescription>
          </Alert>
        </div>

        <DialogFooter>
          <Button v-if="complete" @click="resetAndClose">
            Done
          </Button>
          <Button 
            variant="outline" 
            @click="closeDialog" 
            :disabled="importing"
          >
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, onBeforeUnmount } from 'vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card } from '@/components/ui/card'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Spinner } from '@/components/ui/spinner'
import { 
  Dialog, 
  DialogContent, 
  DialogHeader, 
  DialogTitle, 
  DialogFooter 
} from '@/components/ui/dialog'
import axios from 'axios'

const showDialog = ref(false)
const file = ref(null)
const importing = ref(false)
const complete = ref(false)
const status = ref(null)
const error = ref('')
const pollInterval = ref(null)

const emit = defineEmits(['import-complete'])

const getCSRFToken = () => {
  const name = 'csrf_token='
  const cookies = document.cookie.split(';')
  for (let i = 0; i < cookies.length; i++) {
    let c = cookies[i].trim()
    if (c.indexOf(name) === 0) {
      return c.substring(name.length, c.length)
    }
  }
  return ''
}

const onFileSelect = (e) => {
  file.value = e.target.files[0]
  error.value = ''
}

const startImport = async () => {
  if (!file.value) return

  error.value = ''
  importing.value = true
  
  // Initialize empty status to show logs area immediately
  status.value = {
    running: true,
    logs: ['Uploading CSV file...'],
    total: 0,
    success: 0,
    errors: 0
  }

  const formData = new FormData()
  formData.append('file', file.value)

  try {
    await axios.post('/api/v1/agents/import', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
        'X-CSRFTOKEN': getCSRFToken()
      }
    })

    // Update log after successful upload
    status.value.logs.push('CSV uploaded successfully, starting import...')
    startPolling()
  } catch (err) {
    error.value = err.response?.data?.message || err.message || 'Upload failed'
    importing.value = false
    status.value = null
  }
}

const fetchStatus = async () => {
  try {
    const res = await axios.get('/api/v1/agents/import/status')
    status.value = res.data.data

    // Auto-scroll logs to bottom
    scrollLogsToBottom()

    if (!status.value.running) {
      stopPolling()
      importing.value = false
      complete.value = true
    }
  } catch (err) {
    if (err.response?.status !== 404) {
      console.error('Poll error:', err)
    }
  }
}

const scrollLogsToBottom = () => {
  // Use nextTick to ensure DOM is updated before scrolling
  import('vue').then(({ nextTick }) => {
    nextTick(() => {
      const logsContainer = document.querySelector('.logs-scroll-container')
      if (logsContainer) {
        logsContainer.scrollTop = logsContainer.scrollHeight
      }
    })
  })
}

const startPolling = () => {
  pollInterval.value = setInterval(fetchStatus, 1000)
}

const stopPolling = () => {
  if (pollInterval.value) {
    clearInterval(pollInterval.value)
    pollInterval.value = null
  }
}

const closeDialog = () => {
  if (importing.value && !confirm('Import in progress. Close?')) return
  stopPolling()
  resetState()
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
  importing.value = false
  complete.value = false
  status.value = null
  error.value = ''
}

onBeforeUnmount(() => {
  stopPolling()
})
</script>