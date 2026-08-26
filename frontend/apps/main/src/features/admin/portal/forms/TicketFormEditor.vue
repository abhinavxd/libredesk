<template>
  <div class="space-y-6">
    <div>
      <label for="pt-form-name" class="text-sm font-medium">{{ $t('globals.terms.name') }}</label>
      <Input id="pt-form-name" type="text" v-model="ticketForm.name" class="mt-1" maxlength="140" />
    </div>

    <SwitchField
      :title="$t('admin.portalForm.askSubject')"
      :description="$t('admin.portalForm.askSubject.description')"
      :checked="ticketForm.ask_subject"
      @update:checked="ticketForm.ask_subject = $event"
    />

    <div class="space-y-4">
      <h4 class="font-medium text-foreground">{{ $t('admin.portalForm.fields') }}</h4>

      <Draggable v-model="draggableFields" item-key="key" :animation="200" class="space-y-3">
        <template #item="{ element: field, index }">
          <div class="border rounded-lg p-4 space-y-4">
            <div class="flex items-center justify-between">
              <div class="flex items-center space-x-3">
                <div class="cursor-move text-muted-foreground">
                  <GripVertical class="w-4 h-4" />
                </div>
                <div>
                  <div class="font-medium">{{ field.label || $t('globals.terms.label') }}</div>
                  <div class="text-sm text-muted-foreground">
                    {{ field.type }} ·
                    {{
                      field.target === 'attribute'
                        ? $t('admin.portalForm.savesToAttribute', { key: field.attribute_key })
                        : $t('admin.portalForm.addedToMessage')
                    }}
                  </div>
                </div>
              </div>
              <Button type="button" variant="ghost" size="sm" @click="removeField(index)">
                <X class="w-4 h-4" />
              </Button>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="text-sm font-medium">{{ $t('globals.terms.label') }}</label>
                <Input v-model="field.label" class="mt-1" maxlength="140" />
              </div>
              <div v-if="field.type !== 'checkbox'">
                <label class="text-sm font-medium">{{ $t('globals.terms.placeholder') }}</label>
                <Input v-model="field.placeholder" class="mt-1" />
              </div>
              <div v-if="field.target === 'message'">
                <label class="text-sm font-medium">{{ $t('globals.terms.type') }}</label>
                <Select :model-value="field.type" @update:model-value="setFieldType(field, $event)">
                  <SelectTrigger class="mt-1">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem v-for="type in FIELD_TYPES" :key="type" :value="type">
                        {{ type }}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>
              <div v-if="field.type === 'select'" class="col-span-2">
                <label class="text-sm font-medium">{{ $t('globals.messages.options') }}</label>
                <Textarea
                  :model-value="(field.options || []).join('\n')"
                  @update:model-value="field.options = String($event).split('\n').map((o) => o.trim()).filter(Boolean)"
                  rows="4"
                  class="mt-1"
                />
                <p class="text-xs text-muted-foreground mt-1">
                  {{ $t('admin.portalForm.optionsHint') }}
                </p>
              </div>
            </div>

            <div v-if="field.type !== 'checkbox'" class="flex items-center space-x-2">
              <Checkbox v-model:checked="field.required" />
              <label class="text-sm">{{ $t('globals.terms.required') }}</label>
            </div>
          </div>
        </template>
      </Draggable>

      <div v-if="fields.length === 0" class="text-center py-8 text-muted-foreground">
        {{ $t('admin.portalForm.noFields') }}
      </div>

      <Button type="button" variant="outline" size="sm" @click="addMessageField">
        <Plus class="w-4 h-4 mr-2" />
        {{ $t('admin.portalForm.addField') }}
      </Button>

      <div v-if="availableAttributes.length > 0" class="space-y-3">
        <h5 class="font-medium text-sm">{{ $t('admin.portalForm.availableAttributes') }}</h5>
        <div class="grid grid-cols-2 gap-2 max-h-48 overflow-y-auto">
          <div
            v-for="attr in availableAttributes"
            :key="attr.id"
            class="flex items-center space-x-2 p-2 border rounded-md cursor-pointer hover:bg-accent"
            @click="addAttributeField(attr)"
          >
            <div class="flex-1">
              <div class="font-medium text-sm">{{ attr.name }}</div>
              <div class="text-xs text-muted-foreground">{{ attr.data_type }}</div>
            </div>
            <Plus class="w-4 h-4 text-muted-foreground" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export const FIELD_TYPES = ['text', 'textarea', 'select', 'checkbox', 'number', 'date', 'email', 'link']

const ATTRIBUTE_FIELD_TYPES = {
  text: 'text',
  number: 'number',
  checkbox: 'checkbox',
  date: 'date',
  link: 'link',
  list: 'select'
}

</script>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Button } from '@shared-ui/components/ui/button'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Plus, X, GripVertical } from 'lucide-vue-next'
import Draggable from 'vuedraggable'
import api from '@/api'

const ticketForm = defineModel({
  default: () => ({ id: 0, name: '', ask_subject: true, fields: [] })
})

const attributes = ref([])

const fields = computed(() => ticketForm.value.fields || [])

const availableAttributes = computed(() => {
  const used = fields.value.filter((f) => f.target === 'attribute').map((f) => f.attribute_key)
  return attributes.value.filter((attr) => !used.includes(attr.key))
})

const draggableFields = computed({
  get() {
    return ticketForm.value.fields || []
  },
  set(value) {
    ticketForm.value.fields = value
  }
})

// Skips keys already in use: a removed field must not free a key the next add would collide with.
const nextFieldKey = () => {
  const used = new Set(fields.value.map((f) => f.key))
  let n = fields.value.length + 1
  while (used.has(`field_${n}`)) n++
  return `field_${n}`
}

const setFieldType = (field, type) => {
  field.type = type
  if (type === 'checkbox') field.required = false
  if (type === 'select' && !field.options) field.options = []
}

const removeField = (index) => {
  ticketForm.value.fields = fields.value.filter((_, i) => i !== index)
}

const addMessageField = () => {
  ticketForm.value.fields = [
    ...fields.value,
    {
      key: nextFieldKey(),
      label: '',
      type: 'text',
      required: false,
      placeholder: '',
      options: [],
      target: 'message',
      attribute_key: ''
    }
  ]
}

const addAttributeField = (attr) => {
  ticketForm.value.fields = [
    ...fields.value,
    {
      key: attr.key,
      label: attr.name,
      type: ATTRIBUTE_FIELD_TYPES[attr.data_type] || 'text',
      required: false,
      placeholder: '',
      options: attr.data_type === 'list' ? [...(attr.values || [])] : [],
      target: 'attribute',
      attribute_key: attr.key
    }
  ]
}

onMounted(async () => {
  try {
    const resp = await api.getCustomAttributes('conversation')
    attributes.value = resp.data.data || []
  } catch {
    attributes.value = []
  }
})
</script>
