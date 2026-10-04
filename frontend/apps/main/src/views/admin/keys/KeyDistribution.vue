<template>
  <AdminSplitLayout>
    <template #content>
      <div class="space-y-6 pb-8">
        <div>
          <h1 class="text-xl font-semibold">{{ $t('keyDistribution.title') }}</h1>
          <p class="mt-1 text-sm text-muted-foreground">{{ $t('keyDistribution.description') }}</p>
        </div>

        <p v-if="loadError" role="alert" class="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
          {{ loadError }}
        </p>

        <label class="flex items-start gap-3">
          <Checkbox
            class="mt-0.5"
            :checked="saved.enabled"
            :disabled="!loaded || savingEnabled"
            @update:checked="setEnabled"
          />
          <span>
            <span class="block text-sm font-medium">{{ $t('keyDistribution.enable') }}</span>
            <span class="block text-xs text-muted-foreground">{{ $t('keyDistribution.enableHint') }}</span>
          </span>
        </label>

        <!-- Everything below is inert while Key Distribution is off. -->
        <fieldset
          :disabled="!saved.enabled"
          class="min-w-0 space-y-6 transition-opacity"
          :class="{ 'pointer-events-none select-none opacity-50': !saved.enabled }"
          :aria-disabled="!saved.enabled"
        >
          <form class="flex flex-wrap items-end gap-4" @submit.prevent="saveLimits">
            <label class="block w-40 space-y-1.5">
              <span class="text-sm text-muted-foreground">{{ $t('keyDistribution.maxPerEmail') }}</span>
              <Input v-model.number="form.max_keys_per_email" type="number" min="1" :max="MAX_PER_EMAIL" step="1" required />
            </label>
            <label class="block w-40 space-y-1.5">
              <span class="text-sm text-muted-foreground">{{ $t('keyDistribution.lowStock') }}</span>
              <Input v-model.number="form.low_stock_threshold" type="number" min="0" step="1" required />
            </label>
            <Button type="submit" size="sm" variant="outline" :disabled="!limitsChanged || savingLimits">
              {{ $t('globals.messages.save') }}
            </Button>
          </form>

          <div class="flex flex-wrap items-end gap-2">
            <label class="block min-w-48 flex-1 space-y-1.5">
              <span class="text-sm text-muted-foreground">{{ $t('keyDistribution.game') }}</span>
              <NativeSelect v-model.number="appId" :disabled="!apps.length">
                <option v-if="!apps.length" :value="0">{{ $t('keyDistribution.noGames') }}</option>
                <option v-for="app in apps" :key="app.id" :value="app.id">
                  {{ app.archived ? $t('keyDistribution.gameArchived', { name: app.name }) : app.name }}
                </option>
              </NativeSelect>
            </label>
            <Button type="button" variant="outline" @click="openGameDialog('create')">
              <Plus class="size-4" />{{ $t('keyDistribution.newGame') }}
            </Button>
            <template v-if="app">
              <Button type="button" variant="ghost" @click="openGameDialog('rename')">
                {{ $t('keyDistribution.rename') }}
              </Button>
              <Button type="button" variant="ghost" :disabled="busy" @click="toggleArchived">
                {{ app.archived ? $t('keyDistribution.restore') : $t('keyDistribution.archive') }}
              </Button>
            </template>
          </div>

          <template v-if="app">
            <p v-if="app.archived" class="text-sm text-muted-foreground">{{ $t('keyDistribution.archivedNotice') }}</p>
            <p v-else-if="lowStock" class="flex items-center gap-1.5 text-sm text-warning-600" role="status">
              <TriangleAlert class="size-4 shrink-0" aria-hidden="true" />
              {{ $t('keyDistribution.lowStockNotice', { game: app.name, count: app.redeemable }, app.redeemable) }}
            </p>

            <div class="grid gap-4 lg:grid-cols-2">
              <KeyPool
                status="redeemable"
                :title="$t('keyDistribution.pool.redeemable')"
                :app-id="app.id"
                :enabled="saved.enabled"
                :revealed-id="revealed.id"
                :revealed-key="revealed.key"
                :refresh-key="refreshKey"
                @reveal="toggleReveal"
                @void="voidTarget = $event"
              />
              <KeyPool
                status="activated"
                :title="$t('keyDistribution.pool.activated')"
                :app-id="app.id"
                :enabled="saved.enabled"
                :revealed-id="revealed.id"
                :revealed-key="revealed.key"
                :refresh-key="refreshKey"
                @reveal="toggleReveal"
              />
            </div>

            <form v-if="!app.archived" class="space-y-2" @submit.prevent="importKeys">
              <label class="block space-y-1.5">
                <span class="text-sm font-medium">{{ $t('keyDistribution.addKeys', { game: app.name }) }}</span>
                <Textarea
                  v-model="pasted"
                  rows="4"
                  class="font-mono text-sm"
                  autocomplete="off"
                  spellcheck="false"
                  :placeholder="$t('keyDistribution.addKeysPlaceholder')"
                />
              </label>
              <div class="flex flex-wrap items-center gap-3">
                <Button type="submit" size="sm" :disabled="!pasted.trim() || importing">{{ $t('keyDistribution.addKeysButton') }}</Button>
                <p v-if="importSummary" class="text-sm text-muted-foreground" role="status">{{ importSummary }}</p>
              </div>
            </form>
          </template>
        </fieldset>
      </div>
    </template>
    <template #help>
      <p>{{ $t('keyDistribution.help') }}</p>
      <p>{{ $t('keyDistribution.helpReveal') }}</p>
    </template>
  </AdminSplitLayout>

  <Dialog :open="gameDialog.open" @update:open="gameDialog.open = $event">
    <DialogContent class="sm:max-w-md">
      <form class="space-y-4" @submit.prevent="saveGame">
        <DialogHeader>
          <DialogTitle>{{ gameDialog.mode === 'create' ? $t('keyDistribution.newGame') : $t('keyDistribution.renameGame') }}</DialogTitle>
          <DialogDescription class="sr-only">{{ $t('keyDistribution.gameName') }}</DialogDescription>
        </DialogHeader>
        <label class="block space-y-1.5">
          <span class="text-sm text-muted-foreground">{{ $t('keyDistribution.gameName') }}</span>
          <Input v-model="gameDialog.name" maxlength="100" required autocomplete="off" />
        </label>
        <DialogFooter>
          <Button type="button" variant="outline" @click="gameDialog.open = false">{{ $t('globals.messages.cancel') }}</Button>
          <Button type="submit" :disabled="!gameDialog.name.trim() || busy">{{ $t('globals.messages.save') }}</Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>

  <AlertDialog :open="!!voidTarget" @update:open="(open) => !open && (voidTarget = null)">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('keyDistribution.voidTitle') }}</AlertDialogTitle>
        <AlertDialogDescription>{{ $t('keyDistribution.voidDescription') }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="voidKey">{{ $t('keyDistribution.pool.void') }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, TriangleAlert } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { Input } from '@shared-ui/components/ui/input'
import { NativeSelect } from '@shared-ui/components/ui/native-select'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@shared-ui/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import KeyPool from '@/features/admin/keys/KeyPool.vue'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import api from '@main/api'

const MAX_PER_EMAIL = 20
// A revealed key hides itself again after this long.
const REVEAL_MS = 30000

const { t } = useI18n()
const emitter = useEmitter()

const loaded = ref(false)
const loadError = ref('')
const saved = ref({ enabled: false, max_keys_per_email: 1, low_stock_threshold: 10 })
const form = reactive({ max_keys_per_email: 1, low_stock_threshold: 10 })
const savingEnabled = ref(false)
const savingLimits = ref(false)
const apps = ref([])
const appId = ref(0)
const busy = ref(false)
const refreshKey = ref(0)
const pasted = ref('')
const importing = ref(false)
const importSummary = ref('')
const voidTarget = ref(null)
const gameDialog = reactive({ open: false, mode: 'create', name: '' })
const revealed = ref({ id: 0, key: '' })
let revealTimer = null

const app = computed(() => apps.value.find((a) => a.id === appId.value))
const lowStock = computed(
  () => !!app.value && saved.value.low_stock_threshold > 0 && app.value.redeemable <= saved.value.low_stock_threshold
)
const limitsChanged = computed(
  () =>
    form.max_keys_per_email !== saved.value.max_keys_per_email ||
    form.low_stock_threshold !== saved.value.low_stock_threshold
)

const toastError = (err) =>
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(err).message })
const toast = (description) => emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description })

