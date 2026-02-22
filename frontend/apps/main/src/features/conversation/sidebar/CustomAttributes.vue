<template>
  <div class="space-y-4">
    <div class="relative group/item" v-for="attribute in attributes" :key="attribute.id">
      <!-- Label -->
      <div class="font-medium flex items-center" v-if="attribute.data_type !== 'checkbox'">
        <div class="flex items-center gap-1">
          <p>
            {{ attribute.name }}
          </p>
          <Tooltip>
            <TooltipTrigger>
              <Info class="text-muted-foreground" size="14" />
            </TooltipTrigger>
            <TooltipContent>
              {{ attribute.description }}
            </TooltipContent>
          </Tooltip>
        </div>
      </div>

      <!-- Checkbox -->
      <div class="font-medium flex items-center gap-2" v-else>
        <Checkbox
          v-if="attribute.data_type === 'checkbox'"
          :disabled="loading"
          @update:checked="
            (value) => {
              editingValue = value
              saveAttribute(attribute.key)
            }
          "
          :checked="customAttributes?.[attribute.key]"
        />
        <p>
          {{ attribute.name }}
        </p>
        <Tooltip>
          <TooltipTrigger>
            <Info class="text-muted-foreground" size="14" />
          </TooltipTrigger>
          <TooltipContent>
            {{ attribute.description }}
          </TooltipContent>
        </Tooltip>
      </div>

      <Skeleton v-if="loading" class="w-32 h-4" />

      <!-- Value -->
      <template v-else-if="attribute.data_type !== 'checkbox'">
        <div
          v-if="!editingAttributeKey || editingAttributeKey !== attribute.key"
          class="flex items-center gap-2"
        >
          <span class="break-all" v-if="attribute.data_type !== 'checkbox'">
            {{ customAttributes?.[attribute.key] ?? '-' }}
          </span>
          <Pencil
            size="12"
            class="text-muted-foreground cursor-pointer flex-shrink-0 opacity-0 group-hover/item:opacity-100 transition-opacity duration-200"
            @click="startEditing(attribute)"
          />
          <Trash2
            v-if="customAttributes?.[attribute.key]"
            size="12"
            class="text-muted-foreground cursor-pointer flex-shrink-0 absolute right-0 top-1"
            @click="deleteAttribute(attribute)"
          />
        </div>
        <div v-else>
          <div class="flex items-center gap-2">
            <template v-if="attribute.data_type === 'text'">
              <Input
                v-model="editingValue"
                type="text"
                @keydown.enter="saveAttribute(attribute.key)"
              />
            </template>
            <template v-else-if="attribute.data_type === 'number'">
              <Input
                v-model="editingValue"
                type="number"
                @keydown.enter="saveAttribute(attribute.key)"
              />
            </template>
            <template v-else-if="attribute.data_type === 'checkbox'">
              <Checkbox v-model:checked="editingValue" />
            </template>
            <template v-else-if="attribute.data_type === 'date'">
              <Input v-model="editingValue" type="date" />
            </template>
            <template v-else-if="attribute.data_type === 'link'">
              <Input
                v-model="editingValue"
                type="url"
                @keydown.enter="saveAttribute(attribute.key)"
              />
            </template>
            <template v-else-if="attribute.data_type === 'list'">
              <Select v-model="editingValue">
                <SelectTrigger>
                  <SelectValue :placeholder="t('placeholders.selectValue')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="option in attribute.values" :key="option" :value="option">
                    {{ option }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </template>
            <Check
              size="20"
              class="text-muted-foreground cursor-pointer"
              @click="saveAttribute(attribute.key)"
            />
            <X size="20" class="text-muted-foreground cursor-pointer" @click="cancelEditing" />
          </div>
          <p v-if="errorMessage" class="text-red-500 text-xs mt-1">
            {{ errorMessage }}
          </p>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import * as z from 'zod'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { Input } from '@shared-ui/components/ui/input'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Pencil, Trash2, Check, X, Info } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  attributes: {
    type: Array,
    required: true
  },
  customAttributes: {
    type: Object,
    required: true
  },
  loading: {
    type: Boolean,
    default: false
  }
})
const emit = defineEmits(['update:setattributes'])
const { t } = useI18n()
const errorMessage = ref('')
const editingAttributeKey = ref(null)
const editingValue = ref(null)

const startEditing = (attribute) => {
  errorMessage.value = ''
  editingAttributeKey.value = attribute.key
  const currentValue = props.customAttributes?.[attribute.key]
  editingValue.value = attribute.data_type === 'checkbox' ? !!currentValue : (currentValue ?? null)
}

const cancelEditing = () => {
  editingAttributeKey.value = null
  editingValue.value = null
}

const getValidationSchema = (attribute) => {
  switch (attribute.data_type) {
    case 'text': {
      let schema = z.string().min(1, t('globals.messages.required'))
      // If regex is provided and valid, add it to the schema validation along with the hint
      if (attribute.regex) {
        try {
          const regex = new RegExp(attribute.regex)
          schema = schema.regex(regex, {
            message: attribute.regex_hint
          })
        } catch (err) {
          console.error('Error creating regex:', err)
        }
      }
      return schema.nullable()
    }
    case 'number':
      return z.preprocess(
        (val) => Number(val),
        z
          .number({
            invalid_type_error: t('validation.invalidValue')
          })
          .nullable()
      )
    case 'checkbox':
      return z.boolean().nullable()
    case 'date':
      return z
        .string()
        .refine(
          (val) => !isNaN(Date.parse(val)),
          t('validation.invalidValue')
        )
        .nullable()
    case 'link':
      return z
        .string()
        .url(
          t('validation.invalidUrl')
        )
        .nullable()
    case 'list':
      return z
        .string()
        .refine((val) => attribute.values.includes(val), {
          message: t('validation.invalidValue')
        })
        .nullable()
    default:
      return z.any()
  }
}

const saveAttribute = (key) => {
  const attribute = props.attributes.find((attr) => attr.key === key)
  if (!attribute) return

  try {
    const schema = getValidationSchema(attribute)
    schema.parse(editingValue.value)
  } catch (validationError) {
    if (validationError instanceof z.ZodError) {
      errorMessage.value = validationError.errors[0].message
      return
    }
    errorMessage.value = validationError
    return
  }

  const updatedAttributes = { ...(props.customAttributes || {}) }
  updatedAttributes[attribute.key] = editingValue.value
  emit('update:setattributes', updatedAttributes)
  cancelEditing()
}

const deleteAttribute = (attribute) => {
  const updatedAttributes = { ...(props.customAttributes || {}) }
  delete updatedAttributes[attribute.key]
  emit('update:setattributes', updatedAttributes)
  if (editingAttributeKey.value === attribute.key) cancelEditing()
}
</script>
