<template>
  <Card
    class="flex flex-col p-4 max-w-xs cursor-pointer transition-shadow hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
    role="button"
    tabindex="0"
    @keydown.enter.prevent="handleClick"
    @keydown.space.prevent="handleClick"
    @click="handleClick"
  >
    <div class="flex items-center mb-2">
      <img v-if="typeof icon === 'string'" :src="icon" class="size-6 shrink-0 mr-2" alt="" />
      <component
        v-else
        :is="icon"
        size="24"
        class="size-6 shrink-0 mr-2 text-primary"
        aria-hidden="true"
      />
      <h3 class="text-lg font-medium">{{ title }}</h3>
      <BetaBadge v-if="badge" class="ml-2">{{ badge }}</BetaBadge>
    </div>
    <p class="text-sm text-muted-foreground">{{ subTitle }}</p>
  </Card>
</template>

<script setup>
import BetaBadge from '@main/components/BetaBadge.vue'
import { Card } from '@shared-ui/components/ui/card'

const props = defineProps({
  title: String,
  subTitle: String,
  icon: [Function, String, Object],
  onClick: Function,
  badge: String
})

const emit = defineEmits(['click'])

const handleClick = () => {
  props.onClick?.()
  emit('click')
}
</script>
