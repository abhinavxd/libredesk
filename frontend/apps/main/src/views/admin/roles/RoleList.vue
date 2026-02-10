<template>
  <Spinner v-if="isLoading" />
  <div :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
    <div class="flex justify-end mb-5">
      <router-link :to="{ name: 'new-role' }">
        <Button>
          {{
            $t('globals.messages.new', {
              name: $t('globals.terms.role')
            })
          }}
        </Button>
      </router-link>
    </div>
    <div>
      <DataTable :columns="createColumns(t)" :data="roles" :loading="isLoading" />
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { createColumns } from '../../../features/admin/roles/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import DataTable from '@main/components/datatable/DataTable.vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS} from '../../../constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '../../../api'

const emitter = useEmitter()
const { t } = useI18n()
const roles = ref([])
const isLoading = ref(false)

const getRoles = async () => {
  try {
    isLoading.value = true
    const resp = await api.getRoles()
    roles.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  getRoles()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'team') getRoles()
  })
})
</script>
