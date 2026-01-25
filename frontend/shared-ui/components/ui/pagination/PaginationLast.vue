<script setup>
import { cn } from '../../../lib/utils';
import { Button } from '../button';
import { ChevronsRight } from 'lucide-vue-next';
import { PaginationLast } from 'reka-ui';
import { computed } from 'vue';

const props = defineProps({
  asChild: { type: Boolean, required: false, default: true },
  as: { type: null, required: false },
  class: { type: null, required: false },
});

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props;

  return delegated;
});
</script>

<template>
  <PaginationLast v-bind="delegatedProps">
    <Button :class="cn('w-9 h-9 p-0', props.class)" variant="outline">
      <slot>
        <ChevronsRight />
      </slot>
    </Button>
  </PaginationLast>
</template>
