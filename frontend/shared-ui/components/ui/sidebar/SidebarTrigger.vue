<script setup>
import { ViewVerticalIcon } from '@radix-icons/vue';
import { cn } from '../../../lib/utils';
import { Button } from '../button';
import { useSidebar } from './utils';

const props = defineProps({
  class: { type: null, required: false },
});

const { toggleSidebar } = useSidebar();
</script>

<template>
  <Button
    data-sidebar="trigger"
    variant="ghost"
    size="icon"
    :class="cn('h-11 w-11 md:h-7 md:w-7', props.class)"
    @click="toggleSidebar"
  >
    <ViewVerticalIcon />
    <span class="sr-only">Toggle Sidebar</span>
  </Button>
</template>
