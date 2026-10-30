<template>
  <div>
    <div v-if="selectedTab">
      <AutomationTabs v-model:automationsTab="selectedTab">
        <template #actions>
          <Button type="button" @click="newRule">{{ $t('automation.newRule') }}</Button>
        </template>
      </AutomationTabs>
    </div>
  </div>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { useRouter } from 'vue-router'
import { useStorage } from '@vueuse/core'
import AutomationTabs from '@/features/admin/automation/AutomationTabs.vue'

const router = useRouter()
const selectedTab = useStorage('automationsTab', 'new_conversation')
const newRule = () => {
  router.push({ name: 'new-automation', query: { type: selectedTab.value } })
}
</script>
