<script setup>
import { ContextMenuRadioGroup, useForwardPropsEmits } from "reka-ui";

const props = defineProps({
  modelValue: { type: String, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
});
const emits = defineEmits(["update:modelValue"]);

const forwarded = useForwardPropsEmits(props, emits);
</script>

<template>
  <ContextMenuRadioGroup v-bind="forwarded">
    <slot />
  </ContextMenuRadioGroup>
</template>
