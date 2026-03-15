<script setup>
import { ContextMenuTrigger, useForwardProps } from "reka-ui";

const props = defineProps({
  disabled: { type: Boolean, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
});

const forwardedProps = useForwardProps(props);
</script>

<template>
  <ContextMenuTrigger v-bind="forwardedProps">
    <slot />
  </ContextMenuTrigger>
</template>
