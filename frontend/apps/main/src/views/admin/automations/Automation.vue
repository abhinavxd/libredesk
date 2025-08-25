<template>
  <AdminPageWithHelp>
    <template #content>
      <div v-if="router.currentRoute.value.name === 'automations'">
        <div class="flex justify-between mb-5">
          <div class="ml-auto">
            <Button @click="newRule">{{
              $t('globals.messages.new', {
                name: $t('globals.terms.rule')
              })
            }}</Button>
          </div>
        </div>
        <div v-if="selectedTab">
          <AutomationTabs v-model:automationsTab="selectedTab" />
        </div>
      </div>
      <router-view />
    </template>

    <template #help>
      <p>Create automation rules to streamline your support workflow.</p>
      <p>
        Set actions to be performed when a conversation matches the rule criteria when it is created
        or updated or run rules hourly.
      </p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { useRouter } from 'vue-router'
import { useStorage } from '@vueuse/core'
import AutomationTabs from '@/features/admin/automation/AutomationTabs.vue'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'

const router = useRouter()
const selectedTab = useStorage('automationsTab', 'new_conversation')
const newRule = () => {
  router.push({ name: 'new-automation', query: { type: selectedTab.value } })
}
</script>
