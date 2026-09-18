<script setup>
import { useHelpStore } from '@widget/store/help.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { Button } from '@shared-ui/components/ui/button'
import { Search } from 'lucide-vue-next'
const help = useHelpStore()
const widget = useWidgetStore()
const open = (article) => {
  help.article = null
  help.query = ''
  help.results = null
  help.collectionPath = []
  help.navigationHistory = []
  help.scrollTop = 0
  help.pendingArticle = article || null
  help.focusSearch = !article
  widget.navigateToHelp()
}
</script>

<template>
  <section class="space-y-1 rounded-xl border border-border/80 bg-card p-2 shadow-sm">
    <Button
      type="button"
      variant="outline"
      class="h-10 w-full justify-start rounded-lg"
      @click="open()"
    >
      <Search class="size-4" aria-hidden="true" />
      {{ $t('widget.searchArticles') }}
    </Button>
    <Button
      v-for="article in help.featured"
      :key="article.id"
      type="button"
      variant="ghost"
      class="w-full h-auto justify-start px-2 py-1.5 text-left whitespace-normal"
      @click="open(article)"
      >{{ article.title }}</Button
    >
  </section>
</template>
