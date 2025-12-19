<script setup>
import { ref, onMounted } from 'vue'
import { toast } from 'vue-sonner'
import api from '@/api'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import {
  Database, RefreshCw, Plus, Trash2, Globe, MessageSquare, FileText,
  AlertCircle, CheckCircle, Clock, Search, Upload
} from 'lucide-vue-next'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

const sources = ref([])
const loading = ref(true)
const syncing = ref({})

// Test Query state
const testQuery = ref('')
const testResults = ref([])
const testLoading = ref(false)

// Dialog state
const showAddDialog = ref(false)
const editingSource = ref(null)
const formData = ref({
  name: '',
  source_type: 'macro',
  enabled: true,
  urls: ''
})
const saving = ref(false)

// File upload state
const showUploadDialog = ref(false)
const uploadFile = ref(null)
const uploadName = ref('')
const uploading = ref(false)
const fileInputRef = ref(null)

async function fetchSources() {
  loading.value = true
  try {
    const res = await api.getRAGSources()
    sources.value = res.data.data || []
  } catch (err) {
    console.error('Error fetching sources:', err)
    toast.error('Failed to load knowledge sources')
  } finally {
    loading.value = false
  }
}

async function runTestQuery() {
  if (!testQuery.value.trim()) {
    toast.error('Please enter a query')
    return
  }

  testLoading.value = true
  testResults.value = []
  try {
    const res = await api.ragSearch({
      query: testQuery.value,
      limit: 5,
      threshold: 0.25
    })
    testResults.value = res.data.data || []
    if (testResults.value.length === 0) {
      toast.info('No matching documents found')
    }
  } catch (err) {
    console.error('Error running query:', err)
    toast.error(err.response?.data?.message || 'Failed to run query')
  } finally {
    testLoading.value = false
  }
}

function openAddDialog() {
  editingSource.value = null
  formData.value = {
    name: '',
    source_type: 'macro',
    enabled: true,
    urls: ''
  }
  showAddDialog.value = true
}

function openEditDialog(source) {
  editingSource.value = source
  formData.value = {
    name: source.name,
    source_type: source.source_type,
    enabled: source.enabled,
    urls: source.config?.urls?.join('\n') || ''
  }
  showAddDialog.value = true
}

async function saveSource() {
  if (!formData.value.name.trim()) {
    toast.error('Name is required')
    return
  }

  saving.value = true
  try {
    const config = {}
    if (formData.value.source_type === 'webpage') {
      config.urls = formData.value.urls.split('\n').filter(u => u.trim())
    }

    const data = {
      name: formData.value.name,
      source_type: formData.value.source_type,
      enabled: formData.value.enabled,
      config
    }

    if (editingSource.value) {
      await api.updateRAGSource(editingSource.value.id, data)
      toast.success('Source updated')
    } else {
      await api.createRAGSource(data)
      toast.success('Source created')
    }

    showAddDialog.value = false
    await fetchSources()
  } catch (err) {
    toast.error(err.response?.data?.message || 'Failed to save')
  } finally {
    saving.value = false
  }
}

async function deleteSource(source) {
  if (!confirm(`Delete "${source.name}"? This will remove all indexed documents.`)) {
    return
  }

  try {
    await api.deleteRAGSource(source.id)
    toast.success('Source deleted')
    await fetchSources()
  } catch (err) {
    toast.error(err.response?.data?.message || 'Failed to delete')
  }
}

async function syncSource(source) {
  syncing.value[source.id] = true
  try {
    await api.syncRAGSource(source.id)
    toast.success('Sync started')
    // Poll for completion after a delay
    setTimeout(() => fetchSources(), 3000)
  } catch (err) {
    toast.error(err.response?.data?.message || 'Failed to start sync')
  } finally {
    syncing.value[source.id] = false
  }
}

function getSourceIcon(type) {
  switch (type) {
    case 'macro': return MessageSquare
    case 'webpage': return Globe
    case 'file': return FileText
    default: return FileText
  }
}

function formatDate(date) {
  if (!date) return 'Never'
  return new Date(date).toLocaleString()
}

function formatScore(score) {
  return (score * 100).toFixed(1) + '%'
}

// File upload functions
function openUploadDialog() {
  uploadFile.value = null
  uploadName.value = ''
  showUploadDialog.value = true
}

function handleFileSelect(event) {
  const file = event.target.files?.[0]
  if (file) {
    uploadFile.value = file
    // Pre-fill name with filename (without extension)
    uploadName.value = file.name.replace(/\.[^/.]+$/, '')
  }
}

function triggerFileInput() {
  fileInputRef.value?.click()
}

