<script setup>
import { computed } from 'vue';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '../tooltip';
import SidebarMenuButtonChild from './SidebarMenuButtonChild.vue';
import { useSidebar } from './utils';

defineOptions({
  inheritAttrs: false,
});

const props = defineProps({
  variant: { type: null, required: false, default: 'default' },
  size: { type: null, required: false, default: 'default' },
  isActive: { type: Boolean, required: false },
  class: { type: null, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false, default: 'button' },
  tooltip: { type: null, required: false },
});

const { isMobile, state } = useSidebar();

const delegatedProps = computed(() => {
  const { tooltip, ...delegated } = props;
  return delegated;
});
</script>

<template>
  <SidebarMenuButtonChild
    v-if="!tooltip"
    v-bind="{ ...delegatedProps, ...$attrs }"
  >
    <slot />
  </SidebarMenuButtonChild>

  <Tooltip v-else>
    <TooltipTrigger as-child>
      <SidebarMenuButtonChild v-bind="{ ...delegatedProps, ...$attrs }">
        <slot />
      </SidebarMenuButtonChild>
    </TooltipTrigger>
    <TooltipContent
      side="right"
      align="center"
      :hidden="state !== 'collapsed' || isMobile"
    >
      <template v-if="typeof tooltip === 'string'">
        {{ tooltip }}
      </template>
      <component :is="tooltip" v-else />
    </TooltipContent>
  </Tooltip>
</template>
