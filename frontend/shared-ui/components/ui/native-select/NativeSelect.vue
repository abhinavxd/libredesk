<script setup>
import { useVModel } from '@vueuse/core';
import { ChevronDown } from 'lucide-vue-next';
import { cn } from '../../../lib/utils';

// A styled native <select>, for short option lists where the platform picker is
// the better control (it is the system sheet on phones). Matches Input.
defineOptions({ inheritAttrs: false });

const props = defineProps({
  modelValue: { type: [String, Number], required: false },
  // Declared so v-model.number stays off the DOM; Vue applies it on emit.
  modelModifiers: { type: Object, required: false },
  class: { type: null, required: false },
});

const emits = defineEmits(['update:modelValue']);

const modelValue = useVModel(props, 'modelValue', emits, { passive: true });
</script>

<template>
  <div :class="cn('relative w-full', props.class)">
    <select
      v-model="modelValue"
      v-bind="$attrs"
      class="flex h-9 w-full appearance-none rounded-md border border-input bg-transparent py-1 pl-3 pr-8 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 [&>option]:bg-popover [&>option]:text-popover-foreground"
    >
      <slot />
    </select>
    <ChevronDown
      class="pointer-events-none absolute right-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
      aria-hidden="true"
    />
  </div>
</template>
