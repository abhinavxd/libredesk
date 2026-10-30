<template>
  <div class="space-y-3">
    <div
      v-for="attribute in attributes"
      :key="attribute.id"
      class="group/attribute grid min-w-0 grid-cols-[minmax(0,2fr)_minmax(0,3fr)] items-start gap-x-4"
    >
      <div class="sidebar-label min-w-0 py-0.5 [overflow-wrap:anywhere]">
        <Tooltip :disabled="!attribute.description">
          <TooltipTrigger as-child>
            <span>{{ attribute.name }}</span>
          </TooltipTrigger>
          <TooltipContent v-if="attribute.description">{{ attribute.description }}</TooltipContent>
        </Tooltip>
      </div>

      <Checkbox
        v-if="attribute.data_type === 'checkbox'"
        type="button"
        class="mt-1 justify-self-end"
        :disabled="loading"
        :aria-label="attribute.name"
        :checked="!!customAttributes?.[attribute.key]"
        @update:checked="(value) => saveAttribute(attribute.key, value)"
      />

      <Popover
        v-else
        :open="editingAttributeKey === attribute.key"
        @update:open="(open) => (open ? startEditing(attribute) : cancelEditing())"
      >
        <div class="flex min-w-0 items-baseline justify-end gap-1">
          <PopoverTrigger as-child>
            <Button
              type="button"
              variant="link"
              :size="getLinkUrl(attribute) ? 'icon' : 'default'"
              :disabled="loading"
              :class="[
                getLinkUrl(attribute)
                  ? 'h-6 w-6 shrink-0 text-muted-foreground [&_svg]:size-3 [&_svg]:translate-y-0.5 can-hover:opacity-0 can-hover:group-hover/attribute:opacity-100 can-hover:group-focus-within/attribute:opacity-100 data-[state=open]:opacity-100'
                  : 'sidebar-value h-auto min-h-6 min-w-6 max-w-full justify-end justify-self-end rounded-sm whitespace-normal px-0 py-0.5 text-right font-normal decoration-muted-foreground/40 decoration-dotted underline-offset-4 can-hover:hover:decoration-foreground',
                { 'text-muted-foreground': customAttributes?.[attribute.key] == null }
              ]"
              :aria-label="
                getLinkUrl(attribute)
                  ? `${t('globals.messages.edit')}: ${attribute.name}`
                  : `${t('globals.messages.edit')}: ${attribute.name}: ${customAttributes?.[attribute.key] ?? t('globals.messages.set')}`
              "
            >
              <Pencil v-if="getLinkUrl(attribute)" aria-hidden="true" />
              <span v-else class="line-clamp-2 min-w-0 [overflow-wrap:anywhere]">{{
                customAttributes?.[attribute.key] ?? t('globals.messages.set')
              }}</span>
            </Button>
          </PopoverTrigger>
          <Button
            v-if="getLinkUrl(attribute)"
            as="a"
            variant="link"
            :href="getLinkUrl(attribute)"
            target="_blank"
            rel="noopener noreferrer"
            class="sidebar-value h-auto min-h-6 min-w-0 max-w-full whitespace-normal px-0 py-0.5 text-right font-normal text-link"
          >
            <span class="line-clamp-2 min-w-0 [overflow-wrap:anywhere]">{{
              customAttributes[attribute.key]
            }}</span>
          </Button>
        </div>
        <PopoverContent align="end" class="w-72" :aria-label="attribute.name">
          <form class="space-y-3" novalidate @submit.stop.prevent="saveAttribute(attribute.key)">
            <div class="space-y-1.5">
              <Label :for="`${editorId}-${attribute.key}`">{{ attribute.name }}</Label>
              <p v-if="attribute.description" class="text-xs text-muted-foreground">
                {{ attribute.description }}
              </p>
            </div>

            <Select v-if="attribute.data_type === 'list'" v-model="editingValue">
              <SelectTrigger
                :id="`${editorId}-${attribute.key}`"
                :aria-invalid="!!errorMessage"
                :aria-describedby="errorMessage ? `${editorId}-${attribute.key}-error` : undefined"
              >
                <SelectValue :placeholder="t('placeholders.selectValue')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="option in attribute.values" :key="option" :value="option">
                  {{ option }}
                </SelectItem>
              </SelectContent>
            </Select>
            <Input
              v-else
              :id="`${editorId}-${attribute.key}`"
              v-model="editingValue"
              :type="INPUT_TYPES[attribute.data_type] || 'text'"
              :class="{ 'hide-number-spinners': attribute.data_type === 'number' }"
              :aria-invalid="!!errorMessage"
              :aria-describedby="errorMessage ? `${editorId}-${attribute.key}-error` : undefined"
            />

            <p
              v-if="errorMessage"
              :id="`${editorId}-${attribute.key}-error`"
              class="text-destructive text-xs"
              role="alert"
            >
              {{ errorMessage }}
            </p>
            <div class="flex items-center justify-between gap-2">
              <Button
                v-if="customAttributes?.[attribute.key] != null"
                type="button"
                variant="ghost"
                size="sm"
                class="text-muted-foreground"
                @click="deleteAttribute(attribute)"
              >
                {{ t('globals.terms.remove') }}
              </Button>
              <div class="ml-auto flex items-center gap-2">
                <Button type="button" variant="outline" size="sm" @click="cancelEditing">
                  {{ t('globals.messages.cancel') }}
                </Button>
                <Button type="submit" size="sm">{{ t('globals.messages.save') }}</Button>
              </div>
            </div>
          </form>
        </PopoverContent>
      </Popover>
    </div>
  </div>
</template>

<script setup>
import { ref, useId } from 'vue'
import { Pencil } from 'lucide-vue-next'
import * as z from 'zod'
import { Input } from '@shared-ui/components/ui/input'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Button } from '@shared-ui/components/ui/button'
import { Label } from '@shared-ui/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
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
const editorId = useId()
const INPUT_TYPES = { text: 'text', number: 'number', date: 'date', link: 'url' }
const errorMessage = ref('')
const editingAttributeKey = ref(null)
const editingValue = ref(null)

const getLinkUrl = (attribute) => {
  if (attribute.data_type !== 'link') return null
  try {
    const url = new URL(props.customAttributes?.[attribute.key])
    return url.protocol === 'http:' || url.protocol === 'https:' ? url.href : null
  } catch {
    return null
  }
}

const startEditing = (attribute) => {
  errorMessage.value = ''
  editingAttributeKey.value = attribute.key
  const currentValue = props.customAttributes?.[attribute.key]
  editingValue.value = attribute.data_type === 'checkbox' ? !!currentValue : (currentValue ?? null)
}

const cancelEditing = () => {
  editingAttributeKey.value = null
  editingValue.value = null
  errorMessage.value = ''
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
        .refine((val) => !isNaN(Date.parse(val)), t('validation.invalidValue'))
        .nullable()
    case 'link':
      return z.string().url(t('validation.invalidUrl')).nullable()
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

const saveAttribute = (key, value = editingValue.value) => {
  const attribute = props.attributes.find((attr) => attr.key === key)
  if (!attribute) return

  try {
    const schema = getValidationSchema(attribute)
    schema.parse(value)
  } catch (validationError) {
    if (validationError instanceof z.ZodError) {
      errorMessage.value = validationError.errors[0].message
      return
    }
    errorMessage.value = validationError
    return
  }

  const updatedAttributes = { ...(props.customAttributes || {}) }
  updatedAttributes[attribute.key] = value
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
