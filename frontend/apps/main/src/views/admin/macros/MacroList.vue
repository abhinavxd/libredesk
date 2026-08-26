<template>
  <LoadingOverlay :loading="formLoading" reserve-height>
    <div class="flex justify-end mb-5">
      <router-link :to="{ name: 'new-macro' }">
        <Button>
          {{
            $t('macro.new')
          }}
        </Button>
      </router-link>
    </div>
    <div class="space-y-3">
      <Input
        type="text"
        v-model="searchTerm"
        :placeholder="$t('globals.terms.search')"
        class="max-w-xs"
        @input="getMacrosDebounced"
      />
      <DataTable :columns="createColumns(t)" :data="macros" :loading="formLoading" :searchable="false" />
      <PaginationBar v-model:page="page" v-model:per-page="perPage" :total-pages="totalPages" />
    </div>
  </LoadingOverlay>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { createColumns } from '../../../features/admin/macros/dataTableColumns.js'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { useDebounceFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import api from '../../../api'

const { t } = useI18n()
const formLoading = ref(false)
const macros = ref([])
const page = ref(1)
const perPage = ref(15)
const totalPages = ref(0)
const searchTerm = ref('')
const emit = useEmitter()
let fetchRequestId = 0

onMounted(() => {
  getMacros()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const refreshList = (data) => {
  if (data?.model === 'macros') getMacros()
}

const getMacrosDebounced = useDebounceFn(() => {
  if (page.value === 1) {
    getMacros()
  } else {
    page.value = 1
  }
}, 300)

const getMacros = async () => {
  const requestId = ++fetchRequestId
  try {
    formLoading.value = true
    const resp = await api.getMacrosCompact({
      page: page.value,
      page_size: perPage.value,
      q: searchTerm.value.trim()
    })
    if (requestId !== fetchRequestId) return
    macros.value = resp.data.data.results
    totalPages.value = resp.data.data.total_pages
  } catch (error) {
    if (requestId !== fetchRequestId) return
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    if (requestId === fetchRequestId) formLoading.value = false
  }
}

watch([page, perPage], getMacros)
</script>
