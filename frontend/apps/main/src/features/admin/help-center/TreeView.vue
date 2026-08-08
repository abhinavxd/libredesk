<template>
  <Draggable
    v-model="collections"
    class="space-y-3"
    item-key="id"
    handle=".drag-handle"
    group="collections-root"
    :force-fallback="true"
    fallback-on-body
    :fallback-tolerance="3"
    :scroll-sensitivity="120"
    :scroll-speed="18"
    ghost-class="tree-node--ghost"
    @end="emitCollectionOrder"
  >
    <template #item="{ element }">
      <TreeNode
        :item="element"
        :selected-item="selectedItem"
        @select="$emit('select', $event)"
        @create-collection="$emit('create-collection', $event)"
        @create-article="$emit('create-article', $event)"
        @edit="$emit('edit', $event)"
        @delete="$emit('delete', $event)"
        @toggle-status="$emit('toggle-status', $event)"
        @reorder-collections="$emit('reorder-collections', $event)"
        @reorder-articles="$emit('reorder-articles', $event)"
        @move-article="$emit('move-article', $event)"
        @move="moveCollection"
      />
    </template>
  </Draggable>
</template>

<script setup>
import { ref, watch } from 'vue'
import Draggable from 'vuedraggable'
import TreeNode from './TreeNode.vue'
import { moveWithin } from './treeReorder.js'

const props = defineProps({
  data: {
    type: Array,
    required: true
  },
  selectedItem: {
    type: Object,
    default: null
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
  'move-article'
])

const collections = ref([])

watch(
  () => props.data,
  (data) => {
    collections.value = data.map((item) => ({ ...item, type: 'collection' }))
  },
  { immediate: true }
)

const moveCollection = ({ item, direction }) => {
  if (!moveWithin(collections.value, item.id, direction)) return
  emitCollectionOrder()
}

const emitCollectionOrder = () => {
  emit(
    'reorder-collections',
    collections.value.map((collection) => collection.id)
  )
}
</script>
