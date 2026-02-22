<template>
  <Transition
    enter-active-class="transition ease-out duration-200"
    enter-from-class="opacity-0 translate-y-1"
    enter-to-class="opacity-100 translate-y-0"
    leave-active-class="transition ease-in duration-150"
    leave-from-class="opacity-100 translate-y-0"
    leave-to-class="opacity-0 translate-y-1"
  >
    <div v-show="!isAtBottom" class="absolute bottom-5 right-6 z-10">
      <button
        @click="$emit('scroll-to-bottom')"
        class="w-10 h-10 rounded-full flex items-center justify-center shadow-lg border bg-background text-primary transition-colors duration-200 hover:bg-accent dark:hover:bg-gray-700"
      >
        <ChevronDown class="w-4 h-4" />
      </button>
      <span
        v-if="unreadCount > 0"
        class="absolute -top-1 -right-1 min-w-[20px] h-5 px-1.5 rounded-full bg-green-500 text-primary-foreground text-xs font-medium flex items-center justify-center"
      >
        {{ unreadCount }}
      </span>
    </div>
  </Transition>
</template>

<script setup>
import { ChevronDown } from 'lucide-vue-next'

defineProps({
  isAtBottom: {
    type: Boolean,
    required: true
  },
  unreadCount: {
    type: Number,
    default: 0
  }
})

defineEmits(['scroll-to-bottom'])
</script>
