<script setup>
import { Switch } from '@shared-ui/components/ui/switch'
import { useId } from 'vue'

defineProps({
  title: { type: String, default: '' },
  description: { type: String, default: '' },
  checked: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false }
})

defineEmits(['update:checked'])

const titleId = useId()
const descriptionId = useId()
</script>

<template>
  <div class="space-y-2">
    <p :id="titleId" class="text-sm font-medium text-foreground">{{ title }}</p>
    <div class="flex items-center gap-3">
      <Switch
        :checked="checked"
        :disabled="disabled"
        :aria-labelledby="titleId"
        :aria-describedby="description ? descriptionId : undefined"
        @update:checked="$emit('update:checked', $event)"
      />
      <p :id="descriptionId" class="text-sm text-muted-foreground">{{ description }}</p>
    </div>
  </div>
</template>
