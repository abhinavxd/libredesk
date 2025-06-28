<template>
  <div class="min-h-screen flex flex-col">
    <!-- Main Content Area -->
    <div class="flex flex-wrap gap-4 pb-4">
      <div class="flex items-center gap-4 mb-4">
        <!-- Search Input -->
        <Input
          type="text"
          v-model="searchTerm"
          placeholder="Search by email"
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
                <SelectItem :value="'users.created_at'">Created at</SelectItem>
                <SelectItem :value="'users.email'">Email</SelectItem>
              </SelectContent>
            </Select>

            <!-- order by direction -->
            <Select v-model="orderByDirection" @update:model-value="fetchContacts">
              <SelectTrigger class="h-8 w-full">
                <SelectValue :placeholder="orderByDirection" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem :value="'asc'">Ascending</SelectItem>
                <SelectItem :value="'desc'">Descending</SelectItem>
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

            <div class="space-y-1 overflow-hidden">
              <h4 class="text-sm font-semibold truncate">
                {{ contact.first_name }} {{ contact.last_name }}
              </h4>
              <p class="text-xs text-muted-foreground truncate">
                {{ contact.email }}
              </p>
            </div>
          </div>
        </Card>
        <div v-if="contacts.length === 0" class="flex items-center justify-center w-full h-32">
          <p class="text-lg text-muted-foreground">No contacts found</p>
        </div>
      </template>
    </div>

    <!-- Sticky Pagination Controls -->
    <div class="sticky bottom-0 bg-background p-4 mt-auto">
      <div class="flex flex-col sm:flex-row items-center justify-between gap-4">
        <div class="flex items-center gap-2">
          <span class="text-sm text-muted-foreground"> Page {{ page }} of {{ totalPages }} </span>
          <Select v-model="perPage" @update:model-value="handlePerPageChange">
            <SelectTrigger class="h-8 w-[70px]">
              <SelectValue :placeholder="perPage" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="15">15</SelectItem>
              <SelectItem :value="30">30</SelectItem>
              <SelectItem :value="50">50</SelectItem>
              <SelectItem :value="100">100</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <Pagination>
          <PaginationList class="flex items-center gap-1">
            <PaginationListItem>
              <PaginationFirst
                :class="{ 'cursor-not-allowed opacity-50': page === 1 }"
                @click.prevent="page > 1 ? goToPage(1) : null"
              />
            </PaginationListItem>

            <PaginationListItem>
              <PaginationPrev
                :class="{ 'cursor-not-allowed opacity-50': page === 1 }"
                @click.prevent="page > 1 ? goToPage(page - 1) : null"
              />
            </PaginationListItem>

            <template v-for="pageNumber in visiblePages" :key="pageNumber">
              <PaginationListItem v-if="pageNumber === '...'">
                <PaginationEllipsis />
              </PaginationListItem>
              <PaginationListItem v-else>
                <Button
                  :is-active="pageNumber === page"
                  @click.prevent="goToPage(pageNumber)"
                  :variant="pageNumber === page ? 'default' : 'outline'"
                >
                  {{ pageNumber }}
                </Button>
              </PaginationListItem>
            </template>

            <PaginationListItem>
              <PaginationNext
                :class="{ 'cursor-not-allowed opacity-50': page === totalPages }"
                @click.prevent="page < totalPages ? goToPage(page + 1) : null"
              />
            </PaginationListItem>

            <PaginationListItem>
              <PaginationLast
                :class="{ 'cursor-not-allowed opacity-50': page === totalPages }"
                @click.prevent="page < totalPages ? goToPage(totalPages) : null"
              />
            </PaginationListItem>
          </PaginationList>
        </Pagination>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Card } from '@shared-ui/components/ui/card'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import {
  Pagination,
  PaginationEllipsis,
  PaginationFirst,
  PaginationLast,
  PaginationList,
  PaginationListItem,
  PaginationNext,
  PaginationPrev
} from '@shared-ui/components/ui/pagination'
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
import { debounce } from '../../utils/debounce'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { useEmitter } from '../../composables/useEmitter'
import { handleHTTPError } from '../../utils/http'
import { getVisiblePages } from '../../utils/pagination'
import api from '../../api'

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

// Google-style pagination
const visiblePages = computed(() => getVisiblePages(page.value, totalPages.value))

const fetchContactsDebounced = debounce(() => {
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

const goToPage = (newPage) => {
  page.value = newPage
  fetchContacts()
}

const handlePerPageChange = (newPerPage) => {
  page.value = 1
  perPage.value = newPerPage
  fetchContacts()
}

onMounted(() => {
  fetchContacts()
})
</script>