const applySettings = (settings) => {
  saved.value = settings
  form.max_keys_per_email = settings.max_keys_per_email
  form.low_stock_threshold = settings.low_stock_threshold
}

const loadApps = async () => {
  const response = await api.getKeyApps()
  apps.value = response.data.data
  if (!apps.value.some((a) => a.id === appId.value)) {
    appId.value = (apps.value.find((a) => !a.archived) || apps.value[0])?.id || 0
  }
}

const saveSettings = async (settings) => {
  const response = await api.updateKeyDistributionSettings(settings)
  applySettings(response.data.data)
}

const setEnabled = async (enabled) => {
  savingEnabled.value = true
  try {
    await saveSettings({ ...saved.value, enabled })
    hideKey()
  } catch (err) {
    toastError(err)
  } finally {
    savingEnabled.value = false
  }
}

const saveLimits = async () => {
  savingLimits.value = true
  try {
    await saveSettings({ ...saved.value, ...form })
    toast(t('keyDistribution.saved'))
  } catch (err) {
    toastError(err)
  } finally {
    savingLimits.value = false
  }
}

const openGameDialog = (mode) => {
  gameDialog.mode = mode
  gameDialog.name = mode === 'rename' ? app.value?.name || '' : ''
  gameDialog.open = true
}

const saveGame = async () => {
  busy.value = true
  try {
    if (gameDialog.mode === 'create') {
      const response = await api.createKeyApp({ name: gameDialog.name })
      await loadApps()
      appId.value = response.data.data.id
    } else {
      await api.updateKeyApp(app.value.id, { name: gameDialog.name, archived: app.value.archived })
      await loadApps()
    }
    gameDialog.open = false
  } catch (err) {
    toastError(err)
  } finally {
    busy.value = false
  }
}

