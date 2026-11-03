<script setup>
import { Button } from '@shared-ui/components/ui/button'

const props = defineProps({
  items: { type: Array, required: true, default: () => [] },
  showLineStyles: { type: Boolean, default: false }
})

const emits = defineEmits(['legendItemClick', 'update:items'])

function onLegendItemClick(d, i) {
  emits('legendItemClick', d, i)
  const isBulletActive = !props.items[i].inactive
  const isFilterApplied = props.items.some((i) => i.inactive)
  if (isFilterApplied && isBulletActive) {
    emits(
      'update:items',
      props.items.map((item) => ({ ...item, inactive: false }))
    )
  } else {
    emits(
      'update:items',
      props.items.map((item) =>
        item.name === d.name ? { ...d, inactive: false } : { ...item, inactive: true }
      )
    )
  }
}
</script>

<template>
  <div class="flex max-w-full flex-wrap justify-end gap-1">
    <Button
      v-for="(item, index) in items"
      :key="item.name"
      type="button"
      variant="ghost"
      size="xs"
      class="h-auto min-h-7 whitespace-normal text-left"
      :class="{ 'text-muted-foreground': item.inactive }"
      :aria-pressed="!item.inactive"
      @click="onLegendItemClick(item, index)"
    >
      <span
        v-if="showLineStyles"
        class="w-4 shrink-0 border-t-2"
        :class="{ 'opacity-30': item.inactive }"
        :style="{ borderTopColor: item.color, borderTopStyle: index % 2 ? 'dashed' : 'solid' }"
        aria-hidden="true"
      />
      <span
        v-else
        class="size-2.5 shrink-0 rounded-full"
        :class="{ 'opacity-30': item.inactive }"
        :style="{ backgroundColor: item.color }"
        aria-hidden="true"
      />
      {{ item.name }}
    </Button>
  </div>
</template>
