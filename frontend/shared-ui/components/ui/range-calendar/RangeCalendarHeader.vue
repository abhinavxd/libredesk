<script setup>
import { reactiveOmit } from "@vueuse/core";
import { RangeCalendarHeader, useForwardProps } from "reka-ui";
import { cn } from '@shared-ui/lib/utils';

const props = defineProps({
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  class: { type: null, required: false },
});

const delegatedProps = reactiveOmit(props, "class");

const forwardedProps = useForwardProps(delegatedProps);
</script>

<template>
  <RangeCalendarHeader
    :class="
      cn('relative flex w-full items-center justify-between pt-1', props.class)
    "
    v-bind="forwardedProps"
  >
    <slot />
  </RangeCalendarHeader>
</template>
