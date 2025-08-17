<template>
  <div>
    <!-- Collection Node -->
    <Collapsible v-if="item.type === 'collection'" v-model:open="isOpen">
      <div
        class="group tree-node"
        :class="{
          'tree-node--selected': isSelected,
          'hover:shadow-sm': !isSelected
        }"
        @click="selectItem"
      >
        <div class="flex items-center gap-3">
          <CollapsibleTrigger as-child @click.stop>
            <ChevronRightIcon
              class="h-4 w-4 transition-transform text-muted-foreground hover:text-foreground flex-shrink-0"
              :class="{ 'rotate-90': isOpen }"
            />
          </CollapsibleTrigger>

          <div class="icon-container-folder">
            <FolderIcon class="h-4.5 w-4.5 text-blue-600" />
          </div>

          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-1">
              <h4 class="text-sm font-semibold truncate text-foreground">
                {{ item.name }}
              </h4>
              <span
                v-if="!item.is_published"
                class="text-[10px] font-medium bg-yellow-100 text-yellow-800 px-1.5 py-0.5 rounded"
              >
                Draft
              </span>
            </div>
            <p v-if="item.description" class="text-xs text-muted-foreground leading-tight line-clamp-2 max-w-xs">
              {{ item.description }}
            </p>
          </div>

          <div class="hover-actions ml-2">
            <Badge
              v-if="item.articles && item.articles.length > 0"
              variant="outline"
              class="text-xs px-2 py-0.5 font-normal bg-card/50 text-muted-foreground"
            >
              {{ item.articles.length }} {{ item.articles.length === 1 ? 'article' : 'articles' }}
            </Badge>

            <TreeDropdown
              :item="item"
              @create-collection="$emit('create-collection', item.id)"
              @create-article="$emit('create-article', item)"
              @edit="$emit('edit', $event)"
              @delete="$emit('delete', $event)"
              @toggle-status="$emit('toggle-status', $event)"
            />
          </div>
        </div>
      </div>

      <!-- Child Collections and Articles -->
      <CollapsibleContent>
        <div class="ml-10 mt-2 pl-2 border-l border-border/20">
          <!-- Empty no child content -->
          <div
            v-if="!childCollections.length && !articles.length"
            class="text-sm text-muted-foreground bg-muted/10 rounded-md py-3 px-4 text-center italic"
          >
            <FolderOpenIcon class="h-4 w-4 mx-auto mb-1.5 opacity-60" />
            {{ $t('globals.messages.empty') }}
          </div>

          <!-- Articles -->
          <div class="space-y-1.5">
            <div
              v-for="element in articles"
              :key="element.id"
              class="group tree-node--article"
              :class="{
                'tree-node--selected':
                  selectedItem?.id === element.id && selectedItem?.type === 'article'
              }"
              @click="selectArticle(element)"
            >
              <div class="flex items-center gap-2">
                <div class="icon-container-article">
                  <DocumentTextIcon class="h-4 w-4 text-green-600" />
                </div>

                <div class="flex-1 min-w-0">
                  <h5 class="text-sm font-medium truncate text-foreground">
                    {{ element.title }}
                  </h5>
                  <p
                    v-if="element.description"
                    class="text-xs text-muted-foreground truncate mt-0.5"
                  >
                    {{ element.description }}
                  </p>
                </div>

                <div class="hover-actions--compact">
                  <Badge
                    :variant="getArticleStatusVariant(element.status)"
                    class="text-[11px] px-1.5 py-0.5 font-normal"
                    v-if="element.status"
                  >
                    {{ element.status.charAt(0).toUpperCase() + element.status.slice(1) }}
                  </Badge>

                  <TreeDropdown
                    :item="{ ...element, type: 'article' }"
                    @edit="$emit('edit', $event)"
                    @delete="$emit('delete', $event)"
                    @toggle-status="$emit('toggle-status', $event)"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- Child Collections -->
          <div class="space-y-1.5">
            <TreeNode
              v-for="element in childCollections"
              :key="element.id"
              :item="{ ...element, type: 'collection' }"
              :selected-item="selectedItem"
              :level="level + 1"
              @select="$emit('select', $event)"
              @create-collection="$emit('create-collection', $event)"
              @create-article="$emit('create-article', $event)"
              @edit="$emit('edit', $event)"
              @delete="$emit('delete', $event)"
              @toggle-status="$emit('toggle-status', $event)"
            />
          </div>
        </div>
      </CollapsibleContent>
    </Collapsible>

    <!-- Article Node (when at root level) -->
    <div
      v-else
      class="group tree-node--article"
      :class="{
        'tree-node--selected': isSelected,
        'hover:shadow-xs': !isSelected
      }"
      @click="selectItem"
    >
      <div class="flex items-center gap-2">
        <div class="icon-container-article">
          <DocumentTextIcon class="h-4 w-4 text-green-600" />
        </div>

        <div class="flex-1 min-w-0">
          <h5 class="text-sm font-medium truncate text-foreground">
            {{ item.title }}
          </h5>
          <p v-if="item.description" class="text-xs text-muted-foreground truncate mt-0.5">
            {{ item.description }}
          </p>
        </div>

        <div class="hover-actions--compact">
          <Badge
            :variant="getArticleStatusVariant(item.status)"
            class="text-[11px] px-1.5 py-0.5 font-normal"
          >
            {{ item.status }}
          </Badge>

          <TreeDropdown
            :item="item"
            @edit="$emit('edit', $event)"
            @delete="$emit('delete', $event)"
            @toggle-status="$emit('toggle-status', $event)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger
} from '@shared-ui/components/ui/collapsible'
import {
  ChevronRight as ChevronRightIcon,
  FileText as DocumentTextIcon,
  Folder as FolderIcon,
  FolderOpen as FolderOpenIcon
} from 'lucide-vue-next'
import TreeDropdown from './TreeDropdown.vue'

const props = defineProps({
  item: {
    type: Object,
    required: true
  },
  selectedItem: {
    type: Object,
    default: null
  },
  level: {
    type: Number,
    default: 0
  }
})

const emit = defineEmits([
  'select',
  'create-collection',
  'create-article',
  'edit',
  'delete',
  'toggle-status'
])

const isOpen = ref(true)

const isSelected = computed(() => {
  if (!props.selectedItem) return false
  return props.selectedItem.id === props.item.id && props.selectedItem.type === props.item.type
})

const childCollections = computed(() => props.item.children || [])
const articles = computed(() => props.item.articles || [])

const selectItem = () => {
  emit('select', props.item)
}

const selectArticle = (article) => {
  emit('select', { ...article, type: 'article' })
}

const getArticleStatusVariant = (status) => {
  switch (status) {
    case 'published':
      return 'default'
    case 'draft':
      return 'secondary'
    default:
      return 'secondary'
  }
}

</script>

<style scoped>
.tree-node {
  @apply border border-transparent hover:border-border hover:bg-muted/20 rounded-lg p-3 transition-all duration-200 cursor-pointer;
}

.tree-node--article {
  @apply border border-transparent hover:border-border hover:bg-muted/20 rounded-md p-2.5 transition-all duration-200 cursor-pointer;
}

.tree-node--selected {
  @apply bg-accent/10 border-border shadow-sm ring-1 ring-accent/20;
}

.icon-container-folder {
  @apply flex items-center justify-center w-9 h-9 rounded-lg bg-blue-50 border border-blue-100/70;
}

.icon-container-article {
  @apply flex items-center justify-center w-7 h-7 rounded-md bg-green-50 border border-green-100/70;
}

.hover-actions {
  @apply flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity duration-150;
}

.hover-actions--compact {
  @apply flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-150;
}
</style>
