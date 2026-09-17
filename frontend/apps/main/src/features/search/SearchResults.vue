<template>
  <div>
    <Tabs :model-value="activeTab" @update:model-value="emit('update:activeTab', $event)">
      <div class="mb-3 flex items-center gap-2">
        <TabsList>
          <TabsTrigger v-for="type in TYPES" :key="type" :value="type" class="gap-1.5">
            {{ t(tabLabelKeys[type], 2) }}
            <span class="text-xs tabular-nums text-muted-foreground">{{
              results[type].total
            }}</span>
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
import { X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
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
  activeTab: { type: String, required: true },
  showClearFilters: { type: Boolean, default: false }
})
const emit = defineEmits(['update:activeTab', 'changePage', 'clearFilters'])

const { t } = useI18n()
</script>
