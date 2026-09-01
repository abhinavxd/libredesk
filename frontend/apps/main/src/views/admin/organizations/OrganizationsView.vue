<template>
  <div>
    <AdminSplitLayout>
      <template #content>
        <LoadingOverlay :loading="isLoading" reserve-height>
          <div class="flex justify-between mb-5">
            <div class="flex justify-end mb-4 w-full gap-2">
              <Dialog v-model:open="dialogOpen">
                <DialogTrigger as-child @click="newOrg">
                  <Button>{{ t('organization.new') }}</Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[480px]">
                  <DialogHeader>
                    <DialogTitle class="mb-1">
                      {{ isEditing ? t('organization.edit') : t('organization.new') }}
                    </DialogTitle>
                    <DialogDescription>
                      {{ t('organization.formDescription') }}
                    </DialogDescription>
                  </DialogHeader>
                  <form class="space-y-4" @submit.prevent="onSubmit">
                    <div class="space-y-1">
                      <label class="text-sm font-medium">{{ t('globals.terms.name') }}</label>
                      <Input v-model="form.name" type="text" required />
                    </div>
                    <div class="space-y-1">
                      <label class="text-sm font-medium">{{ t('organization.domains') }}</label>
                      <Input
                        v-model="form.domainsText"
                        type="text"
                        :placeholder="t('organization.domainsPlaceholder')"
                      />
                      <p class="text-xs text-muted-foreground">{{ t('organization.domainsHint') }}</p>
                    </div>
                    <div class="space-y-1">
                      <label class="text-sm font-medium">{{ t('globals.terms.note', 2) }}</label>
                      <Input v-model="form.notes" type="text" />
                    </div>
                    <DialogFooter class="mt-6">
                      <Button type="submit">
                        {{ isEditing ? t('globals.messages.save') : t('globals.messages.create') }}
                      </Button>
                    </DialogFooter>
                  </form>
                </DialogContent>
              </Dialog>
            </div>
          </div>
          <div class="space-y-2">
            <div
              v-for="org in orgs"
              :key="org.id"
              class="flex items-center justify-between rounded-md border px-4 py-3"
            >
              <div class="min-w-0">
                <button class="font-medium hover:underline text-left" @click="editOrg(org)">
                  {{ org.name }}
                </button>
                <p class="text-xs text-muted-foreground truncate">
                  {{ (org.domains || []).join(', ') || t('organization.noDomains') }}
                </p>
              </div>
              <Button variant="ghost" size="sm" class="text-destructive" @click="confirmDelete(org)">
                {{ t('globals.messages.delete') }}
              </Button>
            </div>
            <p v-if="!isLoading && orgs.length === 0" class="text-sm text-muted-foreground">
              {{ t('organization.empty') }}
            </p>
          </div>
        </LoadingOverlay>
      </template>
      <template #help>
        <p>{{ $t('admin.organization.help') }}</p>
      </template>
    </AdminSplitLayout>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog/index.js'
import { useEmitter } from '../../../composables/useEmitter.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import api from '../../../api/index.js'

const { t } = useI18n()
const isLoading = ref(false)
const orgs = ref([])
const dialogOpen = ref(false)
const isEditing = ref(false)
const editingId = ref(null)
const emitter = useEmitter()
const form = ref({ name: '', domainsText: '', notes: '' })

onMounted(getOrgs)

async function getOrgs() {
  isLoading.value = true
  try {
    const resp = await api.getOrganizations()
    orgs.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

function newOrg() {
  editingId.value = null
  isEditing.value = false
  form.value = { name: '', domainsText: '', notes: '' }
}

function editOrg(org) {
  editingId.value = org.id
  isEditing.value = true
  form.value = {
    name: org.name,
    domainsText: (org.domains || []).join(', '),
    notes: org.notes || ''
  }
  dialogOpen.value = true
}

function parseDomains(text) {
  return text
    .split(/[,\s]+/)
    .map((d) => d.trim().replace(/^@/, ''))
    .filter(Boolean)
}

async function onSubmit() {
  isLoading.value = true
  const payload = {
    name: form.value.name.trim(),
    domains: parseDomains(form.value.domainsText),
    notes: form.value.notes
  }
  try {
    if (isEditing.value) {
      await api.updateOrganization(editingId.value, payload)
    } else {
      await api.createOrganization(payload)
    }
    dialogOpen.value = false
    await getOrgs()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

async function confirmDelete(org) {
  if (!window.confirm(t('organization.deleteConfirmation'))) return
  try {
    await api.deleteOrganization(org.id)
    await getOrgs()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
