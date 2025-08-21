<template>
  <div class="space-y-6">
    <!-- Master Toggle -->
    <FormField v-slot="{ componentField, handleChange }" name="config.prechat_form.enabled">
      <FormItem class="flex flex-row items-center justify-between box p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">{{ $t('admin.inbox.livechat.prechatForm.enabled') }}</FormLabel>
          <FormDescription>
            {{ $t('admin.inbox.livechat.prechatForm.enabled.description') }}
          </FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <!-- Form Configuration -->
    <div v-if="form.values.config?.prechat_form?.enabled" class="space-y-6">
      <!-- Form Title -->
      <FormField v-slot="{ componentField }" name="config.prechat_form.title">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.livechat.prechatForm.title') }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" placeholder="Tell us about yourself" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.livechat.prechatForm.title.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <!-- Fields Configuration -->
      <div class="space-y-4">
        <div class="flex justify-between items-center">
          <h4 class="font-medium text-foreground">
            {{ $t('admin.inbox.livechat.prechatForm.fields') }}
          </h4>
          <Button 
            type="button" 
            variant="outline" 
            size="sm" 
            @click="$emit('fetch-custom-attributes')"
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
            item-key="key"
            :animation="200"
            class="space-y-3"
          >
            <template #item="{ element: field, index }">
              <div class="border rounded-lg p-4 space-y-4">
            <!-- Field Header -->
            <div class="flex items-center justify-between">
              <div class="flex items-center space-x-3">
                <div class="cursor-move text-muted-foreground">
                  <GripVertical class="w-4 h-4" />
                </div>
                <div>
                  <div class="font-medium">{{ field.label }}</div>
                  <div class="text-sm text-muted-foreground">
                    {{ field.type }} {{ field.is_default ? '(Default)' : '(Custom)' }}
                  </div>
                </div>
              </div>
              <div class="flex items-center space-x-2">
                <FormField 
                  :name="`config.prechat_form.fields.${index}.enabled`" 
                  v-slot="{ componentField, handleChange }"
                >
                  <FormControl>
                    <Switch 
                      :checked="componentField.modelValue" 
                      @update:checked="handleChange"
                    />
                  </FormControl>
                </FormField>
                <Button
                  v-if="!field.is_default"
                  type="button"
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
                <FormField 
                  :name="`config.prechat_form.fields.${index}.label`" 
                  v-slot="{ componentField }"
                >
                  <FormItem>
                    <FormLabel class="text-sm font-medium">{{ $t('globals.terms.label') }}</FormLabel>
                    <FormControl>
                      <Input
                        v-bind="componentField"
                        placeholder="Field label"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <!-- Placeholder -->
                <FormField 
                  :name="`config.prechat_form.fields.${index}.placeholder`" 
                  v-slot="{ componentField }"
                >
                  <FormItem>
                    <FormLabel class="text-sm font-medium">{{ $t('globals.terms.placeholder') }}</FormLabel>
                    <FormControl>
                      <Input
                        v-bind="componentField"
                        placeholder="Field placeholder"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>

              <!-- Required -->
              <FormField 
                :name="`config.prechat_form.fields.${index}.required`" 
                v-slot="{ componentField, handleChange }"
              >
                <FormItem>
                  <div class="flex items-center space-x-2">
                    <FormControl>
                      <Checkbox
                        :checked="componentField.modelValue"
                        @update:checked="handleChange"
                      />
                    </FormControl>
                    <FormLabel class="text-sm">{{ $t('globals.terms.required') }}</FormLabel>
                  </div>
                </FormItem>
              </FormField>
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
          <h5 class="font-medium text-sm">{{ $t('admin.inbox.livechat.prechatForm.availableFields') }}</h5>
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
import { computed, onMounted } from 'vue'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { Switch } from '@shared-ui/components/ui/switch'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { Plus, X, GripVertical } from 'lucide-vue-next'
import Draggable from 'vuedraggable'

const props = defineProps({
  form: {
    type: Object,
    required: true
  },
  customAttributes: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['fetch-custom-attributes'])

const formFields = computed(() => {
  return props.form.values.config?.prechat_form?.fields || []
})

const availableCustomAttributes = computed(() => {
  const usedIds = formFields.value
    .filter(field => field.custom_attribute_id)
    .map(field => field.custom_attribute_id)
  
  return props.customAttributes.filter(attr => !usedIds.includes(attr.id))
})

const draggableFields = computed({
  get() {
    return formFields.value
  },
  set(newValue) {
    const fieldsWithUpdatedOrder = newValue.map((field, index) => ({
      ...field,
      order: index + 1
    }))
    props.form.setFieldValue('config.prechat_form.fields', fieldsWithUpdatedOrder)
  }
})

const removeField = (index) => {
  const fields = formFields.value.filter((_, i) => i !== index)
  props.form.setFieldValue('config.prechat_form.fields', fields)
}

const addCustomAttributeToForm = (attribute) => {
  const newField = {
    key: attribute.key,
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
  props.form.setFieldValue('config.prechat_form.fields', fields)
}

onMounted(() => {
  emit('fetch-custom-attributes')
})
</script>