<template>
  <div class="flex flex-col h-full min-h-0">
    <div class="flex items-center gap-3 px-4 py-3 border-b shrink-0">
      <Input
        type="text"
        v-model="searchTerm"
        :placeholder="$t('contact.searchPlaceholder')"
        class="h-8 max-w-xs"
        @input="fetchContactsDebounced"
      />

      <Popover>
        <PopoverTrigger>
          <Button variant="outline" size="sm" class="flex items-center h-8">
            <ArrowDownWideNarrow size="16" class="text-muted-foreground" />
          </Button>
        </PopoverTrigger>
        <PopoverContent class="w-[200px] p-4 flex flex-col gap-4">
          <Select v-model="orderByField" @update:model-value="fetchContacts">
            <SelectTrigger class="h-8 w-full">
              <SelectValue :placeholder="orderByField" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="'users.created_at'">{{ $t('globals.terms.createdAt') }}</SelectItem>
              <SelectItem :value="'users.email'">{{ $t('globals.terms.email') }}</SelectItem>
              <SelectItem :value="'users.first_name'">{{ $t('globals.terms.name') }}</SelectItem>
            </SelectContent>
          </Select>

          <Select v-model="orderByDirection" @update:model-value="fetchContacts">
            <SelectTrigger class="h-8 w-full">
              <SelectValue :placeholder="orderByDirection" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="'asc'">{{ $t('globals.terms.ascending') }}</SelectItem>
              <SelectItem :value="'desc'">{{ $t('globals.terms.descending') }}</SelectItem>
            </SelectContent>
          </Select>
        </PopoverContent>
      </Popover>

      <p v-if="!loading && total > 0" class="ml-auto text-xs text-muted-foreground tabular-nums">
        {{ t('contact.count', total, { count: total }) }}
      </p>
    </div>

    <div class="flex-1 min-h-0 overflow-auto">
      <table class="w-full text-sm">
        <thead class="sticky top-0 z-10 bg-background">
          <tr class="border-b text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            <th class="px-4 py-2 font-semibold">{{ $t('globals.terms.name') }}</th>
            <th class="px-4 py-2 font-semibold">{{ $t('globals.terms.email') }}</th>
            <th class="px-4 py-2 w-28 font-semibold">{{ $t('globals.terms.type') }}</th>
            <th class="px-4 py-2 w-28 font-semibold">{{ $t('globals.terms.createdAt') }}</th>
          </tr>
        </thead>
        <tbody v-if="loading">
          <tr v-for="i in perPage" :key="i" class="border-b border-border/50">
            <td class="px-4 py-2">
              <div class="flex items-center gap-3">
                <Skeleton class="h-8 w-8 rounded-full" />
                <Skeleton class="h-3 w-40" />
              </div>
            </td>
            <td class="px-4 py-2"><Skeleton class="h-3 w-48" /></td>
            <td class="px-4 py-2"><Skeleton class="h-3 w-16" /></td>
            <td class="px-4 py-2"><Skeleton class="h-3 w-16" /></td>
          </tr>
        </tbody>
        <tbody v-else-if="contacts.length">
          <tr
            v-for="contact in contacts"
            :key="contact.id"
            class="border-b border-border/50 cursor-pointer hover:bg-muted/40"
            @click="$router.push({ name: 'contact-detail', params: { id: contact.id } })"
          >
            <td class="px-4 py-2">
              <div class="flex items-center gap-3 min-w-0">
                <Avatar class="h-8 w-8 border shrink-0">
                  <AvatarImage :src="contact.avatar_url || ''" />
                  <AvatarFallback class="text-xs font-medium">
                    {{ getInitials(contact.first_name, contact.last_name) }}
                  </AvatarFallback>
                </Avatar>
                <div class="min-w-0">
                  <p class="font-medium truncate">
                    {{ contact.first_name }} {{ contact.last_name }}
                  </p>
                  <p v-if="contact.external_user_id" class="text-xs text-muted-foreground truncate">
                    {{ contact.external_user_id }}
                  </p>
                </div>
              </div>
            </td>
            <td class="px-4 py-2 text-muted-foreground truncate max-w-[18rem]">
              {{ contact.email }}
            </td>
            <td class="px-4 py-2">
              <Badge v-if="contact.type" variant="secondary" class="text-xs px-1.5 py-0">
                {{ contact.type === 'visitor' ? $t('contact.type.visitor') : $t('contact.type.contact') }}
              </Badge>
            </td>
            <td class="px-4 py-2 text-muted-foreground whitespace-nowrap tabular-nums">
              {{ contact.created_at ? formatShortDate(contact.created_at) : '' }}
            </td>
          </tr>
        </tbody>
        <tbody v-else>
          <tr>
            <td colspan="4" class="px-4 py-16 text-center text-muted-foreground">
              {{ $t('contact.noContactsFound') }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <PaginationBar
      v-model:page="page"
      v-model:per-page="perPage"
      :total-pages="totalPages"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { ArrowDownWideNarrow } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { useDebounceFn } from '@vueuse/core'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { formatShortDate } from '@shared-ui/utils/datetime.js'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
import api from '@main/api'

const { t } = useI18n()
const contacts = ref([])
const loading = ref(false)
const page = ref(1)
const perPage = ref(50)
const totalPages = ref(0)
const searchTerm = ref('')
const orderByField = ref('users.created_at')
const orderByDirection = ref('desc')
const total = ref(0)
const emitter = useEmitter()
let fetchRequestId = 0

const fetchContactsDebounced = useDebounceFn(() => {
  if (page.value === 1) {
    fetchContacts()
  } else {
    page.value = 1
  }
}, 300)

const searchFilters = () => {
  const q = searchTerm.value.trim()
  if (q.length < 2) return ''
  return JSON.stringify({
    logic: 'OR',
    rules: [
      { model: 'users', field: 'email', operator: 'ilike', value: q },
      { model: 'users', field: 'first_name', operator: 'ilike', value: q },
      { model: 'users', field: 'last_name', operator: 'ilike', value: q }
    ]
  })
}

const fetchContacts = async () => {
  const requestId = ++fetchRequestId
  loading.value = true
  try {
    const response = await api.getContacts({
      page: page.value,
      page_size: perPage.value,
      filters: searchFilters(),
      order: orderByDirection.value,
      order_by: orderByField.value
    })
    if (requestId !== fetchRequestId) return
    contacts.value = response.data.data.results
    totalPages.value = response.data.data.total_pages
    total.value = response.data.data.total
  } catch (error) {
    if (requestId !== fetchRequestId) return
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    if (requestId === fetchRequestId) loading.value = false
  }
}

const getInitials = (firstName, lastName) => {
  return `${firstName?.[0] || ''}${lastName?.[0] || ''}`.toUpperCase()
}

watch([page, perPage], fetchContacts)

onMounted(() => {
  fetchContacts()
})
</script>
