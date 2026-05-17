<template>
  <div class="min-h-screen flex flex-col">
    <!-- Main Content Area -->
    <div class="flex flex-wrap gap-4 pb-4">
      <div class="flex items-center gap-4 mb-4">
        <!-- Search Input -->
        <Input
          type="text"
          v-model="searchTerm"
          :placeholder="$t('contact.searchByEmail')"
          @input="fetchContactsDebounced"
        />

        <!-- Order By Popover -->
        <Popover>
          <PopoverTrigger>
            <Button variant="outline" size="sm" class="flex items-center h-8">
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
                <SelectItem :value="'users.created_at'">{{ $t('globals.terms.createdAt') }}</SelectItem>
                <SelectItem :value="'users.email'">{{ $t('globals.terms.email') }}</SelectItem>
              </SelectContent>
            </Select>

            <!-- order by direction -->
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
        <Card
          v-for="contact in contacts"
          :key="contact.id"
          class="p-4 w-full hover:bg-accent/50 cursor-pointer"
          @click="$router.push({ name: 'contact-detail', params: { id: contact.id } })"
        >
          <div class="flex items-center gap-4">
            <Avatar class="h-10 w-10 border">
              <AvatarImage :src="contact.avatar_url || ''" />
              <AvatarFallback class="text-sm font-medium">
                {{ getInitials(contact.first_name, contact.last_name) }}
              </AvatarFallback>
            </Avatar>

            <div class="space-y-1 overflow-hidden flex-1">
              <div class="flex items-center gap-2">
                <h4 class="text-sm font-semibold truncate">
                  {{ contact.first_name }} {{ contact.last_name }}
                </h4>
                <Badge v-if="contact.type" variant="secondary" class="text-xs px-1.5 py-0">
                  {{ contact.type === 'visitor' ? $t('contact.type.visitor') : $t('contact.type.contact') }}
                </Badge>
              </div>
              <p class="text-xs text-muted-foreground truncate">
                {{ contact.email }}
              </p>
              <div v-if="contact.external_user_id" class="flex items-center gap-1 text-xs text-muted-foreground">
                <IdCardIcon size="12" class="flex-shrink-0" />
                <span class="truncate">{{ contact.external_user_id }}</span>
              </div>
            </div>
          </div>
        </Card>
        <div v-if="contacts.length === 0" class="flex items-center justify-center w-full h-32">
          <p class="text-lg text-muted-foreground">{{ $t('contact.noContactsFound') }}</p>
        </div>
      </template>
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
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { ArrowDownWideNarrow, IdCardIcon } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { useDebounceFn } from '@vueuse/core'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
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
const emitter = useEmitter()

const fetchContactsDebounced = useDebounceFn(() => {
  fetchContacts()
}, 300)

const fetchContacts = async () => {
  loading.value = true
  let filterJSON = ''
  if (searchTerm.value && searchTerm.value.length > 3) {
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
    contacts.value = response.data.data.results
    totalPages.value = response.data.data.total_pages
    total.value = response.data.data.total
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
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
