<script setup>
import { ContextMenuPortal } from "reka-ui";

const props = defineProps({
  to: { type: null, required: false },
  disabled: { type: Boolean, required: false },
  defer: { type: Boolean, required: false },
  forceMount: { type: Boolean, required: false },
});
</script>

<template>
  <ContextMenuPortal v-bind="props">
    <slot />
  </ContextMenuPortal>
</template>
