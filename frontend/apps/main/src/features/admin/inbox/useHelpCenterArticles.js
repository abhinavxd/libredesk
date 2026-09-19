import { ref, watch } from 'vue'
import api from '@/api'

const publishedTree = (nodes) =>
  (nodes || [])
    .filter((node) => node.is_published)
    .map((node) => {
      const children = publishedTree(node.children)
      const articles = (node.articles || [])
        .filter((article) => article.status === 'published')
        .map((article) => ({ ...article, collection_id: node.id }))
      return {
        ...node,
        children,
        articles,
        article_count:
          articles.length + children.reduce((total, child) => total + child.article_count, 0)
      }
    })

const flatten = (nodes) => nodes.flatMap((node) => [...node.articles, ...flatten(node.children)])

export const useHelpCenterArticles = (helpCenterID) => {
  const tree = ref([])
  const articles = ref([])
  const failed = ref(false)
  let request = 0

  watch(
    helpCenterID,
    async (id) => {
      const current = ++request
      tree.value = []
      articles.value = []
      failed.value = false
      if (!id) return
      try {
        const response = await api.getHelpCenterTree(id)
        if (current !== request) return
        tree.value = publishedTree(response.data.data.tree)
        articles.value = flatten(tree.value)
      } catch {
        if (current === request) failed.value = true
      }
    },
    { immediate: true }
  )

  return { tree, articles, failed }
}
