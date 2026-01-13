<template>
  <Spinner v-if="formLoading" />
  <div :class="{ 'transition-opacity duration-300 opacity-50': formLoading }">
    <div class="flex justify-end mb-5">
      <router-link :to="{ name: 'new-shared-view' }">
        <Button>
          {{
            $t('globals.messages.new', {
              name: $t('globals.terms.sharedView').toLowerCase()
            })
          }}
        </Button>
      </router-link>
    </div>
    <div>
      <DataTable :columns="createColumns(t)" :data="sharedViews" :loading="formLoading" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@/components/datatable/DataTable.vue'
import { createColumns } from '@/features/admin/shared-views/dataTableColumns.js'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import { Button } from '@/components/ui/button'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const { t } = useI18n()
const formLoading = ref(false)
const sharedViews = ref([])
const emit = useEmitter()

onMounted(() => {
  getSharedViews()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const refreshList = (data) => {
  if (data?.model === 'shared-views') getSharedViews()
}

const getSharedViews = async () => {
  try {
    formLoading.value = true
    const resp = await api.getAllSharedViews()
    sharedViews.value = resp.data.data
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
