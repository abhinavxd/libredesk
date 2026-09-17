<template>
  <div class="flex flex-col h-screen">
    <SearchHeader />
    <div class="flex-1 overflow-y-auto">
      <div class="mx-auto max-w-6xl space-y-4 px-4 py-6">
        <div class="relative">
          <SearchIcon
            class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-muted-foreground pointer-events-none"
            aria-hidden="true"
          />
          <Input
            ref="inputRef"
            v-model="term"
            :placeholder="$t('search.searchBy')"
            :aria-label="$t('globals.terms.search')"
            class="h-12 pl-10 pr-20 text-base"
          />
          <Spinner
            v-if="loading"
            size="sm"
            variant="muted"
            :absolute="false"
            :center="false"
            class="absolute right-10 top-1/2 -translate-y-1/2"
          />
          <button
            v-if="term"
            type="button"
            class="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent"
            :aria-label="$t('globals.messages.clear')"
            @click="term = ''"
          >
            <X class="w-4 h-4" aria-hidden="true" />
          </button>
        </div>

        <SearchFilters :filters="filters" @update:filters="filters = $event" />

        <div v-if="loading && totalResults === 0" class="flex justify-center items-center h-64">
          <Spinner :absolute="false" />
        </div>
        <div v-else-if="error" class="py-16 text-center space-y-4">
          <p class="text-destructive">{{ error }}</p>
          <Button type="button" @click="search"> {{ $t('globals.terms.tryAgain') }} </Button>
        </div>
        <SearchResults
          v-else-if="searchPerformed"
          :results="results"
          :term="term"
          :active-tab="activeTab"
          :show-clear-filters="hasActiveFilters(filters)"
          :aria-busy="loading"
          :class="{ 'pointer-events-none opacity-60': loading }"
          @update:active-tab="selectTab"
          @change-page="changePage"
          @clear-filters="filters = emptyFilters()"
        />
        <p
          v-else-if="term.length > 0 && term.length < MIN_SEARCH_LENGTH"
          class="py-16 text-center text-sm text-muted-foreground"
        >
          {{ $t('search.minQueryLength', { length: MIN_SEARCH_LENGTH }) }}
        </p>
        <div v-else class="py-16 text-center text-muted-foreground">
          <SearchIcon class="w-8 h-8 mx-auto mb-2 opacity-60" aria-hidden="true" />
          <p class="text-sm">{{ $t('globals.messages.resultsAppearHere') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { Search as SearchIcon, X } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import Spinner from '@shared-ui/components/ui/spinner/Spinner.vue'
import SearchHeader from '@main/features/search/SearchHeader.vue'
import SearchFilters from '@main/features/search/SearchFilters.vue'
import SearchResults from '@main/features/search/SearchResults.vue'
import {
  emptyFilters,
  filtersFromQuery,
  hasActiveFilters,
  queryFromFilters,
  toFiltersJSON
} from '@main/features/search/searchFilters'
import api from '@main/api'

const MIN_SEARCH_LENGTH = 3
const DEBOUNCE_DELAY = 300
const DEFAULT_PER_PAGE = 30
const TABS = ['conversations', 'messages']

const route = useRoute()
const router = useRouter()

const emptyPage = () => ({
  results: [],
  total: 0,
  page: 1,
  per_page: DEFAULT_PER_PAGE,
  total_pages: 0
})
const emptyResults = () => ({ conversations: emptyPage(), messages: emptyPage() })

const inputRef = ref(null)
const term = ref(String(route.query.q || ''))
const filters = ref(filtersFromQuery(route.query))
const activeTab = ref(TABS.includes(route.query.tab) ? route.query.tab : 'conversations')
const tabSelectedByUser = ref(TABS.includes(route.query.tab))
const results = ref(emptyResults())
const loading = ref(false)
const error = ref(null)
const searchPerformed = ref(false)
let debounceTimer = null
let searchRequestId = 0

const totalResults = computed(
  () => results.value.conversations.total + results.value.messages.total
)

const searchParams = (page, perPage) => {
  const params = { query: term.value, page, page_size: perPage }
  const filtersJSON = toFiltersJSON(filters.value)
  if (filtersJSON) params.filters = filtersJSON
  return params
}

const fetchers = {
  conversations: api.searchConversations,
  messages: api.searchMessages
}

const reset = () => {
  searchRequestId++
  results.value = emptyResults()
  searchPerformed.value = false
  loading.value = false
  error.value = null
}

const search = async () => {
  if (term.value.length < MIN_SEARCH_LENGTH) {
    reset()
    return
  }
  loading.value = true
  error.value = null
  searchPerformed.value = true

  const requestId = ++searchRequestId
  try {
    const perPage = Object.fromEntries(TABS.map((type) => [type, results.value[type].per_page]))
    const pages = await Promise.all(
      TABS.map((type) => fetchers[type](searchParams(1, perPage[type])))
    )
    if (requestId !== searchRequestId) return
    results.value = Object.fromEntries(TABS.map((type, i) => [type, pages[i].data.data]))
    if (!tabSelectedByUser.value && results.value[activeTab.value].total === 0) {
      const populatedTab = TABS.find((type) => results.value[type].total > 0)
      if (populatedTab) activeTab.value = populatedTab
    }
  } catch (err) {
    if (requestId !== searchRequestId) return
    error.value = handleHTTPError(err).message
  } finally {
    if (requestId === searchRequestId) loading.value = false
  }
}

const fetchPage = async (type, page, perPage) => {
  loading.value = true
  error.value = null
  const requestId = ++searchRequestId
  try {
    const response = await fetchers[type](searchParams(page, perPage))
    if (requestId !== searchRequestId) return
    results.value[type] = response.data.data
  } catch (err) {
    if (requestId !== searchRequestId) return
    error.value = handleHTTPError(err).message
  } finally {
    if (requestId === searchRequestId) loading.value = false
  }
}

const changePage = ({ type, page, perPage }) => fetchPage(type, page, perPage)

const selectTab = (tab) => {
  tabSelectedByUser.value = true
  activeTab.value = tab
}

const syncRoute = () => {
  const query = { ...queryFromFilters(filters.value) }
  if (term.value) query.q = term.value
  if (tabSelectedByUser.value || activeTab.value !== 'conversations') {
    query.tab = activeTab.value
  }
  router.replace({ query })
}

const debouncedSearch = () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(search, DEBOUNCE_DELAY)
}

watch(term, (value) => {
  syncRoute()
  if (value.length >= MIN_SEARCH_LENGTH) {
    debouncedSearch()
  } else {
    clearTimeout(debounceTimer)
    reset()
  }
})

watch(filters, () => {
  syncRoute()
  if (term.value.length >= MIN_SEARCH_LENGTH) debouncedSearch()
})

watch(activeTab, syncRoute)

if (term.value.length >= MIN_SEARCH_LENGTH) search()

onMounted(() => inputRef.value?.$el?.focus?.())

onBeforeUnmount(() => {
  clearTimeout(debounceTimer)
})
</script>
