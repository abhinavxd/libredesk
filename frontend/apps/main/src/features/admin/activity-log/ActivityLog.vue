<template>
  <div class="min-h-screen flex flex-col">
    <div class="flex flex-wrap gap-4 pb-4">
      <div class="flex items-center gap-2 mb-4">
        <!-- Filter Popover -->
        <Popover :open="filtersOpen" @update:open="filtersOpen = $event">
          <PopoverTrigger @click="filtersOpen = !filtersOpen">
            <Button variant="outline" size="sm" class="flex items-center gap-2 h-8">
              <ListFilter size="14" />
              <span>Filter</span>
              <span
                v-if="filters.length > 0"
                class="flex items-center justify-center bg-primary text-primary-foreground rounded-full size-4 text-xs"
              >
                {{ filters.length }}
              </span>
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-full p-4 flex flex-col gap-4">
            <div class="w-[32rem]">
              <FilterBuilder
                :fields="filterFields"
                :showButtons="true"
                v-model="filters"
                @apply="fetchActivityLogs"
                @clear="fetchActivityLogs"
              />
            </div>
          </PopoverContent>
        </Popover>

        <!-- Order By Popover -->
        <Popover>
          <PopoverTrigger>
            <Button variant="outline" size="sm" class="flex items-center h-8">
              <ArrowDownWideNarrow size="18" class="text-muted-foreground cursor-pointer" />
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-[200px] p-4 flex flex-col gap-4">
            <!-- order by field -->
            <Select v-model="orderByField" @update:model-value="fetchActivityLogs">
              <SelectTrigger class="h-8 w-full">
                <SelectValue :placeholder="orderByField" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem :value="'activity_logs.created_at'">
                  {{ t('globals.terms.createdAt') }}
                </SelectItem>
              </SelectContent>
            </Select>

            <!-- order by direction -->
            <Select v-model="orderByDirection" @update:model-value="fetchActivityLogs">
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

      <div class="w-full overflow-x-auto">
        <SimpleTable
          :headers="[
            t('globals.terms.name'),
            t('globals.terms.timestamp'),
            t('globals.terms.ipAddress')
          ]"
          :keys="['activity_description', 'created_at', 'ip']"
          :data="activityLogs"
          :showDelete="false"
          :loading="loading"
          :skeletonRows="15"
        />
      </div>
    </div>

    <!-- TODO: deduplicate this code, copied from contacts list -->
    <div class="sticky bottom-0 bg-background p-4 mt-auto">
      <div class="flex flex-col sm:flex-row items-center justify-between gap-4">
        <div class="flex items-center gap-2">
          <span class="text-sm text-muted-foreground">
            {{ t('globals.terms.page') }} {{ page }} of {{ totalPages }}
          </span>
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
import { ref, computed, onMounted, watch } from 'vue'
import SimpleTable from '@main/components/table/SimpleTable.vue'
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
import FilterBuilder from '@main/components/filter/FilterBuilder.vue'
import { Button } from '@shared-ui/components/ui/button'
import { ListFilter, ArrowDownWideNarrow } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { useActivityLogFilters } from '../../../composables/useActivityLogFilters'
import { useI18n } from 'vue-i18n'
import { format } from 'date-fns'
import { getVisiblePages } from '../../../utils/pagination'
import api from '../../../api'

const activityLogs = ref([])
const { t } = useI18n()
const loading = ref(true)
const page = ref(1)
const perPage = ref(15)
const orderByField = ref('activity_logs.created_at')
const orderByDirection = ref('desc')
const totalCount = ref(0)
const totalPages = ref(0)
const filters = ref([])
const filtersOpen = ref(false)
const { activityLogListFilters } = useActivityLogFilters()

const filterFields = computed(() =>
  Object.entries(activityLogListFilters.value).map(([field, value]) => ({
    model: 'activity_logs',
    label: value.label,
    field,
    type: value.type,
    operators: value.operators,
    options: value.options ?? []
  }))
)

const visiblePages = computed(() => getVisiblePages(page.value, totalPages.value))

async function fetchActivityLogs() {
  filtersOpen.value = false
  loading.value = true
  try {
    const resp = await api.getActivityLogs({
      page: page.value,
      page_size: perPage.value,
      filters: JSON.stringify(filters.value),
      order: orderByDirection.value,
      order_by: orderByField.value
    })
    activityLogs.value = resp.data.data.results
    totalCount.value = resp.data.data.count
    totalPages.value = resp.data.data.total_pages

    // Format the created_at field
    activityLogs.value = activityLogs.value.map((log) => ({
      ...log,
      created_at: format(new Date(log.created_at), 'PPpp')
    }))
  } catch (err) {
    console.error('Error fetching activity logs:', err)
    activityLogs.value = []
    totalCount.value = 0
  } finally {
    loading.value = false
  }
}

function goToPage(p) {
  if (p >= 1 && p <= totalPages.value && p !== page.value) {
    page.value = p
  }
}

function handlePerPageChange() {
  page.value = 1
  fetchActivityLogs()
}

watch([page, perPage, orderByField, orderByDirection], fetchActivityLogs)

onMounted(fetchActivityLogs)
</script>
