<script setup>
import { ChevronRight, FileText } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'

defineProps({
  collection: { type: Object, default: null },
  collections: { type: Array, default: () => [] },
  articles: { type: Array, default: () => [] },
  searching: { type: Boolean, default: false }
})
defineEmits(['collection', 'article'])
</script>

<template>
  <div>
    <div v-if="!searching" class="px-4 py-5 space-y-2">
      <template v-if="collection">
        <h3 class="text-lg font-semibold break-words">{{ collection.name }}</h3>
        <p
          v-if="collection.description"
          class="text-sm text-muted-foreground whitespace-pre-line break-words"
        >
          {{ collection.description }}
        </p>
        <p class="text-xs text-muted-foreground">
          {{
            $t(
              'globals.messages.articleCount',
              { count: collection.article_count },
              collection.article_count
            )
          }}
        </p>
      </template>
      <h3 v-else class="text-sm font-medium">
        {{
          $t('globals.messages.collectionCount', { count: collections.length }, collections.length)
        }}
      </h3>
    </div>
    <ul class="px-2 divide-y divide-border">
      <li v-for="item in articles" :key="`article-${item.id}`">
        <Button
          type="button"
          variant="ghost"
          class="w-full h-auto min-h-12 justify-start gap-3 py-3 px-2 text-left whitespace-normal"
          :data-help-article="item.id"
          @click="$emit('article', item)"
        >
          <FileText class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <span class="flex-1 min-w-0 break-words">{{ item.title }}</span>
          <ChevronRight class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
        </Button>
      </li>
      <li v-for="child in searching ? [] : collections" :key="`collection-${child.id}`">
        <Button
          type="button"
          variant="ghost"
          class="w-full h-auto justify-start gap-3 py-4 px-2 text-left whitespace-normal"
          :data-help-collection="child.id"
          @click="$emit('collection', child)"
        >
          <span class="flex-1 min-w-0 space-y-1.5">
            <span class="block font-medium break-words">{{ child.name }}</span>
            <span
              v-if="child.description"
              class="block text-sm font-normal text-muted-foreground whitespace-pre-line break-words"
              >{{ child.description }}</span
            >
            <span class="block text-xs font-normal text-muted-foreground">{{
              $t(
                'globals.messages.articleCount',
                { count: child.article_count },
                child.article_count
              )
            }}</span>
          </span>
          <ChevronRight class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
        </Button>
      </li>
    </ul>
  </div>
</template>
