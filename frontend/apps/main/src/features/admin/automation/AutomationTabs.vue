<template>
  <Tabs default-value="new_conversation" v-model="selectedTab">
    <AdminToolbar class="mb-5">
      <template #left>
        <TabsList class="w-max justify-start">
          <TabsTrigger value="new_conversation">
            {{ i18n._('New conversation') }}
          </TabsTrigger>
          <TabsTrigger value="conversation_update">
            {{ i18n._('Conversation update') }}
          </TabsTrigger>
          <TabsTrigger value="time_trigger">
            {{ i18n._('Time triggers') }}
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
        :helptext="i18n._('Rules that run when a new conversation is created by a contact. Conversations initiated by agents do not trigger these rules. Drag and drop to reorder.')"
      />
    </TabsContent>
    <TabsContent value="conversation_update">
      <RuleTab
        type="conversation_update"
        :helptext="i18n._('Rules that run when a conversation is updated.')"
      />
    </TabsContent>
    <TabsContent value="time_trigger">
      <RuleTab type="time_trigger" :helptext="i18n._('Rules that run once an hour.')" />
    </TabsContent>
  </Tabs>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { useI18n } from '@shared-ui/i18n'
import RuleTab from './RuleTab.vue'

const i18n = useI18n()
const selectedTab = defineModel('automationsTab', {
  default: 'new_conversation',
  type: String,
  required: true
})
</script>
