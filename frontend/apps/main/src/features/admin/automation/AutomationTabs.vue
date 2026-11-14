<template>
  <Tabs default-value="new_conversation" v-model="selectedTab">
    <AdminToolbar class="mb-5">
      <template #left>
        <TabsList class="w-max justify-start">
          <TabsTrigger value="new_conversation">
            {{ $t('conversation.newConversation') }}
          </TabsTrigger>
          <TabsTrigger value="conversation_update">
            {{ $t('admin.automation.conversationUpdate') }}
          </TabsTrigger>
          <TabsTrigger value="time_trigger">
            {{ $t('admin.automation.timeTriggers') }}
          </TabsTrigger>
        </TabsList>
      </template>
      <template v-if="$slots.actions" #actions>
        <slot name="actions" />
      </template>
    </AdminToolbar>
    <TabsContent value="new_conversation">
      <RuleTab
        type="new_conversation"
        :helptext="t('admin.automation.newConversation.description')"
      />
    </TabsContent>
    <TabsContent value="conversation_update">
      <RuleTab
        type="conversation_update"
        :helptext="t('admin.automation.conversationUpdate.description')"
      />
    </TabsContent>
    <TabsContent value="time_trigger">
      <RuleTab type="time_trigger" :helptext="t('admin.automation.timeTriggers.description')" />
    </TabsContent>
  </Tabs>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { useI18n } from 'vue-i18n'
import RuleTab from './RuleTab.vue'

const { t } = useI18n()
const selectedTab = defineModel('automationsTab', {
  default: 'new_conversation',
  type: String,
  required: true
})
</script>
