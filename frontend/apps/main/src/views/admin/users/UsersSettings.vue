<template>
  <div class="flex min-h-0 flex-1 flex-col gap-4">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <p class="max-w-2xl text-sm text-muted-foreground">{{ $t('users.description') }}</p>
      <Button @click="openAdd"><UserPlus class="size-4" />{{ $t('users.add') }}</Button>
    </div>

    <LoadingOverlay :loading="loading" reserve-height>
      <div class="grid gap-4 md:grid-cols-[minmax(16rem,22rem)_minmax(0,1fr)]">
        <!-- People -->
        <section class="flex min-h-0 flex-col overflow-hidden rounded-lg border bg-card" :aria-label="$t('users.title')">
          <div class="border-b p-2">
            <label class="relative block">
              <span class="sr-only">{{ $t('users.search') }}</span>
              <Search class="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
              <input
                v-model="query"
                type="search"
                class="w-full rounded-md border bg-background py-2 pl-8 pr-3 text-sm"
                :placeholder="$t('users.search')"
              />
            </label>
          </div>
          <ul class="max-h-[40vh] min-h-0 overflow-y-auto p-1 md:max-h-[calc(100dvh-15rem)]" role="listbox" :aria-label="$t('users.title')">
            <li v-for="user in visibleUsers" :key="user.id">
              <button
                type="button"
                role="option"
                :aria-selected="user.id === selectedID"
                class="flex w-full items-center gap-3 rounded-md px-2.5 py-2 text-left transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                :class="{ 'bg-accent': user.id === selectedID }"
                @click="requestSelect(user.id)"
              >
                <span
                  class="flex size-9 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold text-muted-foreground"
                  :class="{ 'opacity-50': !user.enabled }"
                  aria-hidden="true"
                  >{{ initials(user) }}</span
                >
                <span class="min-w-0 flex-1">
                  <span class="flex items-center gap-1.5">
                    <span class="truncate text-sm font-medium">{{ fullName(user) || user.email }}</span>
                    <span v-if="user.id === selfID" class="shrink-0 text-xs text-muted-foreground">({{ $t('users.you') }})</span>
                  </span>
                  <span class="block truncate text-xs text-muted-foreground">{{ user.email }}</span>
                </span>
                <span class="flex shrink-0 flex-col items-end gap-1">
                  <Badge :variant="roleVariant(primaryRole(user.roles))" class="font-medium">{{ primaryRole(user.roles) || '—' }}</Badge>
                  <span v-if="!user.enabled" class="text-[11px] text-muted-foreground">{{ $t('users.disabled') }}</span>
                  <span v-else class="text-[11px] text-muted-foreground">{{ addressSummary(user) }}</span>
                </span>
              </button>
            </li>
            <li v-if="!visibleUsers.length" class="px-3 py-6 text-center text-sm text-muted-foreground">{{ $t('users.none') }}</li>
          </ul>
        </section>

        <!-- Access editor -->
        <section v-if="selectedUser" class="space-y-6 rounded-lg border bg-card p-5" :aria-label="fullName(selectedUser)">
          <header class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0">
              <h2 class="truncate text-lg font-semibold">{{ fullName(selectedUser) || selectedUser.email }}</h2>
              <p class="truncate text-sm text-muted-foreground">{{ selectedUser.email }}</p>
            </div>
            <Button
              v-if="selectedUser.id !== selfID"
              variant="ghost"
              size="sm"
              class="text-destructive hover:text-destructive"
              @click="confirmDelete = true"
            >
              <Trash2 class="size-4" />{{ $t('users.delete') }}
            </Button>
          </header>

          <p v-if="isSelf" class="rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-sm">{{ $t('users.selfLocked') }}</p>

          <fieldset class="space-y-2" :disabled="saving">
            <legend class="mb-2 text-sm font-medium">{{ $t('users.role') }}</legend>
            <div class="grid gap-2 sm:grid-cols-3">
              <label
                v-for="role in roleOptions"
                :key="role"
                class="flex cursor-pointer flex-col gap-1 rounded-md border p-3 text-sm transition-colors has-[:checked]:border-primary has-[:checked]:bg-primary/5 has-[:disabled]:cursor-not-allowed has-[:disabled]:opacity-60"
              >
                <span class="flex items-center gap-2 font-medium">
                  <input v-model="draft.role" type="radio" name="role" :value="role" class="accent-primary" :disabled="isSelf" />
                  {{ role }}
                </span>
                <span v-if="roleHelp(role)" class="text-xs text-muted-foreground">{{ roleHelp(role) }}</span>
              </label>
            </div>
            <p v-if="legacyRoles(selectedUser.roles).length" class="text-xs text-muted-foreground">
              {{ $t('users.legacyRole', { roles: legacyRoles(selectedUser.roles).join(', ') }) }}
            </p>
          </fieldset>

          <fieldset class="space-y-2" :disabled="saving">
            <legend class="text-sm font-medium">{{ $t('users.addresses') }}</legend>
            <p class="text-xs text-muted-foreground">
              {{ draft.role === 'Admin' ? $t('users.adminSeesAll') : $t('users.addressesHelp') }}
            </p>
            <div v-if="!addresses.length" class="rounded-md border border-dashed p-4 text-sm text-muted-foreground">{{ $t('users.noAddresses') }}</div>
            <div v-else class="max-h-72 overflow-y-auto rounded-md border" :class="{ 'opacity-70': draft.role === 'Admin' }">
              <label
                v-for="address in addresses"
                :key="address.id"
                class="flex cursor-pointer items-start gap-3 border-b px-3 py-2.5 text-sm last:border-b-0 hover:bg-accent/50"
              >
                <input v-model="draft.address_ids" type="checkbox" :value="address.id" class="mt-1 accent-primary" />
                <span class="min-w-0 flex-1">
                  <span class="block truncate font-medium">{{ address.display_name || address.address }}</span>
                  <span v-if="address.display_name" class="block truncate text-xs text-muted-foreground">{{ address.address }}</span>
                </span>
                <span v-if="!address.enabled" class="shrink-0 text-xs text-muted-foreground">{{ $t('globals.terms.disabled') }}</span>
              </label>
            </div>
          </fieldset>

          <label class="flex items-center justify-between gap-4 rounded-md border p-3 text-sm" :class="{ 'opacity-60': isSelf }">
            <span>
              <span class="block font-medium">{{ $t('users.enabled') }}</span>
              <span class="block text-xs text-muted-foreground">{{ $t('users.enabledHelp') }}</span>
            </span>
            <input v-model="draft.enabled" type="checkbox" role="switch" class="size-4 accent-primary" :disabled="isSelf || saving" />
          </label>

          <div class="flex justify-end gap-2 border-t pt-4">
            <Button variant="outline" :disabled="!dirty || saving" @click="resetDraft">{{ $t('users.reset') }}</Button>
            <Button :disabled="!dirty || saving" @click="confirmSave = true">{{ saving ? $t('users.saving') : $t('users.save') }}</Button>
          </div>
        </section>
        <section v-else class="flex min-h-48 items-center justify-center rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
          {{ $t('users.select') }}
        </section>
      </div>
    </LoadingOverlay>

    <!-- Confirm save -->
    <AlertDialog :open="confirmSave" @update:open="confirmSave = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ $t('users.confirmTitle', { name: selectedName }) }}</AlertDialogTitle>
          <AlertDialogDescription>{{ $t('users.confirmDescription') }}</AlertDialogDescription>
        </AlertDialogHeader>
        <ul class="space-y-1.5 text-sm">
          <li v-for="(change, index) in changes" :key="index" class="rounded-md bg-muted/50 px-3 py-2">{{ describeChange(change) }}</li>
        </ul>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction @click="save">{{ $t('users.confirmSave') }}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Confirm delete -->
    <AlertDialog :open="confirmDelete" @update:open="confirmDelete = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ $t('users.deleteTitle', { name: selectedName }) }}</AlertDialogTitle>
          <AlertDialogDescription>{{ $t('users.deleteDescription') }}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" @click="removeUser">{{ $t('users.delete') }}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Confirm discarding unsaved changes -->
    <AlertDialog :open="pendingSelectID !== null" @update:open="!$event && (pendingSelectID = null)">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ $t('users.unsavedTitle') }}</AlertDialogTitle>
          <AlertDialogDescription>{{ $t('users.unsavedDescription', { name: selectedName }) }}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('users.keepEditing') }}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" @click="select(pendingSelectID)">{{ $t('users.discardChanges') }}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Add user -->
    <Dialog :open="addOpen" @update:open="addOpen = $event">
      <DialogContent class="max-h-[90dvh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t('users.addTitle') }}</DialogTitle>
          <DialogDescription>{{ $t('users.addDescription') }}</DialogDescription>
        </DialogHeader>
        <form id="add-user-form" class="space-y-4" @submit.prevent="createUser">
          <div class="grid gap-3 sm:grid-cols-2">
            <label class="block space-y-1.5 text-sm">
              <span class="font-medium">{{ $t('users.firstName') }}</span>
              <input v-model.trim="newUser.first_name" required maxlength="140" class="w-full rounded-md border bg-background px-3 py-2" autocomplete="off" />
            </label>
            <label class="block space-y-1.5 text-sm">
              <span class="font-medium">{{ $t('users.lastName') }}</span>
              <input v-model.trim="newUser.last_name" maxlength="140" class="w-full rounded-md border bg-background px-3 py-2" autocomplete="off" />
            </label>
          </div>
          <label class="block space-y-1.5 text-sm">
            <span class="font-medium">{{ $t('users.email') }}</span>
            <input v-model.trim="newUser.email" type="email" required class="w-full rounded-md border bg-background px-3 py-2" autocomplete="off" />
          </label>
          <fieldset class="space-y-2 text-sm">
            <legend class="mb-1 font-medium">{{ $t('users.role') }}</legend>
            <label v-for="role in BUILT_IN_ROLES" :key="role" class="flex cursor-pointer items-start gap-2">
              <input v-model="newUser.role" type="radio" name="new-role" :value="role" class="mt-1 accent-primary" />
              <span><span class="font-medium">{{ role }}</span> <span class="text-muted-foreground">— {{ roleHelp(role) }}</span></span>
            </label>
          </fieldset>
          <fieldset v-if="addresses.length" class="space-y-2 text-sm">
            <legend class="mb-1 font-medium">{{ $t('users.addresses') }}</legend>
            <div class="max-h-48 overflow-y-auto rounded-md border">
              <label v-for="address in addresses" :key="address.id" class="flex cursor-pointer items-center gap-3 border-b px-3 py-2 last:border-b-0">
                <input v-model="newUser.address_ids" type="checkbox" :value="address.id" class="accent-primary" />
                <span class="truncate">{{ address.display_name ? `${address.display_name} · ${address.address}` : address.address }}</span>
              </label>
            </div>
          </fieldset>
          <label class="flex cursor-pointer items-center gap-2 text-sm">
            <input v-model="newUser.send_welcome_email" type="checkbox" class="accent-primary" />
            {{ $t('users.sendWelcome') }}
          </label>
        </form>
        <DialogFooter>
          <Button variant="outline" type="button" @click="addOpen = false">{{ $t('globals.messages.cancel') }}</Button>
          <Button type="submit" form="add-user-form" :disabled="creating">{{ creating ? $t('users.adding') : $t('users.add') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Search, Trash2, UserPlus } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
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
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@shared-ui/components/ui/dialog'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import {
  BUILT_IN_ROLES,
  accessDraft,
  describeAccessChanges,
  filterUsers,
  fullName,
  initials,
  legacyRoles,
  primaryRole
} from '@main/utils/user-access'
import api from '@main/api'

