<template>
  <div class="max-w-5xl mx-auto p-6 min-h-screen">
    <div class="space-y-8">
      <div
        v-for="(items, type) in results"
        :key="type"
        class="bg-card rounded shadow overflow-hidden"
      >
        <!-- Header for each section -->
        <h2
          class="bg-primary dark:bg-primary text-lg font-bold text-white dark:text-primary-foreground py-2 px-6 capitalize"
        >
          {{ type }}
        </h2>

        <!-- No results message -->
        <div v-if="items.length === 0" class="p-6 text-gray-500 dark:text-muted-foreground">
          {{
            $t('globals.messages.noResults', {
              name: type
            })
          }}
        </div>

        <!-- Results list -->
        <div class="divide-y divide-gray-200 dark:divide-border">
          <div
            v-for="item in items"
            :key="item.id || item.uuid"
            class="p-6 hover:bg-gray-100 dark:hover:bg-accent transition duration-300 ease-in-out group"
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
                    class="text-sm font-semibold mb-2 group-hover:text-primary dark:group-hover:text-primary transition duration-300"
                  >
                    #{{
                      type === 'conversations'
                        ? item.reference_number
                        : item.conversation_reference_number
                    }}
                  </div>

                  <!-- Content -->
                  <div
                    class="text-gray-900 dark:text-card-foreground font-medium mb-2 text-lg group-hover:text-gray-950 dark:group-hover:text-foreground transition duration-300"
                  >
                    {{
                      truncateText(type === 'conversations' ? item.subject : item.text_content, 100)
                    }}
                  </div>

                  <!-- Timestamp -->
                  <div class="text-sm text-gray-500 dark:text-muted-foreground flex items-center">
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
                  class="bg-gray-200 dark:bg-secondary rounded-full p-2 group-hover:bg-primary dark:group-hover:bg-primary transition duration-300"
                >
                  <ChevronRightIcon
                    class="h-5 w-5 text-gray-700 dark:text-secondary-foreground group-hover:text-white dark:group-hover:text-primary-foreground"
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
import { ChevronRightIcon, ClockIcon } from 'lucide-vue-next'
import { format, parseISO } from 'date-fns'

defineProps({
  results: {
    type: Object,
    required: true
  }
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
