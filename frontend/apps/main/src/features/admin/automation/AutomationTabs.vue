<template>
  <Tabs default-value="new_conversation" v-model="selectedTab">
    <TabsList class="flex w-full justify-start overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden mb-5 sm:grid sm:grid-cols-3">
      <TabsTrigger value="new_conversation" :class="AUTOMATION_TAB_CLASS">
        {{ $t('conversation.newConversation') }}
      </TabsTrigger>
      <TabsTrigger value="conversation_update" :class="AUTOMATION_TAB_CLASS">
        {{ $t('admin.automation.conversationUpdate') }}
      </TabsTrigger>
      <TabsTrigger value="time_trigger" :class="AUTOMATION_TAB_CLASS">
        {{ $t('admin.automation.timeTriggers') }}
      </TabsTrigger>
    </TabsList>
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
const AUTOMATION_TAB_CLASS = 'shrink-0 sm:min-w-0'

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