const { t } = useI18n()
const emitter = useEmitter()

const loading = ref(true)
const saving = ref(false)
const creating = ref(false)
const users = ref([])
const addresses = ref([])
const selfID = ref(null)
const query = ref('')
const selectedID = ref(null)
const pendingSelectID = ref(null)
const draft = ref({ role: '', address_ids: [], enabled: true })
const confirmSave = ref(false)
const confirmDelete = ref(false)
const addOpen = ref(false)
const emptyNewUser = () => ({ first_name: '', last_name: '', email: '', role: 'Agent', address_ids: [], send_welcome_email: true })
const newUser = ref(emptyNewUser())

const toast = (description, variant) => emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant, description })
const toastError = (error) => toast(handleHTTPError(error).message, 'destructive')

const visibleUsers = computed(() => filterUsers(users.value, query.value))
const selectedUser = computed(() => users.value.find((user) => user.id === selectedID.value) || null)
const selectedName = computed(() => fullName(selectedUser.value) || selectedUser.value?.email || '')
const isSelf = computed(() => selectedUser.value?.id === selfID.value)
const addressByID = computed(() => new Map(addresses.value.map((address) => [address.id, address])))
const addressLabel = (id) => {
  const address = addressByID.value.get(id)
  return address ? address.display_name || address.address : `#${id}`
}
const changes = computed(() => (selectedUser.value ? describeAccessChanges(selectedUser.value, draft.value, addressLabel) : []))
const dirty = computed(() => changes.value.length > 0)
// Built-in roles first; a legacy role stays selectable only for the person who already holds it.
const roleOptions = computed(() => {
  const current = primaryRole(selectedUser.value?.roles)
  return BUILT_IN_ROLES.includes(current) || !current ? BUILT_IN_ROLES : [...BUILT_IN_ROLES, current]
})

