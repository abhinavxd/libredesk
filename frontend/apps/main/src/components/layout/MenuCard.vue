<template>
  <Card
    class="flex flex-col p-4 max-w-xs cursor-pointer transition-shadow hover:shadow-md"
    @click="handleClick">
    <div class="flex items-center mb-2">
      <img v-if="typeof icon === 'string'" :src="icon" class="w-6 h-6 mr-2" />
      <component v-else :is="icon" size="24" class="mr-2 text-primary" />
      <h3 class="text-lg font-medium">{{ title }}</h3>
      <Badge v-if="badge" variant="secondary" class="ml-2">{{ badge }}</Badge>
    </div>
    <p class="text-sm text-muted-foreground">{{ subTitle }}</p>
  </Card>
</template>

<script setup>
import { Badge } from '@shared-ui/components/ui/badge'
import { Card } from '@shared-ui/components/ui/card'

const props = defineProps({
  title: String,
  subTitle: String,
  icon: [Function, String],
  onClick: Function,
  badge: String
})

const emit = defineEmits(['click'])

const handleClick = () => {
  props.onClick?.()
  emit('click')
}
</script>
