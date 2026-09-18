import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@widget/api/index.js'

const EXPAND_KEY = 'libredesk-help-expand'
const readExpandArticles = () => {
  try { return localStorage.getItem(EXPAND_KEY) !== 'false' } catch { return true }
}

export const useHelpStore = defineStore('help', () => {
  const data = ref(null)
  const loading = ref(false)
  const failed = ref(false)
  const query = ref('')
  const results = ref(null)
  const collectionPath = ref([])
  const navigationHistory = ref([])
  const article = ref(null)
  const feedback = ref({})
  const feedbackPending = ref({})
  const scrollTop = ref(0)
  const pendingArticle = ref(null)
  const focusSearch = ref(false)
  const expandArticles = ref(readExpandArticles())
  const identity = ref(0)
  let generation = 0
  const articles = computed(() => {
    const collect = (nodes) =>
      (nodes || []).flatMap((node) => [...(node.articles || []), ...collect(node.children)])
    return collect(data.value?.tree)
  })
  const available = computed(() => !!data.value && articles.value.length > 0)
  const featured = computed(() =>
    data.value?.featured_ids?.length
      ? data.value.featured_ids
          .map((id) => articles.value.find((article) => article.id === id))
          .filter(Boolean)
      : data.value?.popular || []
  )
  const reset = () => {
    generation++
    identity.value++
    loading.value = false
    scrollTop.value = 0
    pendingArticle.value = null
    focusSearch.value = false
    data.value = null
    failed.value = false
    query.value = ''
    results.value = null
    article.value = null
    feedback.value = {}
    feedbackPending.value = {}
    collectionPath.value = []
    navigationHistory.value = []
  }
  const vote = async (helpful) => {
    const target = article.value
    if (!target || feedback.value[target.id] !== undefined || feedbackPending.value[target.id])
      return
    const currentIdentity = identity.value
    feedbackPending.value[target.id] = true
    try {
      await api.sendHelpFeedback(data.value.slug, target.slug, target.locale, helpful)
      if (identity.value === currentIdentity) feedback.value[target.id] = helpful
    } finally {
      if (identity.value === currentIdentity) delete feedbackPending.value[target.id]
    }
  }
  const load = async (locale) => {
    const request = ++generation
    loading.value = true
    failed.value = false
    try {
      const response = await api.getHelp(locale)
      if (request === generation) data.value = response.data.data
    } catch {
      if (request === generation) {
        data.value = null
        failed.value = true
      }
    } finally {
      if (request === generation) loading.value = false
    }
  }
  const setExpandArticles = (value) => {
    expandArticles.value = value
    try { localStorage.setItem(EXPAND_KEY, String(value)) } catch { /* storage blocked */ }
  }
  return {
    focusSearch,
    expandArticles,
    setExpandArticles,
    feedback,
    feedbackPending,
    vote,
    navigationHistory,
    pendingArticle,
    identity,
    data,
    loading,
    failed,
    query,
    results,
    collectionPath,
    article,
    scrollTop,
    articles,
    available,
    featured,
    load,
    reset
  }
})
