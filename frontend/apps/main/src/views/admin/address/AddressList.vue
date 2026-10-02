<template>
  <LoadingOverlay :loading="loading" reserve-height>
    <div class="mb-5 flex justify-end">
      <router-link :to="{ name: 'new-address' }"><Button>{{ $t('address.new') }}</Button></router-link>
    </div>

    <div v-if="!addresses.length && !loading" class="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
      {{ $t('address.empty') }}
    </div>
    <div v-else class="overflow-hidden rounded-lg border">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-muted-foreground">
          <tr>
            <th class="px-4 py-3 font-medium">{{ $t('address.address') }}</th>
            <th class="px-4 py-3 font-medium">{{ $t('address.kind') }}</th>
            <th class="px-4 py-3 font-medium">{{ $t('address.access') }}</th>
            <th class="px-4 py-3 font-medium">{{ $t('globals.terms.status') }}</th>
            <th class="px-4 py-3" />
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="address in addresses" :key="address.id">
            <td class="px-4 py-3">
              <router-link :to="{ name: 'edit-address', params: { id: address.id } }" class="font-medium hover:underline">
                {{ address.address }}
              </router-link>
              <div v-if="address.display_name" class="mt-0.5 text-xs text-muted-foreground">{{ address.display_name }}</div>
            </td>
            <td class="px-4 py-3 capitalize">{{ address.kind }}</td>
            <td class="px-4 py-3">{{ accessSummary(address) }}</td>
            <td class="px-4 py-3"><Badge :variant="address.enabled ? 'success' : 'secondary'">{{ address.enabled ? $t('globals.terms.enabled') : $t('globals.terms.disabled') }}</Badge></td>
            <td class="px-4 py-3 text-right">
              <div class="inline-flex gap-1">
                <Button variant="ghost" size="icon" :aria-label="$t('globals.messages.edit')" @click="router.push({ name: 'edit-address', params: { id: address.id } })"><Pencil class="size-4" /></Button>
                <Button v-if="address.kind === 'alias'" variant="ghost" size="icon" class="text-destructive hover:text-destructive" :aria-label="$t('address.delete')" @click="removeAddress(address)"><Trash2 class="size-4" /></Button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </LoadingOverlay>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Pencil, Trash2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const router = useRouter()
const { t } = useI18n()
const emitter = useEmitter()
const loading = ref(false)
const addresses = ref([])

// Admins always have access; everyone else is granted it directly or by team.
function accessSummary(address) {
  const people = t('address.peopleCount', (address.user_ids || []).length)
  const teams = (address.team_ids || []).length
  return teams ? `${people} · ${t('address.teamCount', teams)}` : people
}

async function load() {
  loading.value = true
  try {
    addresses.value = (await api.getAdminAddresses()).data?.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  } finally {
    loading.value = false
  }
}

async function removeAddress(address) {
  if (!window.confirm(`Delete ${address.address}? Addresses with existing mail must be disabled instead.`)) return
  try {
    await api.deleteAddress(address.id)
    await load()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  }
}

onMounted(load)
</script>
