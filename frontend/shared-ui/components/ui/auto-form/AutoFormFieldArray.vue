<script setup>
import * as z from 'zod'
import { computed, provide } from 'vue'
import { PlusIcon, TrashIcon } from 'lucide-vue-next'
import { FieldArray, FieldContextKey, useField } from 'vee-validate'
import { beautifyObjectName, getBaseType } from './utils'
import AutoFormField from './AutoFormField.vue'
import AutoFormLabel from './AutoFormLabel.vue'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '../accordion'
import { Button } from '../button'
import { Separator } from '../separator'
import { FormItem, FormMessage } from '../form'

const props = defineProps({
  fieldName: { type: String, required: true },
  required: { type: Boolean, required: false },
  config: { type: null, required: false },
  schema: { type: null, required: false },
  disabled: { type: Boolean, required: false }
})

function isZodArray(item) {
  return item instanceof z.ZodArray
}

function isZodDefault(item) {
  return item instanceof z.ZodDefault
}

const itemShape = computed(() => {
  if (!props.schema) return

  const schema = isZodArray(props.schema)
    ? props.schema._def.type
    : isZodDefault(props.schema)
      ? props.schema._def.innerType._def.type
      : null

  return {
    type: getBaseType(schema),
    schema
  }
})

const fieldContext = useField(props.fieldName)
provide(FieldContextKey, fieldContext)
</script>

<template>
  <FieldArray v-slot="{ fields, remove, push }" as="section" :name="fieldName">
    <slot v-bind="props">
      <Accordion type="multiple" class="w-full" collapsible :disabled="disabled" as-child>
        <FormItem>
          <AccordionItem :value="fieldName" class="border-none">
            <AccordionTrigger>
              <AutoFormLabel class="text-base" :required="required">
                {{ schema?.description || beautifyObjectName(fieldName) }}
              </AutoFormLabel>
            </AccordionTrigger>

            <AccordionContent>
              <template v-for="(field, index) of fields" :key="field.key">
                <div class="mb-4 p-1">
                  <AutoFormField
                    :field-name="`${fieldName}[${index}]`"
                    :label="fieldName"
                    :shape="itemShape"
                    :config="config"
                  />

                  <div class="!my-4 flex justify-end">
                    <Button type="button" size="icon" variant="secondary" @click="remove(index)">
                      <TrashIcon :size="16" />
                    </Button>
                  </div>
                  <Separator v-if="!field.isLast" />
                </div>
              </template>

              <Button
                type="button"
                variant="secondary"
                class="mt-4 flex items-center"
                @click="push(null)"
              >
                <PlusIcon class="mr-2" :size="16" />
                Add
              </Button>
            </AccordionContent>

            <FormMessage />
          </AccordionItem>
        </FormItem>
      </Accordion>
    </slot>
  </FieldArray>
</template>
