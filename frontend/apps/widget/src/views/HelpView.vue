<script setup>
import { computed, ref, nextTick, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Search, Maximize2, Minimize2, FileQuestionMark } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { useHelpStore } from '@widget/store/help.js'
import { useWidgetStore } from '@widget/store/widget.js'
import HelpCollectionList from '@shared-ui/components/HelpCollectionList.vue'
import ArticleReader from '@widget/components/ArticleReader.vue'
import { Spinner } from '@shared-ui/components/ui/spinner'
import api from '@widget/api/index.js'

const help = useHelpStore()
const widget = useWidgetStore()
const { t, locale } = useI18n()
let mounted = true
onUnmounted(() => {
  mounted = false
})
const valid = (identity) => mounted && identity === help.identity
const busy = ref(false)
const error = ref('')
const scroller = ref(null)
const backButton = ref(null)
const searchInput = ref(null)
const articleHistory = ref([])
const originArticleID = ref(null)
const currentCollection = computed(() => help.collectionPath.at(-1))
const collections = computed(() => currentCollection.value?.children || help.data?.tree || [])
const items = computed(() => help.results ?? currentCollection.value?.articles ?? [])
const canExpand = computed(() => !!help.article && !widget.isMobileFullScreen)
const toggleExpand = () => {
  widget.toggleExpand()
  help.setExpandArticles(widget.isExpanded)
}
const retry = () => {
  error.value = ''
  help.load(locale.value)
}
const search = async () => {
  const query = help.query.trim()
  if (!query) {
    help.results = null
    return
  }
  busy.value = true
  error.value = ''
  const identity = help.identity
  try {
    const response = await api.searchHelp(query, help.data.locale)
    if (!valid(identity) || help.query.trim() !== query) return
    help.results = response.data.data || []
  } catch {
    if (valid(identity) && help.query.trim() === query) error.value = t('widget.helpLoadError')
  } finally {
    busy.value = false
  }
}
const openArticle = async ({ id, slug, locale: articleLocale }) => {
  if (busy.value) return
  busy.value = true
  error.value = ''
  const identity = help.identity
  try {
    const response = await api.getHelpArticle(slug, articleLocale || help.data.locale)
    if (!valid(identity)) return
    if (help.article) articleHistory.value.push(help.article)
    else {
      help.scrollTop = scroller.value?.scrollTop || 0
      originArticleID.value = id
      if (help.expandArticles && !widget.isMobileFullScreen) widget.expandWidget()
    }
    help.article = response.data.data
    await nextTick()
    backButton.value?.$el?.focus()
  } catch {
    if (valid(identity)) error.value = t('widget.helpLoadError')
  } finally {
    busy.value = false
  }
}
const openCollection = async (collection) => {
  help.navigationHistory.push({ top: help.scrollTop, id: collection.id })
  help.collectionPath.push(collection)
  help.scrollTop = 0
  await nextTick()
  if (scroller.value) scroller.value.scrollTop = 0
  backButton.value?.$el?.focus()
}
const back = async () => {
  let selector
  if (help.article) {
    help.article = articleHistory.value.pop() || null
    if (!help.article) {
      selector = `[data-help-article="${originArticleID.value}"]`
      if (widget.isExpanded) widget.collapseWidget()
    }
  } else if (help.results !== null) {
    help.results = null
  } else {
    help.collectionPath.pop()
    const previous = help.navigationHistory.pop()
    help.scrollTop = previous?.top || 0
    selector = `[data-help-collection="${previous?.id}"]`
  }
  await nextTick()
  if (scroller.value) {
    scroller.value.scrollTop = help.scrollTop
    if (selector) scroller.value.querySelector(selector)?.focus({ preventScroll: true })
  }
}
onMounted(async () => {
  if (!help.data && !help.loading) await help.load(locale.value)
  if (scroller.value) scroller.value.scrollTop = help.scrollTop
  if (help.article && help.expandArticles && !widget.isMobileFullScreen) widget.expandWidget()
  if (help.pendingArticle && mounted) {
    const article = help.pendingArticle
    help.pendingArticle = null
    await openArticle(article)
  }
  if (help.focusSearch && mounted) {
    help.focusSearch = false
    await nextTick()
    requestAnimationFrame(() => searchInput.value?.$el?.focus())
  }
})
</script>

<template>
  <div class="flex flex-col h-full">
    <header class="relative flex items-center justify-center p-4 border-b min-h-[3.75rem]">
      <Button
        v-if="help.article || currentCollection || help.results !== null"
        ref="backButton"
        type="button"
        variant="ghost"
        size="icon"
        class="absolute left-2"
        :aria-label="t('globals.messages.goBack')"
        @click="back"
        ><ArrowLeft class="size-4" aria-hidden="true"
      /></Button>
      <h3 v-if="!help.article" class="text-base font-semibold text-foreground">
        {{ t('globals.terms.help') }}
      </h3>
      <Button
        v-if="canExpand"
        type="button"
        variant="ghost"
        size="icon"
        class="absolute right-2"
        :aria-label="widget.isExpanded ? t('globals.terms.collapse') : t('globals.terms.expand')"
        @click="toggleExpand"
      >
        <Minimize2 v-if="widget.isExpanded" class="size-4" aria-hidden="true" />
        <Maximize2 v-else class="size-4" aria-hidden="true" />
      </Button>
    </header>
    <div class="relative flex flex-col flex-1 min-h-0">
      <div
        v-if="help.loading || busy"
        class="absolute inset-0 bg-background/80 backdrop-blur-sm z-10"
        role="status"
      >
        <Spinner size="md" :text="t('globals.terms.loading')" absolute />
      </div>
      <div v-if="error || help.failed" role="alert" class="p-4 text-sm">
        <p>{{ error || t('widget.helpLoadError') }}</p>
        <Button type="button" variant="outline" class="mt-2" @click="retry">{{
          t('globals.terms.tryAgain')
        }}</Button>
      </div>
      <template v-if="help.article">
        <div class="flex-1 min-h-0 overflow-auto">
          <ArticleReader :article="help.article" :base-url="help.data.url" />
        </div>
      </template>
      <template v-else>
        <form class="flex gap-2 p-3 border-b" @submit.prevent="search">
          <Input
            ref="searchInput"
            v-model="help.query"
            :aria-label="t('widget.searchArticles')"
            :placeholder="t('widget.searchArticles')"
          />
          <Button
            type="submit"
            variant="outline"
            :disabled="busy || !help.available"
            :aria-label="t('globals.terms.search')"
            ><Search class="size-4" aria-hidden="true"
          /></Button>
        </form>
        <div
          ref="scroller"
          class="flex-1 min-h-0 overflow-auto"
          @scroll="help.scrollTop = $event.target.scrollTop"
        >
          <HelpCollectionList
            :collection="currentCollection"
            :collections="collections"
            :articles="items"
            :searching="help.results !== null"
            @collection="openCollection"
            @article="openArticle"
          />
          <div
            v-if="help.results?.length === 0"
            role="status"
            class="flex flex-col items-center justify-center px-4 py-12 text-center"
          >
            <FileQuestionMark class="w-10 h-10 text-muted-foreground mb-4" aria-hidden="true" />
            <p class="text-sm text-muted-foreground">{{ t('widget.noArticles') }}</p>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
