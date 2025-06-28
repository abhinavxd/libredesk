<script setup>
import { computed, toRefs } from 'vue'
import { toTypedSchema } from '@vee-validate/zod'
import { getBaseSchema, getBaseType, getDefaultValueInZodStack, getObjectFormSchema } from './utils'
import AutoFormField from './AutoFormField.vue'
import { provideDependencies } from './dependencies'
import { Form } from '../form'

const props = defineProps({
  schema: { type: null, required: true },
  form: { type: Object, required: false },
  fieldConfig: { type: null, required: false },
  dependencies: { type: Array, required: false }
})

const emits = defineEmits(['submit'])

const { dependencies } = toRefs(props)
provideDependencies(dependencies)

const shapes = computed(() => {
  const val = {}
  const baseSchema = getObjectFormSchema(props.schema)
  const shape = baseSchema.shape
  Object.keys(shape).forEach((name) => {
    const item = shape[name]
    const baseItem = getBaseSchema(item)
    let options = baseItem && 'values' in baseItem._def ? baseItem._def.values : undefined
    if (!Array.isArray(options) && typeof options === 'object') options = Object.values(options)

    val[name] = {
      type: getBaseType(item),
      default: getDefaultValueInZodStack(item),
      options,
      required: !['ZodOptional', 'ZodNullable'].includes(item._def.typeName),
      schema: baseItem
    }
  })
  return val
})

const fields = computed(() => {
  const val = {}
  for (const key in shapes.value) {
    const shape = shapes.value[key]
    val[key] = {
      shape,
      config: props.fieldConfig?.[key],
      fieldName: key
    }
  }
  return val
})

const formComponent = computed(() => (props.form ? 'form' : Form))
const formComponentProps = computed(() => {
  if (props.form) {
    return {
      onSubmit: props.form.handleSubmit((val) => emits('submit', val))
    }
  } else {
    const formSchema = toTypedSchema(props.schema)
    return {
      keepValues: true,
      validationSchema: formSchema,
      onSubmit: (val) => emits('submit', val)
    }
  }
})
</script>

<template>
  <component :is="formComponent" v-bind="formComponentProps">
    <slot name="customAutoForm" :fields="fields">
      <template v-for="(shape, key) of shapes" :key="key">
        <slot
          :shape="shape"
          :name="key.toString()"
          :field-name="key.toString()"
          :config="fieldConfig?.[key]"
        >
          <AutoFormField :config="fieldConfig?.[key]" :field-name="key.toString()" :shape="shape" />
        </slot>
      </template>
    </slot>

    <slot :shapes="shapes" />
  </component>
</template>
