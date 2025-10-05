<script setup>
import { computed } from 'vue'
import { DEFAULT_ZOD_HANDLERS, INPUT_COMPONENTS } from './constant'
import useDependencies from './dependencies'

const props = defineProps({
  fieldName: { type: String, required: true },
  shape: { type: Object, required: true },
  config: { type: null, required: false }
})

function isValidConfig(config) {
  return !!config?.component
}

const delegatedProps = computed(() => {
  if (['ZodObject', 'ZodArray'].includes(props.shape?.type)) return { schema: props.shape?.schema }
  return undefined
})

const { isDisabled, isHidden, isRequired, overrideOptions } = useDependencies(props.fieldName)
</script>

<template>
  <component
    :is="
      isValidConfig(config)
        ? typeof config.component === 'string'
          ? INPUT_COMPONENTS[config.component]
          : config.component
        : INPUT_COMPONENTS[DEFAULT_ZOD_HANDLERS[shape.type]]
    "
    v-if="!isHidden"
    :field-name="fieldName"
    :label="shape.schema?.description"
    :required="isRequired || shape.required"
    :options="overrideOptions || shape.options"
    :disabled="isDisabled"
    :config="config"
    v-bind="delegatedProps"
  >
    <slot />
  </component>
</template>