const toggleArchived = async () => {
  busy.value = true
  try {
    await api.updateKeyApp(app.value.id, { name: app.value.name, archived: !app.value.archived })
    await loadApps()
  } catch (err) {
    toastError(err)
  } finally {
    busy.value = false
  }
}

const importKeys = async () => {
  importing.value = true
  importSummary.value = ''
  try {
    const response = await api.importActivationKeys(app.value.id, pasted.value)
    const { added, duplicates, invalid } = response.data.data
    const parts = [t('keyDistribution.importAdded', { count: added }, added)]
    if (duplicates) parts.push(t('keyDistribution.importDuplicates', { count: duplicates }, duplicates))
    if (invalid) parts.push(t('keyDistribution.importInvalid', { count: invalid }, invalid))
    importSummary.value = parts.join(' ')
    // Leave the text in place when something was rejected, so it can be fixed.
    if (!invalid) pasted.value = ''
    await refresh()
  } catch (err) {
    toastError(err)
  } finally {
    importing.value = false
  }
}

const hideKey = () => {
  clearTimeout(revealTimer)
  revealed.value = { id: 0, key: '' }
}

// One key at a time: revealing another hides the first.
const toggleReveal = async (key) => {
  if (revealed.value.id === key.id) {
    hideKey()
    return
  }
  hideKey()
  try {
    const response = await api.revealActivationKey(key.id)
    revealed.value = { id: key.id, key: response.data.data.key }
    revealTimer = setTimeout(hideKey, REVEAL_MS)
  } catch (err) {
    toastError(err)
  }
}

const voidKey = async () => {
  const target = voidTarget.value
  voidTarget.value = null
  if (!target) return
  try {
    await api.voidActivationKey(target.id)
    if (revealed.value.id === target.id) hideKey()
    await refresh()
  } catch (err) {
    toastError(err)
  }
}

const refresh = async () => {
  await loadApps()
  refreshKey.value++
}

watch(appId, hideKey)

onMounted(async () => {
  try {
    const [settings] = await Promise.all([api.getKeyDistributionSettings(), loadApps()])
    applySettings(settings.data.data)
    loaded.value = true
  } catch (err) {
    loadError.value = handleHTTPError(err).message
  }
})

onBeforeUnmount(hideKey)
</script>
