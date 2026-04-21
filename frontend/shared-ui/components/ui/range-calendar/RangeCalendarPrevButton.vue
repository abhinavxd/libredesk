<script setup>
import { reactiveOmit } from "@vueuse/core";
import { ChevronLeftIcon } from '@radix-icons/vue';
import { RangeCalendarPrev, useForwardProps } from "reka-ui";
import { cn } from '@shared-ui/lib/utils';
import { buttonVariants } from '@shared-ui/components/ui/button';

const props = defineProps({
  prevPage: { type: Function, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  class: { type: null, required: false },
});

const delegatedProps = reactiveOmit(props, "class");

const forwardedProps = useForwardProps(delegatedProps);
</script>

<template>
  <RangeCalendarPrev
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
      <ChevronLeftIcon class="h-4 w-4" />
    </slot>
  </RangeCalendarPrev>
</template>
