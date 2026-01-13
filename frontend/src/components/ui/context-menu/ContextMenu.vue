<script setup>
import { ContextMenuRoot, useForwardPropsEmits } from "reka-ui";

const props = defineProps({
  dir: { type: String, required: false },
  modal: { type: Boolean, required: false },
});
const emits = defineEmits(["update:open"]);

const forwarded = useForwardPropsEmits(props, emits);
</script>

<template>
  <ContextMenuRoot v-bind="forwarded">
    <slot />
  </ContextMenuRoot>
</template>
