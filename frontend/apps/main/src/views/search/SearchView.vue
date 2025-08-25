<template>
  <div class="flex flex-col h-screen">
    <SearchHeader v-model="searchQuery" @search="handleSearch" />
    <div class="flex-1 overflow-y-auto">
      <div v-if="loading" class="flex justify-center items-center h-64">
        <Spinner />
      </div>
      <div v-else-if="error" class="mt-8 text-center space-y-4">
        <p class="text-lg text-destructive">{{ error }}</p>
        <Button @click="handleSearch"> {{ $t('globals.terms.tryAgain') }} </Button>
      </div>

      <div v-else>
        <p
          v-if="searchPerformed && totalResults === 0"
          class="mt-8 text-center text-muted-foreground"
        >
          {{
            $t('search.noResultsForQuery', {
              query: searchQuery
            })
          }}
        </p>
        <SearchResults v-else-if="searchPerformed" :results="results" class="h-full" />

        <p
          v-else-if="searchQuery.length > 0 && searchQuery.length < MIN_SEARCH_LENGTH"
          class="mt-8 text-center text-muted-foreground"
        >
          {{
            $t('search.minQueryLength', {
              length: MIN_SEARCH_LENGTH
            })
          }}
        </p>
        <div v-else class="mt-16 text-center">
          <h2 class="text-2xl font-semibold text-primary mb-4">
            {{
              $t('globals.messages.search', {
                name: $t('globals.terms.conversation', 2).toLowerCase()
              })
            }}
          </h2>
          <p class="text-lg text-muted-foreground">
            {{ $t('search.searchBy') }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { handleHTTPError } from '../../utils/http'
import { Button } from '@shared-ui/components/ui/button'
import SearchHeader from '@/features/search/SearchHeader.vue'
import SearchResults from '@/features/search/SearchResults.vue'
import Spinner from '@shared-ui/components/ui/spinner/Spinner.vue'
import api from '../../api'

const MIN_SEARCH_LENGTH = 3
const DEBOUNCE_DELAY = 300

const searchQuery = ref('')
const results = ref({ conversations: [], messages: [] })
const loading = ref(false)
const error = ref(null)
const searchPerformed = ref(false)
let debounceTimer = null

const totalResults = computed(() => {
  return results.value.conversations.length + results.value.messages.length
})

const handleSearch = async () => {
  if (searchQuery.value.length < MIN_SEARCH_LENGTH) {
    results.value = { conversations: [], messages: [] }
    searchPerformed.value = false
    return
  }

  loading.value = true
  error.value = null
  searchPerformed.value = true

  try {
    const [convResults, messagesResults] = await Promise.all([
      api.searchConversations({ query: searchQuery.value }),
      api.searchMessages({ query: searchQuery.value })
    ])

    results.value = {
      conversations: convResults.data.data,
      messages: messagesResults.data.data
    }
  } catch (err) {
    error.value = handleHTTPError(err).message
  } finally {
    loading.value = false
  }
}

const debouncedSearch = () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(handleSearch, DEBOUNCE_DELAY)
}

watch(searchQuery, (newValue) => {
  if (newValue.length >= MIN_SEARCH_LENGTH) {
    debouncedSearch()
  } else {
    clearTimeout(debounceTimer)
    results.value = { conversations: [], messages: [] }
    searchPerformed.value = false
  }
})

onBeforeUnmount(() => {
  clearTimeout(debounceTimer)
})
</script>
