<template>
  <div class="relative">
    <Spinner
      v-if="loading"
      size="sm"
      variant="muted"
      :absolute="false"
      :center="false"
      class="absolute left-3 top-1/2 size-4 -translate-y-1/2 pointer-events-none"
    />
    <Search
      v-else
      class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground pointer-events-none"
      aria-hidden="true"
    />
    <Input
      ref="input"
      :model-value="modelValue"
      :placeholder="placeholder"
      :aria-label="ariaLabel || placeholder"
      :class="['pl-10', modelValue ? 'pr-10' : 'pr-3', inputClass]"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <slot name="trailing" />
    <button
      v-if="modelValue"
      type="button"
      :aria-label="clearLabel"
      class="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent"
      @click="clear"
    >
      <X class="size-4" aria-hidden="true" />
    </button>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Search, X } from 'lucide-vue-next'
import { Input } from '@shared-ui/components/ui/input'
import { Spinner } from '@shared-ui/components/ui/spinner'

defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  ariaLabel: { type: String, default: '' },
  clearLabel: { type: String, default: '' },
  inputClass: { type: [String, Array, Object], default: '' },
  loading: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue', 'clear'])
const input = ref(null)

const clear = () => {
  emit('update:modelValue', '')
  emit('clear')
  input.value?.$el?.focus?.()
}

defineExpose({ input })
</script>
