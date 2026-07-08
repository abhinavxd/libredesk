<template>
  <div class="space-y-6">
    <!-- Master Toggle -->
    <SwitchField
      :title="$t('admin.inbox.livechat.prechatForm.enabled')"
      :description="$t('admin.inbox.livechat.prechatForm.enabled.description')"
      :checked="prechatConfig.enabled"
      @update:checked="prechatConfig.enabled = $event"
    />

    <!-- Form Configuration -->
    <div v-if="prechatConfig.enabled" class="space-y-6">
      <!-- Form Title -->
      <div>
        <label class="text-sm font-medium">
          {{ $t('admin.inbox.livechat.prechatForm.title') }}
        </label>
        <Input
          type="text"
          v-model="prechatConfig.title"
          :placeholder="$t('placeholders.tellUsAboutYourself')"
          class="mt-1"
        />
      </div>

      <!-- Fields Configuration -->
      <div class="space-y-4">
        <h4 class="font-medium text-foreground">
          {{ $t('admin.inbox.livechat.prechatForm.fields') }}
        </h4>

        <!-- Field List -->
        <div class="space-y-3">
          <Draggable
            v-model="draggableFields"
            :item-key="(field) => field.key || `field_${field.custom_attribute_id || 'unknown'}`"
            :animation="200"
            class="space-y-3"
          >
            <template #item="{ element: field, index }">
              <div :key="field.key || `field-${index}`" class="border rounded-lg p-4 space-y-4">
                <!-- Field Header -->
                <div class="flex items-center justify-between">
                  <div class="flex items-center space-x-3">
                    <div class="cursor-move text-muted-foreground">
                      <GripVertical class="w-4 h-4" />
                    </div>
                    <div>
                      <div class="font-medium">{{ field.label }}</div>
                      <div class="text-sm text-muted-foreground">
                        {{ field.type }} ({{ field.is_default ? $t('globals.terms.default') : $t('globals.terms.custom') }})
                      </div>
                    </div>
                  </div>
                  <div class="flex items-center space-x-2">
                    <Switch v-model:checked="field.enabled" />
                    <Button
                      v-if="!field.is_default"
                      variant="ghost"
                      size="sm"
                      @click="removeField(index)"
                    >
                      <X class="w-4 h-4" />
                    </Button>
                  </div>
                </div>

                <!-- Field Configuration -->
                <div v-if="field.enabled" class="space-y-4">
                  <div class="grid grid-cols-2 gap-4">
                    <!-- Label -->
                    <div>
                      <label class="text-sm font-medium">{{ $t('globals.terms.label') }}</label>
                      <Input v-model="field.label" :placeholder="$t('placeholders.fieldLabel')" class="mt-1" />
                    </div>

                    <!-- Placeholder -->
                    <div>
                      <label class="text-sm font-medium">
                        {{ $t('globals.terms.placeholder') }}
                      </label>
                      <Input
                        v-model="field.placeholder"
                        :placeholder="$t('placeholders.fieldPlaceholder')"
                        class="mt-1"
                      />
                    </div>
                  </div>

                  <!-- Required -->
                  <div class="flex items-center space-x-2">
                    <Checkbox v-model:checked="field.required" />
                    <label class="text-sm">{{ $t('globals.terms.required') }}</label>
                  </div>
                </div>
              </div>
            </template>
          </Draggable>

          <!-- Empty State -->
          <div v-if="formFields.length === 0" class="text-center py-8 text-muted-foreground">
            {{ $t('admin.inbox.livechat.prechatForm.noFields') }}
          </div>
        </div>

        <!-- Custom Attributes Selection -->
        <div v-if="availableCustomAttributes.length > 0" class="space-y-3">
          <h5 class="font-medium text-sm">
            {{ $t('admin.inbox.livechat.prechatForm.availableFields') }}
          </h5>
          <div class="grid grid-cols-2 gap-2 max-h-48 overflow-y-auto">
            <div
              v-for="attr in availableCustomAttributes"
              :key="attr.id"
              class="flex items-center space-x-2 p-2 border rounded cursor-pointer hover:bg-accent"
              @click="addCustomAttributeToForm(attr)"
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
  </div>
</template>

<script>
export const getDefaultPrechatFields = () => [
  { key: 'name', type: 'text', label: 'Full name', placeholder: 'Enter your name', required: true, enabled: true, order: 1, is_default: true },
  { key: 'email', type: 'email', label: 'Email address', placeholder: 'your@email.com', required: true, enabled: true, order: 2, is_default: true },
  { key: 'phone', type: 'phone', label: 'Phone number', placeholder: '', required: false, enabled: false, order: 3, is_default: true }
]
</script>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { Switch } from '@shared-ui/components/ui/switch'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { Plus, X, GripVertical } from 'lucide-vue-next'
import Draggable from 'vuedraggable'
import api from '@/api'

const prechatConfig = defineModel({
  default: () => ({
    enabled: false,
    title: '',
    fields: getDefaultPrechatFields()
  })
})

const customAttributes = ref([])

const formFields = computed(() => {
  return prechatConfig.value.fields || []
})

const availableCustomAttributes = computed(() => {
  const usedIds = formFields.value
    .filter((field) => field.custom_attribute_id)
    .map((field) => field.custom_attribute_id)

  return customAttributes.value.filter((attr) => !usedIds.includes(attr.id))
})

const draggableFields = computed({
  get() {
    return prechatConfig.value.fields || []
  },
  set(newValue) {
    const fieldsWithUpdatedOrder = newValue.map((field, index) => ({
      ...field,
      order: index + 1
    }))
    prechatConfig.value.fields = fieldsWithUpdatedOrder
  }
})

const removeField = (index) => {
  const fields = formFields.value.filter((_, i) => i !== index)
  prechatConfig.value.fields = fields
}

const addCustomAttributeToForm = (attribute) => {
  const newField = {
    key: attribute.key || `custom_attr_${attribute.id || Date.now()}`,
    type: attribute.data_type,
    label: attribute.name,
    placeholder: '',
    required: false,
    enabled: false,
    order: formFields.value.length + 1,
    is_default: false,
    custom_attribute_id: attribute.id
  }

  const fields = [...formFields.value, newField]
  prechatConfig.value.fields = fields
}

const fetchCustomAttributes = async () => {
  try {
    // Fetch both contact and conversation custom attributes
    const [contactAttrs, conversationAttrs] = await Promise.all([
      api.getCustomAttributes('contact'),
      api.getCustomAttributes('conversation')
    ])

    customAttributes.value = [
      ...(contactAttrs.data?.data || []),
      ...(conversationAttrs.data?.data || [])
    ]

    // Build lookup map for custom attributes
    const customAttrMap = new Map(customAttributes.value.map((attr) => [attr.id, attr]))

    // Clean up orphaned fields and sync labels/types from current custom attribute definitions
    const cleanedFields = (prechatConfig.value.fields || []).filter((field) => {
      if (field.is_default) return true
      if (field.custom_attribute_id && customAttrMap.has(field.custom_attribute_id)) return true
      return false
    }).map((field) => {
      if (!field.is_default && field.custom_attribute_id) {
        const attr = customAttrMap.get(field.custom_attribute_id)
        if (attr) {
          field.label = attr.name
          field.type = attr.data_type
        }
      }
      return field
    })

    prechatConfig.value.fields = cleanedFields
  } catch (error) {
    console.error('Error fetching custom attributes:', error)
    customAttributes.value = []
  }
}

onMounted(() => {
  fetchCustomAttributes()
})
</script>
