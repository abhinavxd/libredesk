<script setup>
import { cn } from '../../../lib/utils';
import { DotsHorizontalIcon } from '@radix-icons/vue';
import { PaginationEllipsis } from 'reka-ui';
import { computed } from 'vue';

const props = defineProps({
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  class: { type: null, required: false },
});

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props;

  return delegated;
});
</script>

<template>
  <PaginationEllipsis
    v-bind="delegatedProps"
    :class="cn('w-9 h-9 flex items-center justify-center', props.class)"
  >
    <slot>
      <DotsHorizontalIcon />
    </slot>
  </PaginationEllipsis>
</template>
