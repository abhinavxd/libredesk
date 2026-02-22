<template>
  <div class="space-y-6">
    <!-- Master Toggle -->
    <div class="flex flex-row items-center justify-between box p-4">
      <div class="space-y-0.5">
        <label class="text-base font-medium">
          {{ $t('admin.inbox.livechat.prechatForm.enabled') }}
        </label>
        <p class="text-sm text-muted-foreground">
          {{ $t('admin.inbox.livechat.prechatForm.enabled.description') }}
        </p>
      </div>
      <Switch v-model:checked="prechatConfig.enabled" />
    </div>

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
        <p class="text-sm text-muted-foreground mt-1">
          {{ $t('admin.inbox.livechat.prechatForm.title.description') }}
        </p>
      </div>

      <!-- Fields Configuration -->
      <div class="space-y-4">
        <div class="flex justify-between items-center">
          <h4 class="font-medium text-foreground">
            {{ $t('admin.inbox.livechat.prechatForm.fields') }}
          </h4>
          <Button
            variant="outline"
            size="sm"
            @click="fetchCustomAttributes"
            :disabled="availableCustomAttributes.length === 0"
          >
            <Plus class="w-4 h-4 mr-2" />
            {{ $t('admin.inbox.livechat.prechatForm.addField') }}
          </Button>
        </div>

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

<script setup>
import { computed, onMounted, ref } from 'vue'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { Switch } from '@shared-ui/components/ui/switch'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { Plus, X, GripVertical } from 'lucide-vue-next'
import Draggable from 'vuedraggable'
import api from '@/api'

const prechatConfig = defineModel({
  default: () => ({
    enabled: false,
    title: '',
    fields: [
      {
        key: 'name',
        type: 'text',
        label: 'Full name',
        placeholder: 'Enter your name',
        required: true,
        enabled: true,
        order: 1,
        is_default: true
      },
      {
        key: 'email',
        type: 'email',
        label: 'Email address',
        placeholder: 'your@email.com',
        required: true,
        enabled: true,
        order: 2,
        is_default: true
      }
    ]
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

    // Clean up orphaned custom attribute fields
    const availableCustomAttrIds = customAttributes.value.map((attr) => attr.id)
    const cleanedFields = (prechatConfig.value.fields || []).filter((field) => {
      // Keep default fields
      if (field.is_default) return true

      // Keep custom fields that still exist
      if (field.custom_attribute_id && availableCustomAttrIds.includes(field.custom_attribute_id))
        return true

      // Remove orphaned custom fields
      return false
    })

    // Update fields if any were removed
    if (cleanedFields.length !== (prechatConfig.value.fields || []).length) {
      prechatConfig.value.fields = cleanedFields
    }
  } catch (error) {
    console.error('Error fetching custom attributes:', error)
    customAttributes.value = []
  }
}

onMounted(() => {
  fetchCustomAttributes()
})
</script>