const roleHelp = (role) =>
  ({ Admin: t('users.roleAdminHelp'), Agent: t('users.roleAgentHelp'), Contributor: t('users.roleContributorHelp') })[role] || ''
const roleVariant = (role) => (role === 'Admin' ? 'default' : role === 'Agent' ? 'secondary' : 'outline')
const addressSummary = (user) =>
  primaryRole(user.roles) === 'Admin' ? t('users.allAddresses') : t('users.addressCount', (user.address_ids || []).length)

function describeChange(change) {
  switch (change.kind) {
    case 'role':
      return t('users.changeRole', { from: change.from, to: change.to })
    case 'added':
      return t('users.changeAdded', { list: change.list.join(', ') })
    case 'removed':
      return t('users.changeRemoved', { list: change.list.join(', ') })
    case 'enabled':
      return t('users.changeEnable')
    default:
      return t('users.changeDisable')
  }
}

function resetDraft() {
  draft.value = accessDraft(selectedUser.value)
}

function select(id) {
  pendingSelectID.value = null
  selectedID.value = id
  resetDraft()
}

function requestSelect(id) {
  if (id === selectedID.value) return
  if (dirty.value) {
    pendingSelectID.value = id
    return
  }
  select(id)
}

function replaceUser(updated) {
  const index = users.value.findIndex((user) => user.id === updated.id)
  if (index === -1) users.value = [...users.value, updated]
  else users.value.splice(index, 1, updated)
}

