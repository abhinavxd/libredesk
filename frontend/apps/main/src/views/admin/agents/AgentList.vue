<template>
  <Spinner v-if="isLoading" />
  <div :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
    <div class="flex justify-end mb-5">
      <router-link :to="{ name: 'new-agent' }">
        <Button>{{
          $t('globals.messages.new', {
            name: $t('globals.terms.agent', 1)
          })
        }}</Button>
      </router-link>
    </div>
    <div>
      <DataTable :columns="createColumns(t)" :data="data" />
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { createColumns } from '@/features/admin/agents/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/datatable/DataTable.vue'
import { handleHTTPError } from '@/utils/http'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useUsersStore } from '@/stores/users'
import { useI18n } from 'vue-i18n'

const isLoading = ref(false)
const usersStore = useUsersStore()
const { t } = useI18n()
const data = ref([])
const emitter = useEmitter()

onMounted(async () => {
  getData()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'agent') getData()
  })
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST)
})

const getData = async () => {
  try {
    isLoading.value = true
    await usersStore.fetchUsers(true)
    data.value = usersStore.users
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}
</script>
