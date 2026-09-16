<template>
  <div class="flex flex-wrap items-center gap-2">
    <div class="flex-1 min-w-28">
      <SelectComboBox
        :model-value="filters.status"
        :items="conversationStore.statusOptions"
        :placeholder="t('globals.terms.status')"
        align="start"
        @update:model-value="set('status', $event)"
      />
    </div>
    <div class="flex-1 min-w-28">
      <SelectComboBox
        :model-value="filters.priority"
        :items="conversationStore.priorityOptions"
        :placeholder="t('globals.terms.priority')"
        align="start"
        @update:model-value="set('priority', $event)"
      />
    </div>
    <div class="flex-1 min-w-28">
      <SelectComboBox
        :model-value="filters.inbox"
        :items="inboxStore.options"
        :placeholder="t('globals.terms.inbox')"
        align="start"
        @update:model-value="set('inbox', $event)"
      />
    </div>
    <div class="flex-1 min-w-28">
      <SelectAgentCombobox
        :model-value="filters.assignee"
        :placeholder="t('globals.terms.assignee')"
        :prepend-items="unassignedItem"
        align="start"
        @update:model-value="set('assignee', $event)"
      />
    </div>
    <div class="flex-1 min-w-28">
      <SelectTeamCombobox
        :model-value="filters.team"
        :placeholder="t('globals.terms.team')"
        :prepend-items="unassignedItem"
        align="start"
        @update:model-value="set('team', $event)"
      />
    </div>
    <div class="flex-[2] min-w-56">
      <SelectTagCombobox
        :model-value="filters.tags"
        :placeholder="t('globals.terms.tag', 2)"
        value-field="id"
        multiple
        @update:model-value="set('tags', $event)"
      />
    </div>
    <div class="flex-[1.5] min-w-60">
      <DateFilterValue
        :model-value="filters.created"
        :placeholder="t('search.conversationCreatedDate')"
        range
        @update:model-value="set('created', $event)"
      />
    </div>
    <Button
      v-if="hasActiveFilters(filters)"
      type="button"
      variant="ghost"
      size="sm"
      class="text-muted-foreground"
      @click="emit('update:filters', emptyFilters())"
    >
      <X class="w-4 h-4 mr-1" aria-hidden="true" />
      {{ t('globals.messages.clearFilters') }}
    </Button>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import SelectTagCombobox from '@main/components/combobox/SelectTagCombobox.vue'
import DateFilterValue from '@main/components/filter/DateFilterValue.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useInboxStore } from '@main/stores/inbox'
import { emptyFilters, hasActiveFilters, UNASSIGNED } from './searchFilters'

const props = defineProps({
  filters: { type: Object, required: true }
})
const emit = defineEmits(['update:filters'])

const { t } = useI18n()
const conversationStore = useConversationStore()
const inboxStore = useInboxStore()

const unassignedItem = computed(() => [{ value: UNASSIGNED, label: t('globals.terms.unassigned') }])

const set = (key, value) => {
  emit('update:filters', { ...props.filters, [key]: value ?? (key === 'tags' ? [] : '') })
}

onMounted(() => {
  conversationStore.fetchStatuses()
  conversationStore.fetchPriorities()
  inboxStore.fetchInboxes()
})
</script>
