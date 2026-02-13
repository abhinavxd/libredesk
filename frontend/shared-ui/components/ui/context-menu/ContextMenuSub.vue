<script setup>
import { ContextMenuSub, useForwardPropsEmits } from "reka-ui";

const props = defineProps({
  defaultOpen: { type: Boolean, required: false },
  open: { type: Boolean, required: false },
});
const emits = defineEmits(["update:open"]);

const forwarded = useForwardPropsEmits(props, emits);
</script>

<template>
  <ContextMenuSub v-bind="forwarded">
    <slot />
  </ContextMenuSub>
</template>