async function uploadFileSource() {
  if (!uploadFile.value) {
    toast.error('Please select a file')
    return
  }

  // Validate file type
  const validTypes = ['.txt', '.csv', '.json']
  const ext = '.' + uploadFile.value.name.split('.').pop().toLowerCase()
  if (!validTypes.includes(ext)) {
    toast.error('Unsupported file type. Only .txt, .csv, and .json files are allowed.')
    return
  }

  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', uploadFile.value)
    if (uploadName.value.trim()) {
      formData.append('name', uploadName.value.trim())
    }
    formData.append('enabled', 'true')

    await api.ragFileUpload(formData)
    toast.success('File uploaded and indexing started')
    showUploadDialog.value = false
    await fetchSources()
  } catch (err) {
    console.error('Upload error:', err)
    toast.error(err.response?.data?.message || 'Failed to upload file')
  } finally {
    uploading.value = false
  }
}

onMounted(() => {
  fetchSources()
})
</script>

<template>
  <AdminPageWithHelp>
    <template #content>
      <div class="space-y-6">
        <!-- Test Query Card -->
        <Card>
          <CardHeader>
            <div class="flex items-center gap-2">
              <Search class="h-5 w-5" />
              <CardTitle>Test Knowledge Base</CardTitle>
            </div>
            <CardDescription>
              Test what content will be retrieved for a given query. This shows what context the AI will use when generating responses.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="flex gap-2">
              <Input
                v-model="testQuery"
                placeholder="Enter a test query..."
                class="flex-1"
                @keyup.enter="runTestQuery"
              />
              <Button @click="runTestQuery" :disabled="testLoading">
                <Search v-if="!testLoading" class="w-4 h-4 mr-2" />
                <RefreshCw v-else class="w-4 h-4 mr-2 animate-spin" />
                Search
              </Button>
            </div>

            <!-- Results -->
            <div v-if="testResults.length > 0" class="space-y-3">
              <Label>Results ({{ testResults.length }} matches)</Label>
              <div class="space-y-2 max-h-[400px] overflow-y-auto">
                <div
                  v-for="(result, index) in testResults"
                  :key="index"
                  class="p-3 border rounded-lg bg-muted/50"
                >
                  <div class="flex items-center justify-between mb-2">
                    <Badge variant="outline">{{ result.source_type || 'document' }}</Badge>
                    <Badge class="bg-green-100 text-green-800">
                      {{ formatScore(result.similarity) }} match
                    </Badge>
                  </div>
                  <p class="font-medium text-sm mb-1">{{ result.title }}</p>
                  <p class="text-sm whitespace-pre-wrap text-muted-foreground">{{ result.content?.substring(0, 500) }}{{ result.content?.length > 500 ? '...' : '' }}</p>
                </div>
              </div>
            </div>

            <div v-else-if="testQuery && !testLoading" class="text-sm text-muted-foreground text-center py-4">
              Run a search to see matching knowledge base content
            </div>
          </CardContent>
        </Card>

        <!-- Knowledge Sources -->
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold">Knowledge Sources</h2>
            <p class="text-sm text-muted-foreground">
              Configure sources for AI-powered response generation
            </p>
          </div>
          <div class="flex gap-2">
            <Button variant="outline" @click="openUploadDialog">
              <Upload class="w-4 h-4 mr-2" />
              Upload File
            </Button>
            <Button @click="openAddDialog">
              <Plus class="w-4 h-4 mr-2" />
              Add Source
            </Button>
          </div>
        </div>

        <div v-if="loading" class="flex justify-center py-8">
          <Spinner />
        </div>

        <div v-else-if="sources.length === 0" class="text-center py-8 text-muted-foreground">
          <Database class="w-12 h-12 mx-auto mb-4 opacity-50" />
          <p>No knowledge sources configured</p>
          <p class="text-sm">Add a source to enable AI-powered responses</p>
        </div>

        <div v-else class="grid gap-4">
          <Card v-for="source in sources" :key="source.id">
            <CardHeader class="pb-3">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <component :is="getSourceIcon(source.source_type)" class="w-5 h-5 text-muted-foreground" />
                  <div>
                    <CardTitle class="text-base">{{ source.name }}</CardTitle>
                    <CardDescription>
                      <Badge variant="outline" class="mr-2">{{ source.source_type }}</Badge>
                      <Badge v-if="source.enabled" variant="default" class="bg-green-500">Enabled</Badge>
                      <Badge v-else variant="secondary">Disabled</Badge>
                    </CardDescription>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <Button
                    v-if="source.source_type !== 'file'"
                    variant="outline"
                    size="sm"
                    @click="syncSource(source)"
                    :disabled="syncing[source.id]"
                  >
                    <RefreshCw :class="['w-4 h-4 mr-1', syncing[source.id] ? 'animate-spin' : '']" />
                    Sync
                  </Button>
                  <Button v-if="source.source_type !== 'file'" variant="outline" size="sm" @click="openEditDialog(source)">
                    Edit
                  </Button>
                  <Button variant="destructive" size="sm" @click="deleteSource(source)">
                    <Trash2 class="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div class="flex items-center gap-4 text-sm text-muted-foreground">
                <div class="flex items-center gap-1">
                  <Clock class="w-4 h-4" />
                  Last synced: {{ formatDate(source.last_synced_at) }}
                </div>
                <div v-if="source.source_type === 'file' && source.config?.filename" class="flex items-center gap-1">
                  <FileText class="w-4 h-4" />
                  {{ source.config.filename }}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <!-- Add/Edit Dialog -->
      <Dialog :open="showAddDialog" @update:open="showAddDialog = $event">
        <DialogContent class="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{{ editingSource ? 'Edit' : 'Add' }} Knowledge Source</DialogTitle>
            <DialogDescription>
              Configure a source for indexing into the knowledge base
            </DialogDescription>
          </DialogHeader>

          <div class="space-y-4 py-4">
            <div class="space-y-2">
              <Label>Name</Label>
              <Input v-model="formData.name" placeholder="My Knowledge Source" />
            </div>

            <div class="space-y-2">
              <Label>Type</Label>
              <Select v-model="formData.source_type" :disabled="!!editingSource">
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="macro">Macros (Saved Replies)</SelectItem>
                  <SelectItem value="webpage">Web Pages</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div v-if="formData.source_type === 'webpage'" class="space-y-2">
              <Label>URLs (one per line)</Label>
              <Textarea
                v-model="formData.urls"
                placeholder="https://example.com/help&#10;https://example.com/faq"
                rows="4"
              />
            </div>

            <div class="flex items-center gap-2">
              <Switch v-model:checked="formData.enabled" />
              <Label>Enabled</Label>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" @click="showAddDialog = false">Cancel</Button>
            <Button @click="saveSource" :disabled="saving">
              <Spinner v-if="saving" class="w-4 h-4 mr-2" />
              {{ editingSource ? 'Save' : 'Create' }}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <!-- File Upload Dialog -->
      <Dialog :open="showUploadDialog" @update:open="showUploadDialog = $event">
        <DialogContent class="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Upload Knowledge File</DialogTitle>
            <DialogDescription>
              Upload a TXT, CSV, or JSON file to index into the knowledge base
            </DialogDescription>
          </DialogHeader>

          <div class="space-y-4 py-4">
            <div class="space-y-2">
              <Label>File</Label>
              <input
                ref="fileInputRef"
                type="file"
                accept=".txt,.csv,.json"
                class="hidden"
                @change="handleFileSelect"
              />
              <div
                class="border-2 border-dashed rounded-lg p-6 text-center cursor-pointer hover:border-primary transition-colors"
                @click="triggerFileInput"
              >
                <Upload class="w-8 h-8 mx-auto mb-2 text-muted-foreground" />
                <p v-if="!uploadFile" class="text-sm text-muted-foreground">
                  Click to select a file or drag and drop
                </p>
                <p v-else class="text-sm font-medium">
                  {{ uploadFile.name }}
                </p>
                <p class="text-xs text-muted-foreground mt-1">
                  Supported: .txt, .csv, .json
                </p>
              </div>
            </div>

            <div class="space-y-2">
              <Label>Name (optional)</Label>
              <Input v-model="uploadName" placeholder="Knowledge source name" />
              <p class="text-xs text-muted-foreground">
                Leave empty to use filename
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" @click="showUploadDialog = false">Cancel</Button>
            <Button @click="uploadFileSource" :disabled="uploading || !uploadFile">
              <Spinner v-if="uploading" class="w-4 h-4 mr-2" />
              <Upload v-else class="w-4 h-4 mr-2" />
              Upload
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </template>

    <template #help>
      <h4 class="font-medium mb-2">Knowledge Sources</h4>
      <p class="text-sm text-muted-foreground mb-4">
        Knowledge sources provide context for AI-generated responses.
      </p>

      <h5 class="font-medium mb-1">Test Query</h5>
      <p class="text-sm text-muted-foreground mb-4">
        Use the test query panel to see exactly what content will be retrieved for any query.
        This helps you understand and refine your knowledge base.
      </p>

      <h5 class="font-medium mb-1">Source Types</h5>
      <ul class="text-sm text-muted-foreground list-disc pl-4 space-y-1 mb-4">
        <li><strong>Macros</strong> - Indexes your saved replies</li>
        <li><strong>Web Pages</strong> - Fetches and indexes web content</li>
        <li><strong>Files</strong> - Upload TXT, CSV, or JSON files</li>
      </ul>

      <h5 class="font-medium mb-1">File Upload</h5>
      <p class="text-sm text-muted-foreground mb-2">
        Upload files to add knowledge:
      </p>
      <ul class="text-sm text-muted-foreground list-disc pl-4 space-y-1 mb-4">
        <li><strong>.txt</strong> - Plain text files, split into chunks</li>
        <li><strong>.csv</strong> - Each row becomes a document</li>
        <li><strong>.json</strong> - Array of objects or nested data</li>
      </ul>

      <h5 class="font-medium mb-1">How It Works</h5>
      <p class="text-sm text-muted-foreground">
        When you click "Generate Response", the AI searches the knowledge base
        for relevant content and uses it to craft a helpful reply.
      </p>
    </template>
  </AdminPageWithHelp>
</template>
