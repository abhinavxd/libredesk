<template>
  <MenuCard @click="handleClick">
    <template #title>
      <BookOpen size="24" class="mr-2 text-primary" />
      {{ helpCenter.name }}
    </template>
    <template #subtitle>
      <p class="text-sm mb-3">{{ helpCenter.page_title }}</p>
    </template>
    <div class="mt-3 pt-3 border-t">
      <div class="flex items-center justify-between text-xs text-muted-foreground">
        <span>{{ helpCenter.view_count || 0 }} views</span>
      </div>
    </div>
  </MenuCard>
</template>

<script setup>
import { defineEmits } from 'vue'
import MenuCard from '@shared-ui/components/ui/menu-card/MenuCard.vue'

import { BookOpen } from 'lucide-vue-next'

const props = defineProps({
  helpCenter: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['edit', 'delete', 'click'])
const handleClick = () => {
  emit('click', props.helpCenter)
}
</script>
