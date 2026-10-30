<template>
  <Collapsible v-model:open="advancedOpen" class="space-y-2">
    <div class="flex flex-wrap items-center gap-2">
      <div :class="FILTER_CLASS">
        <SelectComboBox
          :model-value="filters.status"
          :items="statusItems"
          :placeholder="i18n.ngettext('Status', 'Statuses', 1)"
          align="start"
          @update:model-value="set('status', $event)"
        />
      </div>
      <div :class="FILTER_CLASS">
        <SelectComboBox
          :model-value="filters.inbox"
          :items="inboxStore.options"
          :placeholder="i18n.ngettext('Inbox', 'Inboxes', 1)"
          align="start"
          @update:model-value="set('inbox', $event)"
        />
      </div>
      <div :class="FILTER_CLASS">
        <SelectAgentCombobox
          :model-value="filters.assignee"
          :placeholder="i18n._('Assignee')"
          :prepend-items="unassignedItem"
          align="start"
          @update:model-value="set('assignee', $event)"
        />
      </div>
      <CollapsibleTrigger as-child>
        <Button type="button" variant="outline" class="shrink-0 gap-2 font-normal">
          <SlidersHorizontal class="size-4" aria-hidden="true" />
          {{ i18n._('More filters') }}
          <Badge v-if="advancedFilterCount" variant="secondary" class="tabular-nums">
            {{ advancedFilterCount }}
          </Badge>
          <ChevronDown :class="['size-4', { 'rotate-180': advancedOpen }]" aria-hidden="true" />
        </Button>
      </CollapsibleTrigger>
    </div>

    <CollapsibleContent style="animation: none">
      <div class="flex flex-wrap items-center gap-2 pt-1">
        <div :class="FILTER_CLASS">
          <SelectComboBox
            :model-value="filters.priority"
            :items="priorityItems"
            :placeholder="i18n.ngettext('Priority', 'Priorities', 1)"
            align="start"
            @update:model-value="set('priority', $event)"
          />
        </div>
        <div :class="FILTER_CLASS">
          <SelectTeamCombobox
            :model-value="filters.team"
            :placeholder="i18n.ngettext('Team', 'Teams', 1)"
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
            :placeholder="i18n._('Conversation date')"
            range
            @update:model-value="set('created', $event)"
          />
        </div>
      </div>
    </CollapsibleContent>

    <div v-if="selectedTags.length" class="flex flex-wrap gap-1.5">
      <Button
        v-for="tag in selectedTags"
        :key="tag.value"
        type="button"
        variant="secondary"
        size="xs"
        class="max-w-full gap-1 font-normal"
        :aria-label="`${i18n._('Remove')} ${tag.label}`"
        @click="removeTag(tag.value)"
      >
        <span class="truncate">{{ tag.label }}</span>
        <X aria-hidden="true" />
      </Button>
    </div>
  </Collapsible>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { useStorage } from '@vueuse/core'
import { ChevronDown, SlidersHorizontal, X } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger
} from '@shared-ui/components/ui/collapsible'
import { useI18n } from '@shared-ui/i18n'
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
const asStringValues = (options) =>
  options.map((option) => ({ ...option, value: String(option.value) }))

const props = defineProps({
  filters: { type: Object, required: true }
})
const emit = defineEmits(['update:filters'])

const advancedFilterCount = computed(
  () =>
    ['priority', 'team', 'tags', 'created'].filter((key) =>
      key === 'tags' ? props.filters.tags.length > 0 : Boolean(props.filters[key])
    ).length
)
const advancedOpen = useStorage('searchMoreFiltersOpen', advancedFilterCount.value > 0)

watch(advancedFilterCount, (count, previousCount) => {
  if (count > 0 && previousCount === 0) advancedOpen.value = true
})

const i18n = useI18n()
const conversationStore = useConversationStore()
const statusItems = computed(() => asStringValues(conversationStore.statusOptions))
const priorityItems = computed(() => asStringValues(conversationStore.priorityOptions))
const inboxStore = useInboxStore()
const tagStore = useTagStore()

const unassignedItem = computed(() => [{ value: UNASSIGNED, label: i18n._('Unassigned') }])
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
