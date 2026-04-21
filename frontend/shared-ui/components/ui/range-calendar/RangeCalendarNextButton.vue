<script setup>
import { reactiveOmit } from "@vueuse/core";
import { ChevronRightIcon } from '@radix-icons/vue';
import { RangeCalendarNext, useForwardProps } from "reka-ui";
import { cn } from '@shared-ui/lib/utils';
import { buttonVariants } from '@shared-ui/components/ui/button';

const props = defineProps({
  nextPage: { type: Function, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  class: { type: null, required: false },
});

const delegatedProps = reactiveOmit(props, "class");

const forwardedProps = useForwardProps(delegatedProps);
</script>

<template>
  <RangeCalendarNext
    :class="
      cn(
        buttonVariants({ variant: 'outline' }),
        'h-7 w-7 bg-transparent p-0 opacity-50 hover:opacity-100',
        props.class,
      )
    "
    v-bind="forwardedProps"
  >
    <slot>
      <ChevronRightIcon class="h-4 w-4" />
    </slot>
  </RangeCalendarNext>
</template>
