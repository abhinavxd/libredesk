<template>
  <div class="min-h-0 flex-1 flex flex-col">
    <div class="pb-4">
      <AdminToolbar class="mb-5">
        <template #left>
          <div class="flex items-center gap-2">
            <!-- Filter Popover -->
            <Popover :open="filtersOpen" @update:open="filtersOpen = $event">
              <PopoverTrigger as-child>
                <Button type="button" variant="outline" class="gap-2">
                  <ListFilter size="14" />
                  <span>{{ t('globals.terms.filter', 1) }}</span>
                  <span
                    v-if="filters.length > 0"
                    class="flex items-center justify-center bg-primary text-primary-foreground rounded-full size-4 text-xs"
                  >
                    {{ filters.length }}
                  </span>
                </Button>
              </PopoverTrigger>
              <PopoverContent class="w-full p-4 flex flex-col gap-4">
                <div class="w-[32rem] max-w-full">
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
              <PopoverTrigger as-child>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  :aria-label="t('globals.messages.sortBy')"
                >
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
                    <SelectItem :value="'asc'">{{ t('globals.terms.ascending') }}</SelectItem>
                    <SelectItem :value="'desc'">{{ t('globals.terms.descending') }}</SelectItem>
                  </SelectContent>
                </Select>
              </PopoverContent>
            </Popover>
          </div>
        </template>
      </AdminToolbar>

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

    <PaginationBar
      class="-mx-6 sm:-mx-8 -mb-6 sm:-mb-8 px-6 sm:px-8"
      v-model:page="page"
      v-model:per-page="perPage"
      :total-pages="totalPages"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'
import { useStorage } from '@vueuse/core'
import SimpleTable from '@main/components/table/SimpleTable.vue'
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
import { useActivityLogFilters } from '@main/composables/useActivityLogFilters'
import { useI18n } from 'vue-i18n'
import { format } from 'date-fns'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
import api from '@main/api'

const activityLogs = ref([])
const { t } = useI18n()
const loading = ref(true)
const page = ref(1)
const perPage = ref(15)
const orderByField = ref('activity_logs.created_at')
const orderByDirection = ref('desc')
const totalCount = ref(0)
const totalPages = ref(0)
const filters = useStorage('activity-log-filters', [])
const filtersOpen = ref(false)
const { activityLogListFilters } = useActivityLogFilters()

const filterFields = computed(() =>
  Object.entries(activityLogListFilters.value).map(([field, value]) => ({
    model: 'activity_logs',
    label: value.label,
    field,
    type: value.type,
    operators: value.operators,
    options: value.options ?? [],
    entity: value.entity
  }))
)

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

watch([page, perPage, orderByField, orderByDirection], fetchActivityLogs)

onMounted(fetchActivityLogs)
</script>
