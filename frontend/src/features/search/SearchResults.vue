<template>
  <div class="max-w-5xl mx-auto p-6 bg-background min-h-screen">
    <div v-if="isEmptyResults" class="text-center py-16 rounded-lg">
      <SearchXIcon class="h-20 w-20 text-muted-foreground mx-auto mb-6" />
      <h2 class="text-2xl font-bold text-foreground mb-3">No results found</h2>
      <p class="text-muted-foreground text-lg max-w-md mx-auto">
        We couldn't find any matches. Try adjusting your search query.
      </p>
    </div>
    <div v-else class="space-y-8">
      <div
        v-for="(items, type) in results"
        :key="type"
        class="bg-card rounded-lg shadow-md overflow-hidden"
      >
        <h2 class="bg-primary text-lg font-bold text-secondary py-2 px-6 capitalize">
          {{ type }}
        </h2>

        <div v-if="items.length === 0" class="p-6 text-muted-foreground">
          No {{ type }} found
        </div>

        <div class="divide-y divide-border">
          <div
            v-for="item in items"
            :key="item.id || item.uuid"
            class="p-6 hover:bg-accent transition duration-300 ease-in-out group"
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
                  <div
                    class="text-sm font-semibold text-primary mb-2 group-hover:text-primary transition duration-300"
                  >
                    #{{
                      type === 'conversations'
                        ? item.reference_number
                        : item.conversation_reference_number
                    }}
                  </div>
                  <div
                    class="text-card-foreground font-medium mb-2 text-lg group-hover:text-foreground transition duration-300"
                  >
                    {{
                      truncateText(type === 'conversations' ? item.subject : item.text_content, 100)
                    }}
                  </div>
                  <div class="text-sm text-muted-foreground flex items-center">
                    <ClockIcon class="h-4 w-4 mr-1" />
                    {{
                      formatDate(
                        type === 'conversations' ? item.created_at : item.conversation_created_at
                      )
                    }}
                  </div>
                </div>
                <div
                  class="bg-secondary rounded-full p-2 group-hover:bg-primary transition duration-300"
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
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { SearchXIcon, ChevronRightIcon, ClockIcon } from 'lucide-vue-next'
import { format, parseISO } from 'date-fns'

const props = defineProps({
  results: {
    type: Object,
    required: true
  }
})

const isEmptyResults = computed(() => {
  return Object.values(props.results).every((arr) => arr.length === 0)
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
