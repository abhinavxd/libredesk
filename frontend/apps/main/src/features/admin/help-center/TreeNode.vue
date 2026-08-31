<template>
  <Collapsible v-if="item.type === 'collection'" v-model:open="isOpen" as-child>
    <section :class="depth === 0 ? 'overflow-hidden rounded-lg border bg-card' : 'border-t'">
      <div
        class="group tree-node tree-node--collection"
        :class="{ 'tree-node--selected': isSelected }"
        :style="indentStyle"
      >
        <GripVertical
          class="drag-handle -m-1 h-4 w-4 flex-shrink-0 cursor-grab p-1 text-muted-foreground opacity-0 group-hover:opacity-100 box-content"
          aria-hidden="true"
          @click.stop
        />

        <CollapsibleTrigger as-child @click.stop>
          <button
            type="button"
            class="-m-1 flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-muted hover:text-foreground"
            :aria-label="isOpen ? t('globals.terms.collapse') : t('globals.terms.expand')"
          >
            <ChevronRight class="h-4 w-4 transition-transform" :class="{ 'rotate-90': isOpen }" />
          </button>
        </CollapsibleTrigger>

        <Folder class="h-4 w-4 flex-shrink-0 text-primary" />

        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <button
              type="button"
              class="tree-node-title truncate text-sm font-medium text-foreground"
              @click="selectItem"
            >
              {{ item.name }}
            </button>
            <Badge v-if="!item.is_published" variant="secondary" class="flex-shrink-0 font-normal">
              {{ $t('globals.terms.draft') }}
            </Badge>
          </div>
          <p v-if="item.description" class="truncate text-xs text-muted-foreground">
            {{ item.description }}
          </p>
        </div>

        <span v-if="item.article_count" class="flex-shrink-0 text-xs text-muted-foreground">
          {{ item.article_count }} {{ $t('globals.terms.article', item.article_count) }}
        </span>

        <span class="hover-actions flex-shrink-0">
          <TreeDropdown
            :item="item"
            :can-create-collection="depth < 2"
            :is-first="isFirst"
            :is-last="isLast"
            @create-collection="$emit('create-collection', item.id)"
            @create-article="$emit('create-article', item)"
            @edit="$emit('edit', $event)"
            @delete="$emit('delete', $event)"
            @toggle-status="$emit('toggle-status', $event)"
            @move="$emit('move', $event)"
          />
        </span>
      </div>

      <CollapsibleContent>
        <Draggable
          v-model="articles"
          class="divide-y border-t empty:border-t-0"
          :class="{ 'empty:h-8': articleDragging }"
          item-key="id"
          handle=".drag-handle"
          group="help-center-articles"
          @start="articleDragging = true"
          @end="articleDragging = false"
          :force-fallback="true"
          fallback-on-body
          :fallback-tolerance="3"
          :scroll-sensitivity="120"
          :scroll-speed="18"
          ghost-class="tree-node--ghost"
          @change="onArticleChange"
        >
          <template #header>
            <p
              v-if="!childCollections.length && !articles.length"
              class="px-4 py-2.5 text-sm text-muted-foreground"
              :style="childIndentStyle"
            >
              {{ $t('globals.messages.empty', { name: $t('globals.terms.collection') }) }}
            </p>
          </template>

          <template #item="{ element, index }">
            <TreeNode
              :item="{ ...element, type: 'article' }"
              :selected-item="selectedItem"
              :depth="depth + 1"
              :is-first="index === 0"
              :is-last="index === articles.length - 1"
              @select="$emit('select', $event)"
              @edit="$emit('edit', $event)"
              @delete="$emit('delete', $event)"
              @toggle-status="$emit('toggle-status', $event)"
              @move="moveArticle"
            />
          </template>
        </Draggable>

        <Draggable
          v-model="childCollections"
          item-key="id"
          handle=".drag-handle"
          :group="`collections-${item.id}`"
          :force-fallback="true"
          fallback-on-body
          :fallback-tolerance="3"
          :scroll-sensitivity="120"
          :scroll-speed="18"
          ghost-class="tree-node--ghost"
          @end="emitCollectionOrder"
        >
          <template #item="{ element, index }">
            <TreeNode
              :item="{ ...element, type: 'collection' }"
              :selected-item="selectedItem"
              :depth="depth + 1"
              :is-first="index === 0"
              :is-last="index === childCollections.length - 1"
              @select="$emit('select', $event)"
              @create-collection="$emit('create-collection', $event)"
              @create-article="$emit('create-article', $event)"
              @edit="$emit('edit', $event)"
              @delete="$emit('delete', $event)"
              @toggle-status="$emit('toggle-status', $event)"
              @reorder-collections="$emit('reorder-collections', $event)"
              @reorder-articles="$emit('reorder-articles', $event)"
              @move-article="$emit('move-article', $event)"
              @move="moveChildCollection"
            />
          </template>
        </Draggable>
      </CollapsibleContent>
    </section>
  </Collapsible>

  <div
    v-else
    class="group tree-node"
    :class="{ 'tree-node--selected': isSelected }"
    :style="indentStyle"
  >
    <GripVertical
      class="drag-handle -m-1 h-4 w-4 flex-shrink-0 cursor-grab p-1 text-muted-foreground opacity-0 group-hover:opacity-100 box-content"
      aria-hidden="true"
      @click.stop
    />

    <FileText class="h-4 w-4 flex-shrink-0 text-muted-foreground" />

    <div class="min-w-0 flex-1">
      <button
        type="button"
        class="tree-node-title max-w-full truncate text-sm text-foreground"
        @click="selectItem"
      >
        {{ item.title }}
      </button>
    </div>

    <Badge v-if="item.status !== 'published'" variant="secondary" class="flex-shrink-0 font-normal">
      {{ t('globals.terms.draft') }}
    </Badge>

    <span class="hover-actions flex-shrink-0">
      <TreeDropdown
        :item="item"
        :is-first="isFirst"
        :is-last="isLast"
        @edit="$emit('edit', $event)"
        @delete="$emit('delete', $event)"
        @toggle-status="$emit('toggle-status', $event)"
        @move="$emit('move', $event)"
      />
    </span>
  </div>
