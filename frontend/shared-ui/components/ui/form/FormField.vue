<script setup>
import { Field } from 'vee-validate'

defineOptions({ inheritAttrs: false })

const fieldBindings = (field) => {
  const handleChange = (value, validate = field.meta.validated) =>
    field.handleChange(value, validate)
  return {
    ...field,
    handleChange,
    componentField: {
      ...field.componentField,
      'onUpdate:modelValue': handleChange,
      onInput: handleChange,
      onChange: handleChange
    },
    field: {
      ...field.field,
      onInput: handleChange,
      onChange: handleChange
    }
  }
}
</script>

<template>
  <Field
    v-bind="$attrs"
    :validate-on-blur="false"
    :validate-on-change="false"
    :validate-on-input="false"
    :validate-on-model-update="false"
    v-slot="field"
  >
    <slot v-bind="fieldBindings(field)" />
  </Field>
</template>
