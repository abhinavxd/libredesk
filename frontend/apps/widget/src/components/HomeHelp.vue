<script setup>
import { useHelpStore } from '@widget/store/help.js'
import { useWidgetStore } from '@widget/store/widget.js'
import { Button } from '@shared-ui/components/ui/button'
import { Search, FileText } from 'lucide-vue-next'
const help = useHelpStore()
const widget = useWidgetStore()
const open = (article) => {
  help.article = null
  help.query = ''
  help.results = null
  help.pendingArticle = article || null
  widget.navigateToHelp()
}
</script>

<template>
  <section class="border rounded-lg p-3 space-y-2 bg-card">
    <Button type="button" variant="outline" class="w-full justify-start" @click="open()"><Search class="size-4" />{{ $t('widget.searchArticles') }}</Button>
    <Button v-for="article in help.featured" :key="article.id" type="button" variant="ghost" class="w-full h-auto justify-start text-left whitespace-normal" @click="open(article)"><FileText class="size-4 shrink-0" />{{ article.title }}</Button>
  </section>
</template>
