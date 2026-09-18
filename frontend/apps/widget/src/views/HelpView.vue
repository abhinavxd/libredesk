<script setup>
import { computed, ref, nextTick, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Search, Maximize2, Minimize2, ExternalLink } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { useHelpStore } from '@widget/store/help.js'
import { useWidgetStore } from '@widget/store/widget.js'
import HelpCollectionList from '@shared-ui/components/HelpCollectionList.vue'
import ArticleReader from '@widget/components/ArticleReader.vue'
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
const feedback = computed(() => help.feedback[help.article?.id] ?? null)
const feedbackPending = computed(() => !!help.feedbackPending[help.article?.id])
const scroller = ref(null)
const backButton = ref(null)
const searchInput = ref(null)
const articleHistory = ref([])
const originArticleID = ref(null)
const currentCollection = computed(() => help.collectionPath.at(-1))
const collections = computed(() => currentCollection.value?.children || help.data?.tree || [])
const items = computed(() => help.results ?? currentCollection.value?.articles ?? [])
const articleURL = computed(() =>
  help.article
    ? `${help.data.url}/${encodeURIComponent(help.article.locale)}/articles/${encodeURIComponent(help.article.slug)}`
    : ''
)
const canExpand = computed(() => !!help.article && !widget.isMobileFullScreen)
const availableTranslations = computed(() => help.article?.translations || [])
const languageName = (code) => {
  try {
    return new Intl.DisplayNames([locale.value], { type: 'language' }).of(code) || code
  } catch {
    return code
  }
}
const toggleExpand = () => {
  widget.toggleExpand()
  help.setExpandArticles(widget.isExpanded)
}
const retry = () => {
  error.value = ''
  help.load(locale.value)
}
const search = async () => {
  if (busy.value) return
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
    if (valid(identity) && help.query.trim() === query) {
      error.value = t('widget.helpLoadError')
      help.failed = true
    }
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
const switchArticleLanguage = async (articleLocale) => {
  if (busy.value || articleLocale === help.article?.locale) return
  const translation = availableTranslations.value.find(({ locale }) => locale === articleLocale)
  if (!translation) return
  busy.value = true
  error.value = ''
  const identity = help.identity
  try {
    const response = await api.getHelpArticle(translation.slug, articleLocale)
    if (!valid(identity) || response.data.data.locale !== articleLocale) return
    help.article = response.data.data
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
const vote = async (helpful) => {
  error.value = ''
  const identity = help.identity
  try {
    await help.vote(helpful)
  } catch {
    if (valid(identity)) error.value = t('widget.helpFeedbackError')
  }
}
onMounted(async () => {
  if (!help.data && !help.loading) await help.load(locale.value)
  if (scroller.value) scroller.value.scrollTop = help.scrollTop
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
    <div v-if="help.loading || busy" role="status" class="px-4 py-2 text-sm text-muted-foreground">
      {{ t('globals.terms.loading') }}
    </div>
    <div v-if="error || help.failed" role="alert" class="p-4 text-sm">
      <p>{{ error || t('widget.helpLoadError') }}</p>
      <Button type="button" variant="outline" class="mt-2" @click="retry">{{
        t('globals.terms.tryAgain')
      }}</Button>
    </div>
    <template v-if="help.article">
      <div class="flex-1 min-h-0 overflow-auto">
        <ArticleReader
          :article="help.article"
          :help-slug="help.data.slug"
          :dark="widget.config.dark_mode"
          @article="openArticle"
        />
        <div class="flex items-center justify-center gap-2 p-3 border-t text-sm">
          <span>{{
            feedback !== null ? t('helpCenter.feedbackThanks') : t('widget.helpful')
          }}</span>
          <template v-if="feedback === null">
            <Button
              type="button"
              variant="outline"
              size="sm"
              :disabled="busy || feedbackPending"
              @click="vote(true)"
              >{{ t('globals.messages.yes') }}</Button
            >
            <Button
              type="button"
              variant="outline"
              size="sm"
              :disabled="busy || feedbackPending"
              @click="vote(false)"
              >{{ t('globals.messages.no') }}</Button
            >
          </template>
        </div>
        <div
          v-if="availableTranslations.length > 1"
          class="flex items-center justify-center gap-2 px-4 py-3 border-t text-sm"
        >
          <span>{{ t('globals.terms.language') }}</span>
          <Select :model-value="help.article.locale" @update:model-value="switchArticleLanguage">
            <SelectTrigger class="h-8 w-44" :aria-label="t('globals.terms.language')">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="translation in availableTranslations"
                :key="translation.locale"
                :value="translation.locale"
              >
                {{ languageName(translation.locale) }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <a
          :href="articleURL"
          target="_blank"
          rel="noopener noreferrer"
          class="flex items-center justify-center gap-1.5 pb-4 text-xs text-muted-foreground hover:text-foreground no-underline"
        >
          {{ t('widget.openInHelpCenter') }}
          <ExternalLink class="size-3.5" aria-hidden="true" />
        </a>
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
        <p
          v-if="help.results?.length === 0"
          role="status"
          class="p-4 text-sm text-muted-foreground"
        >
          {{ t('widget.noArticles') }}
        </p>
      </div>
    </template>
  </div>
</template>
