<template>
  <Button
    type="button"
    variant="outline"
    class="h-full w-full min-w-0 flex-col items-stretch whitespace-normal bg-card p-4 text-left font-normal text-card-foreground"
    @click="handleClick">
    <span class="mb-2 flex items-center">
      <img v-if="typeof icon === 'string'" :src="icon" alt="" class="w-6 h-6 mr-2" />
      <component v-else :is="icon" size="24" class="mr-2 text-primary" aria-hidden="true" />
      <span class="ui-heading">{{ title }}</span>
      <BetaBadge v-if="badge" class="ml-2">{{ badge }}</BetaBadge>
    </span>
    <span class="ui-description">{{ subTitle }}</span>
  </Button>
</template>

<script setup>
import BetaBadge from '@main/components/BetaBadge.vue'
import { Button } from '@shared-ui/components/ui/button'

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
