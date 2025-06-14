<template>
  <Spinner v-if="isLoading" />
  <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
    <div class="flex justify-between mb-5">
      <div></div>
      <div>
        <RouterLink :to="{ name: 'new-webhook' }">
          <Button>{{
            $t('globals.messages.new', {
              name: $t('globals.terms.webhook')
            })
          }}</Button>
        </RouterLink>
      </div>
    </div>
    <div>
      <DataTable :columns="createColumns(t)" :data="webhooks" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@/components/datatable/DataTable.vue'
import { createColumns } from '@/features/admin/webhooks/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import { useEmitter } from '@/composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { Spinner } from '@/components/ui/spinner'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const webhooks = ref([])
const { t } = useI18n()
const isLoading = ref(false)
const emit = useEmitter()

onMounted(() => {
  fetchAll()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const refreshList = (data) => {
  if (data?.model === 'webhook') fetchAll()
}

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getWebhooks()
    webhooks.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}
</script>
