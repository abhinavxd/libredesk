<template>
  <div class="space-y-2">
    <Label class="block">{{ label }}</Label>
    <div v-for="(field, index) in fields" :key="field.key" class="flex items-start gap-2">
      <slot name="leading" :index="index">
        <FormField v-slot="{ componentField }" :name="`${name}[${index}].label`">
          <FormItem class="flex-1">
            <FormControl>
              <Input type="text" :placeholder="i18n.ngettext('Label', 'Labels', 1)" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </slot>
      <FormField v-slot="{ componentField }" :name="`${name}[${index}].url`">
        <FormItem class="flex-1">
          <FormControl>
            <Input type="text" :placeholder="i18n.ngettext('URL', 'URLs', 1)" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        :aria-label="i18n._('Remove')"
        @click="remove(index)"
      >
        <X class="w-4 h-4" />
      </Button>
    </div>
    <Button type="button" variant="outline" size="sm" @click="push({ ...newItem })">
      {{ i18n._('Add') }}
    </Button>
  </div>
</template>

<script setup>
import { useI18n } from '@shared-ui/i18n'
import { useFieldArray } from 'vee-validate'
import { X } from 'lucide-vue-next'
import { Label } from '@shared-ui/components/ui/label'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import {
  FormControl,
  FormField,
  FormItem,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'

const props = defineProps({
  name: { type: String, required: true },
  label: { type: String, required: true },
  newItem: { type: Object, default: () => ({ label: '', url: '' }) }
})

const i18n = useI18n()
const { fields, push, remove } = useFieldArray(props.name)
</script>
