<template>
  <Spinner v-if="isLoading" />
  <div :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
    <div class="flex justify-between items-center mb-5">
      <div class="flex-1">
        <Input
          v-model="searchQuery"
          type="text"
          :placeholder="$t('globals.messages.searchByNameOrDomain')"
          class="max-w-md"
          @input="handleSearch"
        />
      </div>
      <router-link :to="{ name: 'new-organization' }">
        <Button>
          {{ $t('globals.messages.new', { name: $t('globals.terms.organization', 1) }) }}
        </Button>
      </router-link>
    </div>
    <div>
      <DataTable :columns="createColumns(t)" :data="data" @row-click="handleRowClick" />
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { createColumns } from '@/features/admin/organizations/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import DataTable from '@/components/datatable/DataTable.vue'
import { handleHTTPError } from '@/utils/http'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import api from '@/api'

const isLoading = ref(false)
const { t } = useI18n()
const router = useRouter()
const data = ref([])
const emitter = useEmitter()
const searchQuery = ref('')
let searchTimeout = null

onMounted(async () => {
  getData()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'organization') getData()
  })
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST)
})

const getData = async () => {
  try {
    isLoading.value = true
    const params = {}
    if (searchQuery.value) {
      params.search = searchQuery.value
    }
    const response = await api.getOrganizations(params)
    data.value = response.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const handleSearch = () => {
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
  searchTimeout = setTimeout(() => {
    getData()
  }, 300)
}

const handleRowClick = (row) => {
  router.push({ name: 'organization-detail', params: { id: row.id } })
}
</script>