async function load() {
  loading.value = true
  try {
    const data = (await api.getManagedUsers()).data?.data || {}
    users.value = data.users || []
    addresses.value = data.addresses || []
    selfID.value = data.self_id ?? null
    if (!selectedUser.value && users.value.length) select(users.value[0].id)
    else if (selectedUser.value) resetDraft()
  } catch (error) {
    toastError(error)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!selectedUser.value) return
  saving.value = true
  const name = selectedName.value
  try {
    const response = await api.updateManagedUserAccess(selectedUser.value.id, {
      role: draft.value.role,
      address_ids: draft.value.address_ids.map(Number),
      enabled: draft.value.enabled
    })
    replaceUser(response.data.data)
    resetDraft()
    toast(t('users.saved', { name }))
  } catch (error) {
    toastError(error)
  } finally {
    saving.value = false
  }
}

async function removeUser() {
  const user = selectedUser.value
  if (!user) return
  const name = selectedName.value
  try {
    await api.deleteManagedUser(user.id)
    users.value = users.value.filter((entry) => entry.id !== user.id)
    selectedID.value = null
    if (users.value.length) select(users.value[0].id)
    toast(t('users.deleted', { name }))
  } catch (error) {
    toastError(error)
  }
}

function openAdd() {
  newUser.value = emptyNewUser()
  addOpen.value = true
}

async function createUser() {
  creating.value = true
  try {
    const response = await api.createManagedUser({ ...newUser.value, address_ids: newUser.value.address_ids.map(Number) })
    const created = response.data.data
    replaceUser(created)
    addOpen.value = false
    query.value = ''
    select(created.id)
    toast(t('users.created', { name: fullName(created) || created.email }))
  } catch (error) {
    toastError(error)
  } finally {
    creating.value = false
  }
}

// Keep the selection valid when a search hides it.
watch(visibleUsers, (list) => {
  if (!dirty.value && list.length && !list.some((user) => user.id === selectedID.value)) select(list[0].id)
})

onMounted(load)
</script>
