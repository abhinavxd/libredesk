<template>
  <div class="max-w-5xl mx-auto p-6 min-h-screen">
    <Tabs :default-value="defaultTab" v-model="activeTab">
      <TabsList class="grid w-full mb-6" :class="tabsGridClass">
        <TabsTrigger v-for="(items, type) in results" :key="type" :value="type" class="capitalize">
          {{ type }} ({{ items.length }})
        </TabsTrigger>
      </TabsList>

      <TabsContent v-for="(items, type) in results" :key="type" :value="type" class="mt-0">
        <div class="bg-background rounded border overflow-hidden">
          <!-- No results message -->
          <div v-if="items.length === 0" class="p-8 text-center text-muted-foreground">
            <div class="text-lg font-medium mb-2">
              {{
                $t('globals.messages.noResults', {
                  name: type
                })
              }}
            </div>
            <div class="text-sm">{{ $t('search.adjustSearchTerms') }}</div>
          </div>

          <!-- Results list -->
          <div v-else class="divide-y divide-border">
            <div
              v-for="item in items"
              :key="item.id || item.uuid"
              class="p-6 hover:bg-accent/50 transition duration-200 ease-in-out group"
            >
              <router-link
                :to="{
                  name: 'inbox-conversation',
                  params: {
                    uuid: type === 'conversations' ? item.uuid : item.conversation_uuid,
                    type: 'assigned'
                  }
                }"
                class="block"
              >
                <div class="flex justify-between items-start">
                  <div class="flex-grow">
                    <!-- Reference number -->
                    <div
                      class="text-sm font-semibold mb-2 text-muted-foreground group-hover:text-primary transition duration-200"
                    >
                      #{{
                        type === 'conversations'
                          ? item.reference_number
                          : item.conversation_reference_number
                      }}
                    </div>

                    <!-- Content -->
                    <div
                      class="text-foreground font-medium mb-2 text-lg group-hover:text-primary transition duration-200"
                    >
                      {{
                        truncateText(
                          type === 'conversations' ? item.subject : item.text_content,
                          100
                        )
                      }}
                    </div>

                    <!-- Timestamp -->
                    <div class="text-sm text-muted-foreground flex items-center">
                      <ClockIcon class="h-4 w-4 mr-1" />
                      {{
                        formatDate(
                          type === 'conversations' ? item.created_at : item.conversation_created_at
                        )
                      }}
                    </div>
                  </div>

                  <!-- Right arrow icon -->
                  <div
                    class="bg-secondary rounded-full p-2 group-hover:bg-primary transition duration-200"
                  >
                    <ChevronRightIcon
                      class="h-5 w-5 text-secondary-foreground group-hover:text-primary-foreground"
                      aria-hidden="true"
                    />
                  </div>
                </div>
              </router-link>
            </div>
          </div>
        </div>
      </TabsContent>
    </Tabs>
  </div>
</template>
<script setup>
import { computed, ref, watch } from 'vue'
import { ChevronRightIcon, ClockIcon } from 'lucide-vue-next'
import { format, parseISO } from 'date-fns'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

const props = defineProps({
  results: {
    type: Object,
    required: true
  }
})

// Get the first available tab as default
const defaultTab = computed(() => {
  const types = Object.keys(props.results)
  return types.length > 0 ? types[0] : ''
})

const activeTab = ref('')

// Watch for changes in results and set the first tab as active
watch(
  () => props.results,
  (newResults) => {
    const types = Object.keys(newResults)
    if (types.length > 0 && !activeTab.value) {
      activeTab.value = types[0]
    }
  },
  { immediate: true }
)

// Dynamic grid class based on number of tabs
const tabsGridClass = computed(() => {
  const tabCount = Object.keys(props.results).length
  if (tabCount <= 2) return 'grid-cols-2'
  if (tabCount <= 3) return 'grid-cols-3'
  if (tabCount <= 4) return 'grid-cols-4'
  return 'grid-cols-5'
})

const formatDate = (dateString) => {
  const date = parseISO(dateString)
  return format(date, 'MMM d, yyyy HH:mm')
}

const truncateText = (text, length) => {
  if (!text) return ''
  if (text.length <= length) return text
  return text.slice(0, length) + '...'
}
</script>
