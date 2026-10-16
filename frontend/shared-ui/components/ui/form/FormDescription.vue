<script setup>
import { useFormField } from './useFormField'
import { cn } from '@shared-ui/lib/utils'

const props = defineProps({
  class: { type: null, required: false }
})

const { formDescriptionId } = useFormField()
</script>

<template>
  <p :id="formDescriptionId" :class="cn('ui-description', props.class)">
    <slot />
  </p>
</template>
