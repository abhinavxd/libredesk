<template>
  <div class="space-y-3">
    <TreeNode
      v-for="element in collections"
      :key="element.id"
      :item="element"
      :selected-item="selectedItem"
      :level="0"
      @select="$emit('select', $event)"
      @create-collection="$emit('create-collection', $event)"
      @create-article="$emit('create-article', $event)"
      @edit="$emit('edit', $event)"
      @delete="$emit('delete', $event)"
      @toggle-status="$emit('toggle-status', $event)"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import TreeNode from './TreeNode.vue'

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

defineEmits([
  'select',
  'create-collection',
  'create-article',
  'edit',
  'delete',
  'toggle-status'
])

const collections = computed(() => props.data.map((item) => ({ ...item, type: 'collection' })))
</script>
