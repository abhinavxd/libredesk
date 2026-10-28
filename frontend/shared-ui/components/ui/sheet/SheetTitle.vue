<script setup>
import { reactiveOmit } from '@vueuse/core';
import { DialogTitle } from 'radix-vue';
import { cn } from '@shared-ui/lib/utils';

const props = defineProps({
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  class: { type: null, required: false },
});

const delegatedProps = reactiveOmit(props, 'class');
</script>

<template>
  <DialogTitle
    :class="cn('ui-dialog-heading', props.class)"
    v-bind="delegatedProps"
  >
    <slot />
  </DialogTitle>
</template>