</template>

<script setup>
import { ref, computed, watch, inject } from 'vue'
import Draggable from 'vuedraggable'
import { useI18n } from 'vue-i18n'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger
} from '@shared-ui/components/ui/collapsible'
import { ChevronRight, FileText, Folder, GripVertical } from 'lucide-vue-next'
import TreeDropdown from './TreeDropdown.vue'
import { moveWithin } from './treeReorder.js'

const { t } = useI18n()

const props = defineProps({
  item: {
    type: Object,
    required: true
  },
  selectedItem: {
    type: Object,
    default: null
  },
  depth: {
    type: Number,
    default: 0
  },
  isFirst: {
    type: Boolean,
    default: false
  },
  isLast: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits([
  'select',
  'create-collection',
  'create-article',
  'edit',
  'delete',
  'toggle-status',
  'reorder-collections',
  'reorder-articles',
  'move-article',
  'move'
])

const collapsedStore = inject('helpCenterTreeCollapsed', null)
const expandSignal = inject('helpCenterTreeExpand', null)
const articleDragging = inject('helpCenterArticleDragging', ref(false))

const isOpen = ref(!collapsedStore?.isCollapsed(props.item.id))

watch(isOpen, (open) => collapsedStore?.setCollapsed(props.item.id, !open))

if (expandSignal) {
  watch(
    () => expandSignal.value.n,
    () => {
      isOpen.value = expandSignal.value.open
    }
  )
}

const isSelected = computed(() => {
  if (!props.selectedItem) return false
  return props.selectedItem.id === props.item.id && props.selectedItem.type === props.item.type
})

const indentStyle = computed(() => ({ paddingLeft: `${0.75 + props.depth * 1.5}rem` }))
const childIndentStyle = computed(() => ({ paddingLeft: `${0.75 + (props.depth + 1) * 1.5}rem` }))

const childCollections = ref([])
const articles = ref([])

watch(
  () => props.item.children,
  (children) => {
    childCollections.value = [...(children || [])]
  },
  { immediate: true }
)

watch(
  () => props.item.articles,
  (items) => {
    articles.value = [...(items || [])]
  },
  { immediate: true }
)

const emitCollectionOrder = () => {
  emit(
    'reorder-collections',
    childCollections.value.map((collection) => collection.id)
  )
}

const emitArticleOrder = () => {
  emit('reorder-articles', {
    collectionId: props.item.id,
    ids: articles.value.map((article) => article.id)
  })
}

// vuedraggable reports the source list's removal and the target list's insertion separately;
// only the target knows the article's new collection.
const onArticleChange = (event) => {
  if (event.added) {
    emit('move-article', {
      articleId: event.added.element.id,
      collectionId: props.item.id,
      ids: articles.value.map((article) => article.id)
    })
    return
  }
  emitArticleOrder()
}

const moveArticle = ({ item, direction }) => {
  if (!moveWithin(articles.value, item.id, direction)) return
  emitArticleOrder()
}

const moveChildCollection = ({ item, direction }) => {
  if (!moveWithin(childCollections.value, item.id, direction)) return
  emitCollectionOrder()
}

const selectItem = () => {
  emit('select', props.item)
}
</script>

<style scoped>
.tree-node {
  @apply flex items-center gap-2.5 px-3 py-2 hover:bg-muted/50;
}

.tree-node-title {
  @apply cursor-pointer text-left hover:underline;
}

.tree-node-title:focus-visible {
  @apply rounded-sm outline-none ring-1 ring-ring;
}

.tree-node--collection {
  @apply bg-muted/40 py-2.5 hover:bg-muted/60;
}

.tree-node--selected {
  @apply bg-accent;
}

.tree-node--ghost {
  @apply bg-accent/60 opacity-60;
}

.hover-actions {
  @apply opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100;
}

/* Touch devices have no hover, so row actions stay visible there. */
@media (hover: none) {
  .drag-handle,
  .hover-actions {
    @apply opacity-100;
  }
}
</style>
