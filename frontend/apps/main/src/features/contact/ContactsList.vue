<template>
  <div class="min-h-full flex flex-col">
    <!-- Main Content Area -->
    <div class="flex flex-wrap gap-4 pb-4">
      <div class="flex flex-wrap items-center gap-3 mb-4 w-full">
        <!-- Search Input -->
        <SearchInput
          class="min-w-0 flex-1 max-w-sm"
          v-model="searchTerm"
          :placeholder="i18n._('Search by email')"
          :clear-label="i18n._('Clear search')"
          @update:model-value="fetchContactsDebounced"
        />

        <!-- Order By Popover -->
        <Popover>
          <PopoverTrigger as-child>
            <Button
              type="button"
              variant="outline"
              size="icon"
              class="size-9"
              :aria-label="i18n._('Sort by')"
            >
              <ArrowDownWideNarrow size="18" class="text-muted-foreground cursor-pointer" />
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-[200px] p-4 flex flex-col gap-4">
            <!-- order by field -->
            <Select v-model="orderByField" @update:model-value="fetchContacts">
              <SelectTrigger class="h-8 w-full">
                <SelectValue :placeholder="orderByField" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem :value="'users.created_at'">{{
                  i18n._('Created at')
                }}</SelectItem>
                <SelectItem :value="'users.email'">{{ i18n.ngettext('Email', 'Emails', 1) }}</SelectItem>
              </SelectContent>
            </Select>

            <!-- order by direction -->
            <Select v-model="orderByDirection" @update:model-value="fetchContacts">
              <SelectTrigger class="h-8 w-full">
                <SelectValue :placeholder="orderByDirection" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem :value="'asc'">{{ i18n._('Ascending') }}</SelectItem>
                <SelectItem :value="'desc'">{{ i18n._('Descending') }}</SelectItem>
              </SelectContent>
            </Select>
          </PopoverContent>
        </Popover>

        <Button
          v-if="userStore.can('contacts:write')"
          type="button"
          size="sm"
          class="flex items-center h-8 ml-auto"
          @click="showCreateDialog = true"
        >
          {{ i18n._('New contact') }}
        </Button>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex flex-col gap-4 w-full">
        <Card v-for="i in perPage" :key="i" class="p-4 flex-shrink-0">
          <div class="flex items-center gap-4">
            <Skeleton class="h-10 w-10 rounded-full" />
            <div class="space-y-2">
              <Skeleton class="h-3 w-[160px]" />
              <Skeleton class="h-3 w-[140px]" />
            </div>
          </div>
        </Card>
      </div>

      <!-- Loaded State -->
      <template v-else>
        <router-link
          v-for="contact in contacts"
          :key="contact.id"
          :to="{ name: 'contact-detail', params: { id: contact.id } }"
          class="block w-full rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <Card class="p-4 w-full transition-colors duration-150 hover:bg-accent/50">
            <div class="flex items-center gap-4">
              <Avatar class="h-10 w-10 border">
                <AvatarImage :src="contact.avatar_url || ''" />
                <AvatarFallback class="text-sm font-medium">
                  {{ getInitials(contact.first_name, contact.last_name) }}
                </AvatarFallback>
              </Avatar>

              <div class="space-y-1 overflow-hidden flex-1">
                <div class="flex items-center gap-2">
                  <h4 class="ui-subheading truncate">
                    {{ contact.first_name }} {{ contact.last_name }}
                  </h4>
                  <Badge v-if="contact.type" variant="secondary" class="text-xs px-1.5 py-0">
                    {{
                      contact.type === 'visitor'
                        ? i18n.pgettext('Contact type', 'Visitor')
                        : i18n.pgettext('Contact type', 'Contact')
                    }}
                  </Badge>
                </div>
                <p class="text-xs text-muted-foreground truncate">
                  {{ contact.email }}
                </p>
                <div
                  v-if="contact.external_user_id"
                  class="flex items-center gap-1 text-xs text-muted-foreground"
                >
                  <IdCardIcon size="12" class="flex-shrink-0" />
                  <span class="truncate">{{ contact.external_user_id }}</span>
                </div>
              </div>
            </div>
          </Card>
        </router-link>
        <div v-if="contacts.length === 0" class="flex items-center justify-center w-full h-32">
          <p class="text-lg text-muted-foreground">{{ i18n._('No contacts found') }}</p>
        </div>
      </template>
    </div>

    <PaginationBar v-model:page="page" v-model:per-page="perPage" :total-pages="totalPages" />
  </div>

  <CreateContactDialog v-model:open="showCreateDialog" @created="onContactCreated" />
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Card } from '@shared-ui/components/ui/card'
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
import SearchInput from '@shared-ui/components/SearchInput.vue'
import { Button } from '@shared-ui/components/ui/button'
import { ArrowDownWideNarrow, IdCardIcon } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { useDebounceFn } from '@vueuse/core'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
import CreateContactDialog from './CreateContactDialog.vue'
import { useUserStore } from '@main/stores/user'
import api from '@main/api'

const contacts = ref([])
const loading = ref(false)
const page = ref(1)
const perPage = ref(15)
const totalPages = ref(0)
const searchTerm = ref('')
const orderByField = ref('users.created_at')
const orderByDirection = ref('desc')
const total = ref(0)
const showCreateDialog = ref(false)
const emitter = useEmitter()
const router = useRouter()
const userStore = useUserStore()
let fetchRequestId = 0

const fetchContactsDebounced = useDebounceFn(() => {
  if (page.value === 1) {
    fetchContacts()
  } else {
    page.value = 1
  }
}, 300)

const fetchContacts = async () => {
  const requestId = ++fetchRequestId
  loading.value = true
  let filterJSON = ''
  if (searchTerm.value && searchTerm.value.length >= 3) {
    filterJSON = JSON.stringify([
      {
        model: 'users',
        field: 'email',
        operator: 'ilike',
        value: searchTerm.value
      }
    ])
  }
  try {
    const response = await api.getContacts({
      page: page.value,
      page_size: perPage.value,
      filters: filterJSON,
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

const onContactCreated = (contact) => {
  router.push({ name: 'contact-detail', params: { id: contact.id } })
}

watch([page, perPage], fetchContacts)

onMounted(() => {
  fetchContacts()
})
</script>
