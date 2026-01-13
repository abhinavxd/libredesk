<template>
  <Spinner v-if="formLoading" />
  <div :class="{ 'transition-opacity duration-300 opacity-50': formLoading }">
    <div class="flex justify-end mb-5">
      <router-link :to="{ name: 'new-macro' }">
        <Button>
          {{
            $t('globals.messages.new', {
              name: $t('globals.terms.macro')
            })
          }}
        </Button>
      </router-link>
    </div>
    <div>
      <DataTable :columns="createColumns(t)" :data="macros" :loading="formLoading" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { createColumns } from '../../../features/admin/macros/dataTableColumns.js'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { handleHTTPError } from '../../../utils/http'
import { Button } from '@shared-ui/components/ui/button'
import { useI18n } from 'vue-i18n'
import api from '../../../api'

const { t } = useI18n()
const formLoading = ref(false)
const macros = ref([])
const emit = useEmitter()

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

const getMacros = async () => {
  try {
    formLoading.value = true
    const resp = await api.getAllMacros()
    macros.value = resp.data.data
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
