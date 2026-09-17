<template>
  <div>
    <Tabs :model-value="activeTab" @update:model-value="emit('update:activeTab', $event)">
      <TabsList class="mb-3">
        <TabsTrigger v-for="type in TYPES" :key="type" :value="type" class="gap-1.5">
          {{ t(tabLabelKeys[type], 2) }}
          <span class="text-xs tabular-nums text-muted-foreground">{{ results[type].total }}</span>
        </TabsTrigger>
      </TabsList>

      <TabsContent
        v-for="type in TYPES"
        :key="type"
        :value="type"
        class="mt-0"
        :class="{ 'pb-4': results[type].total_pages <= 1 }"
      >
        <div class="bg-background rounded-md border overflow-hidden">
          <div v-if="results[type].results.length === 0" class="p-8 text-center text-muted-foreground">
            <div class="text-lg font-medium mb-2">{{ t('globals.messages.noResultsFound') }}</div>
            <div class="text-sm">{{ t('search.adjustSearchTerms') }}</div>
          </div>
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
        <PaginationBar
          v-if="results[type].total_pages > 1"
          :page="results[type].page"
          :per-page="results[type].per_page"
          :total-pages="results[type].total_pages"
          @change="emit('changePage', { type, ...$event })"
        />
      </TabsContent>
    </Tabs>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import PaginationBar from '@main/components/pagination/PaginationBar.vue'
import SearchResultItem from './SearchResultItem.vue'

const TYPES = ['conversations', 'messages']
const tabLabelKeys = {
  conversations: 'globals.terms.conversation',
  messages: 'globals.terms.message'
}

defineProps({
  results: { type: Object, required: true },
  term: { type: String, default: '' },
  activeTab: { type: String, required: true }
})
const emit = defineEmits(['update:activeTab', 'changePage'])

const { t } = useI18n()
</script>
