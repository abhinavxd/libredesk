<script setup>
import { Primitive } from 'reka-ui';
import { cn } from '../../../lib/utils';
import { sidebarMenuButtonVariants } from '.';

const props = defineProps({
  variant: { type: null, required: false, default: 'default' },
  size: { type: null, required: false, default: 'default' },
  isActive: { type: Boolean, required: false },
  class: { type: null, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false, default: 'button' },
});
</script>

<template>
  <Primitive
    data-sidebar="menu-button"
    :data-size="size"
    :data-active="isActive"
    :class="cn(sidebarMenuButtonVariants({ variant, size }), props.class)"
    :as="as"
    :as-child="asChild"
    v-bind="$attrs"
  >
    <slot />
  </Primitive>
</template>
