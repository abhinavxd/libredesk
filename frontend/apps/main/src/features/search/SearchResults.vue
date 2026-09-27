<template>
  <div>
    <Tabs :model-value="activeTab" @update:model-value="emit('update:activeTab', $event)">
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <TabsList>
          <TabsTrigger v-for="type in TYPES" :key="type" :value="type" class="gap-1.5">
            {{ t(tabLabelKeys[type], 2) }}
            <span class="text-xs tabular-nums text-muted-foreground">
              {{ knownResultCount(results[type]) }}
            </span>
          </TabsTrigger>
        </TabsList>
        <Button
          v-if="showClearFilters"
          type="button"
          variant="ghost"
          size="sm"
          class="shrink-0 text-muted-foreground"
          @click="emit('clearFilters')"
        >
          <X class="h-4 w-4" aria-hidden="true" />
          {{ t('globals.messages.clearFilters') }}
        </Button>
        <div class="ml-auto flex items-center gap-2">
          <Label for="search-sort" class="font-normal text-muted-foreground">
            {{ t('globals.messages.sortBy') }}
          </Label>
          <Select
            :model-value="sorts[activeTab]"
            @update:model-value="emit('changeSort', { type: activeTab, sort: $event })"
          >
            <SelectTrigger id="search-sort" class="h-9 w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent align="end">
              <SelectItem
                v-for="option in SORT_OPTIONS[activeTab]"
                :key="option.value"
                :value="option.value"
              >
                {{ t(option.label) }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <TabsContent v-for="type in TYPES" :key="type" :value="type" class="mt-0">
        <div
          class="overflow-hidden rounded-md border bg-background"
          :class="{ 'border-dashed': results[type].results.length === 0 }"
        >
          <p
            v-if="results[type].results.length === 0"
            class="p-8 text-center text-sm text-muted-foreground"
          >
            {{ t('globals.messages.noResultsFound') }}
          </p>
          <div v-else class="divide-y divide-border">
            <SearchResultItem
              v-for="item in results[type].results"
              :key="item.uuid"
              :item="item"
              :type="type"
              :term="term"
            />
          </div>
        </div>
        <SearchPagination
          v-if="results[type].page > 1 || results[type].has_more"
          :page="results[type].page"
          :per-page="results[type].per_page"
          :has-more="results[type].has_more"
          @change="emit('changePage', { type, ...$event })"
        />
      </TabsContent>
    </Tabs>
  </div>
</template>

<script setup>
import { X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Label } from '@shared-ui/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import SearchPagination from './SearchPagination.vue'
import SearchResultItem from './SearchResultItem.vue'

const TYPES = ['conversations', 'messages']
const SORT_OPTIONS = {
  conversations: [
    { value: 'newest', label: 'conversation.sort.newestActivity' },
    { value: 'oldest', label: 'conversation.sort.oldestActivity' },
    { value: 'started_first', label: 'conversation.sort.startedFirst' },
    { value: 'started_last', label: 'conversation.sort.startedLast' }
  ],
  messages: [
    { value: 'newest', label: 'conversation.sort.newestActivity' },
    { value: 'oldest', label: 'conversation.sort.oldestActivity' }
  ]
}
const tabLabelKeys = {
  conversations: 'globals.terms.conversation',
  messages: 'globals.terms.message'
}

defineProps({
  results: { type: Object, required: true },
  term: { type: String, default: '' },
  activeTab: { type: String, required: true },
  sorts: { type: Object, required: true },
  showClearFilters: { type: Boolean, default: false }
})
const emit = defineEmits(['update:activeTab', 'changeSort', 'changePage', 'clearFilters'])

const { t } = useI18n()

const knownResultCount = (page) => {
  const count = (page.page - 1) * page.per_page + page.results.length
  return page.has_more ? `${count}+` : count
}
</script>
