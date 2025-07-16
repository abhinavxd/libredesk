<template>
  <div :class="containerClass" role="status" :aria-label="text || 'Loading'">
    <div class="flex items-center justify-center gap-2">
      <div :class="spinnerClass"></div>
      <span v-if="text" :class="textClass">{{ text }}</span>
      <span v-if="!text" class="sr-only">{{ $t('globals.terms.loading') }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  size: {
    type: String,
    default: 'md',
    validator: (value) => ['xs', 'sm', 'md', 'lg'].includes(value)
  },
  variant: {
    type: String,
    default: 'primary',
    validator: (value) => ['primary', 'muted'].includes(value)
  },
  text: {
    type: String,
    default: ''
  },
  inline: {
    type: Boolean,
    default: false
  },
  center: {
    type: Boolean,
    default: true
  },
  absolute: {
    type: Boolean,
    default: true
  }
})

const sizeClasses = {
  xs: 'w-3 h-3 border',
  sm: 'w-4 h-4 border-2',
  md: 'w-5 h-5 border-2',
  lg: 'w-8 h-8 border-2'
}

const colorClasses = {
  primary: 'border-muted-foreground border-t-primary',
  muted: 'border-muted border-t-muted-foreground'
}

const containerClass = computed(() => {
  const classes = []

  if (props.absolute) {
    classes.push('absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 z-50')
  } else {
    if (props.inline) {
      classes.push('inline-flex items-center gap-2')
    } else {
      classes.push('flex items-center gap-2')
    }

    if (props.center) {
      classes.push('justify-center')
    }
  }

  return classes.join(' ')
})

const spinnerClass = computed(() => {
  const sizeClass = sizeClasses[props.size] || sizeClasses.md
  const colorClass = colorClasses[props.variant]

  return `${sizeClass} ${colorClass} rounded-full animate-spin`
})

const textClass = computed(() => {
  const sizeMap = {
    xs: 'text-xs',
    sm: 'text-sm',
    md: 'text-sm',
    lg: 'text-base'
  }

  return `${sizeMap[props.size] || 'text-sm'} text-muted-foreground`
})
</script>
