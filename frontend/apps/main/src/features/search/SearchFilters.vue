<template>
  <div class="space-y-2">
    <div class="flex flex-wrap items-center gap-2">
      <div :class="FILTER_CLASS">
        <SelectComboBox
          :model-value="filters.status"
          :items="statusItems"
          :placeholder="t('globals.terms.status')"
          align="start"
          @update:model-value="set('status', $event)"
        />
      </div>
      <div :class="FILTER_CLASS">
        <SelectComboBox
          :model-value="filters.priority"
          :items="priorityItems"
          :placeholder="t('globals.terms.priority')"
          align="start"
          @update:model-value="set('priority', $event)"
        />
      </div>
      <div :class="FILTER_CLASS">
        <SelectComboBox
          :model-value="filters.inbox"
          :items="inboxStore.options"
          :placeholder="t('globals.terms.inbox')"
          align="start"
          @update:model-value="set('inbox', $event)"
        />
      </div>
      <div :class="FILTER_CLASS">
        <SelectAgentCombobox
          :model-value="filters.assignee"
          :placeholder="t('globals.terms.assignee')"
          :prepend-items="unassignedItem"
          align="start"
          @update:model-value="set('assignee', $event)"
        />
      </div>
      <div :class="FILTER_CLASS">
        <SelectTeamCombobox
          :model-value="filters.team"
          :placeholder="t('globals.terms.team')"
          :prepend-items="unassignedItem"
          align="start"
          @update:model-value="set('team', $event)"
        />
      </div>
      <div class="w-36 shrink-0">
        <SearchTagFilter :model-value="filters.tags" @update:model-value="set('tags', $event)" />
      </div>
      <div class="flex-[1.5] min-w-60">
        <DateFilterValue
          :model-value="filters.created"
          :placeholder="t('globals.terms.conversationDate')"
          range
          @update:model-value="set('created', $event)"
        />
      </div>
    </div>

    <div v-if="selectedTags.length" class="flex flex-wrap gap-1.5">
      <Button
        v-for="tag in selectedTags"
        :key="tag.value"
        type="button"
        variant="secondary"
        size="xs"
        class="max-w-full gap-1 font-normal"
        :aria-label="`${t('globals.terms.remove')} ${tag.label}`"
        @click="removeTag(tag.value)"
      >
        <span class="truncate">{{ tag.label }}</span>
        <X aria-hidden="true" />
      </Button>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { X } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useI18n } from 'vue-i18n'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import DateFilterValue from '@main/components/filter/DateFilterValue.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useInboxStore } from '@main/stores/inbox'
import { useTagStore } from '@main/stores/tag'
import SearchTagFilter from './SearchTagFilter.vue'
import { UNASSIGNED } from './searchFilters'

const FILTER_CLASS = 'flex-1 min-w-28'

// The combobox matches the selected value by strict equality against a string.
const asStringValues = (options) => options.map((option) => ({ ...option, value: String(option.value) }))

const props = defineProps({
  filters: { type: Object, required: true }
})
const emit = defineEmits(['update:filters'])

const { t } = useI18n()
const conversationStore = useConversationStore()
const statusItems = computed(() => asStringValues(conversationStore.statusOptions))
const priorityItems = computed(() => asStringValues(conversationStore.priorityOptions))
const inboxStore = useInboxStore()
const tagStore = useTagStore()

const unassignedItem = computed(() => [{ value: UNASSIGNED, label: t('globals.terms.unassigned') }])
const selectedTags = computed(() => {
  const labels = new Map(tagStore.tagOptions.map((tag) => [String(tag.value), tag.label]))
  return props.filters.tags.map((value) => ({
    value,
    label: labels.get(String(value)) || value
  }))
})

const set = (key, value) => {
  emit('update:filters', { ...props.filters, [key]: value ?? (key === 'tags' ? [] : '') })
}

const removeTag = (value) => {
  set(
    'tags',
    props.filters.tags.filter((tag) => String(tag) !== String(value))
  )
}

onMounted(() => {
  conversationStore.fetchStatuses()
  conversationStore.fetchPriorities()
  inboxStore.fetchInboxes()
})
</script>
