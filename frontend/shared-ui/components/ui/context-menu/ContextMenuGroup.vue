<script setup>
import { ContextMenuGroup } from "reka-ui";

const props = defineProps({
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
});
</script>

<template>
  <ContextMenuGroup v-bind="props">
    <slot />
  </ContextMenuGroup>
</template>
