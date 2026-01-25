<template>
  <Tabs default-value="new_conversation" v-model="selectedTab">
    <TabsList class="grid w-full grid-cols-3 mb-5">
      <TabsTrigger value="new_conversation">
        {{
          $t('globals.messages.new', {
            name: $t('globals.terms.conversation')
          })
        }}
      </TabsTrigger>
      <TabsTrigger value="conversation_update">
        {{ $t('admin.automation.conversationUpdate') }}
      </TabsTrigger>
      <TabsTrigger value="time_trigger">
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
