<template>
  <AdminSplitLayout>
    <template #content>
      <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
      </div>

      <p
        v-if="isEditing && template && !canEditWhatsAppTemplate(template)"
        class="mb-5 text-sm text-muted-foreground"
        role="status"
      >
        {{ $t('admin.whatsappTemplates.error.editUnavailable') }}
      </p>
      <form @submit="onSubmit" class="w-full">
        <fieldset :disabled="isFetching || isLoading || editUnavailable" class="space-y-6">
          <div :class="FIELD_GRID_CLASS">
            <FormField v-slot="{ componentField, handleChange, meta }" name="inbox_id">
              <FormItem>
                <FormLabel>{{ $t('globals.terms.inbox') }}</FormLabel>
                <FormControl>
                  <Select
                    :modelValue="componentField.modelValue"
                    @update:modelValue="(v) => handleChange(v, meta.validated)"
                    :disabled="isEditing"
                  >
                    <SelectTrigger>
                      <SelectValue :placeholder="$t('placeholders.selectInbox')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="inb in whatsappInboxes" :key="inb.id" :value="inb.id">
                        {{ inb.name }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField, handleChange, meta }" name="category">
              <FormItem>
                <FormLabel>{{ $t('globals.terms.category') }}</FormLabel>
                <FormControl>
                  <Select
                    :modelValue="componentField.modelValue"
                    @update:modelValue="(v) => handleChange(v, meta.validated)"
                    :disabled="template?.status === 'APPROVED'"
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="c in TEMPLATE_CATEGORIES" :key="c" :value="c">{{
                        c
                      }}</SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormDescription>{{
                  $t('admin.whatsappTemplates.category.description')
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>

          <div :class="FIELD_GRID_CLASS">
            <FormField v-slot="{ componentField, handleChange, meta }" name="name">
              <FormItem>
                <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
                <FormControl>
                  <Input
                    type="text"
                    placeholder="order_status"
                    :modelValue="componentField.modelValue"
                    @update:modelValue="(v) => handleChange(v, meta.validated)"
                    :disabled="isEditing"
                  />
                </FormControl>
                <FormDescription>{{
                  $t('globals.messages.lowercaseLettersNumbersUnderscoresOnly')
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField, handleChange, meta }" name="language">
              <FormItem>
                <FormLabel>{{ $t('globals.terms.language') }}</FormLabel>
                <FormControl>
                  <ComboBox
                    :modelValue="componentField.modelValue"
                    @update:modelValue="(v) => handleChange(v, meta.validated)"
                    :disabled="isEditing"
                    :items="WHATSAPP_TEMPLATE_LANGUAGES"
                    :placeholder="$t('globals.terms.search')"
                  />
                </FormControl>
                <FormDescription>{{
                  $t('admin.whatsappTemplates.language.description')
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>

          <div :class="SECTION_CLASS">
            <h3 class="font-semibold">{{ $t('globals.terms.header') }}</h3>

            <div :class="FIELD_GRID_CLASS">
              <FormField v-slot="{ componentField, handleChange, meta }" name="header_type">
                <FormItem>
                  <FormLabel>{{ $t('globals.terms.headerType') }}</FormLabel>
                  <FormControl>
                    <Select
                      :modelValue="componentField.modelValue"
                      @update:modelValue="(v) => handleChange(v, meta.validated)"
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem v-for="h in HEADER_TYPES" :key="h" :value="h">{{
                          h
                        }}</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <FormField
              v-if="form.values.header_type === 'TEXT'"
              v-slot="{ componentField, handleChange, meta }"
              name="header_content"
            >
              <FormItem>
                <FormLabel>{{ $t('globals.terms.headerText') }}</FormLabel>
                <FormControl>
                  <Input
                    type="text"
                    :modelValue="componentField.modelValue"
                    @update:modelValue="(v) => handleChange(v, meta.validated)"
                  />
                </FormControl>
                <FormDescription>{{
                  $t('admin.whatsappTemplates.headerText.description')
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>

          <div :class="SECTION_CLASS">
            <h3 class="font-semibold">{{ $t('globals.terms.body') }}</h3>

            <FormField v-slot="{ componentField, handleChange, meta }" name="body_content">
              <FormItem>
                <FormLabel>{{ $t('globals.terms.bodyText') }}</FormLabel>
                <FormControl>
                  <Textarea
                    rows="4"
                    :modelValue="componentField.modelValue"
                    @update:modelValue="(v) => handleChange(v, meta.validated)"
                  />
                </FormControl>
                <FormDescription>{{
                  $t('admin.whatsappTemplates.bodyText.description')
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField, handleChange, meta }" name="footer_content">
              <FormItem>
                <FormLabel>{{ $t('globals.terms.footer') }}</FormLabel>
                <FormControl>
                  <Input
                    type="text"
                    maxlength="60"
                    :modelValue="componentField.modelValue"
                    @update:modelValue="(v) => handleChange(v, meta.validated)"
                  />
                </FormControl>
                <FormDescription>{{
                  $t('globals.messages.maxLength', { max: 60 })
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>

          <div v-if="placeholders.length" :class="SECTION_CLASS">
            <h3 class="font-semibold">{{ $t('globals.terms.sampleValue', 2) }}</h3>
            <p class="text-xs text-muted-foreground">
              {{ $t('admin.whatsappTemplates.sampleValues.description') }}
            </p>
            <div v-for="key in placeholders" :key="key" class="grid grid-cols-3 gap-3 items-start">
              <label :for="`sample-${key}`" class="text-sm font-mono pt-2">{{
                placeholderLabel(key)
              }}</label>
              <div class="col-span-2">
                <Input
                  v-model="sampleValues[key]"
                  :id="`sample-${key}`"
                  :placeholder="$t('globals.terms.sampleValue')"
                />
                <p v-if="sampleErrors[key]" class="text-sm text-destructive mt-1">
                  {{ sampleErrors[key] }}
                </p>
              </div>
            </div>
          </div>

          <div :class="SECTION_CLASS">
            <div class="flex items-center justify-between">
              <h3 class="font-semibold">{{ $t('globals.terms.button', 2) }}</h3>
              <Button
                type="button"
                variant="outline"
                size="sm"
                :disabled="buttons.length >= 3"
                @click="addButton"
              >
                <Plus class="size-4" />
                {{ $t('globals.messages.addButton') }}
              </Button>
            </div>

            <div
              v-for="(btn, idx) in buttons"
              :key="idx"
              class="grid grid-cols-12 gap-2 items-start"
            >
              <Select v-model="btn.type">
                <SelectTrigger class="col-span-4" :aria-label="$t('globals.terms.type')">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="t in BUTTON_TYPES" :key="t" :value="t">{{ t }}</SelectItem>
                </SelectContent>
              </Select>
              <Input
                v-model="btn.text"
                :placeholder="$t('globals.terms.title')"
                :aria-label="$t('globals.terms.title')"
                class="col-span-3"
              />
              <Input
                v-if="btn.type === 'URL'"
                v-model="btn.url"
                :aria-label="$t('globals.terms.url', 1)"
                placeholder="https://example.com/"
                class="col-span-4"
              />
              <Input
                v-else-if="btn.type === 'PHONE_NUMBER'"
                v-model="btn.phone_number"
                :aria-label="$t('globals.terms.phoneNumber', 1)"
                placeholder="+1234567890"
                class="col-span-4"
              />
              <div v-else class="col-span-4"></div>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                class="col-span-1"
                :aria-label="$t('globals.messages.delete')"
                @click="removeButton(idx)"
              >
                <X class="size-4" />
              </Button>
              <p v-if="buttonErrors[idx]" class="col-span-12 text-sm text-destructive">
                {{ buttonErrors[idx] }}
              </p>
            </div>
          </div>
        </fieldset>
        <div class="flex gap-2 mt-6">
          <Button
            type="submit"
            :is-loading="isLoading"
            :disabled="isLoading || isFetching || editUnavailable"
          >
            {{ $t('admin.whatsappTemplates.submit') }}
          </Button>
          <Button type="button" variant="outline" @click="cancel">
            {{ $t('globals.messages.cancel') }}
          </Button>
        </div>
      </form>
    </template>

    <template #help>
      <div class="space-y-4">
        <div class="space-y-1">
          <p class="text-sm font-medium text-foreground">
            {{ isEditing ? $t('globals.messages.edit') : $t('globals.messages.newTemplate') }}
          </p>
          <a
            href="https://developers.facebook.com/docs/whatsapp/business-management-api/message-templates"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style text-sm"
          >
            {{ $t('globals.terms.learnMore') }}
          </a>
        </div>
      </div>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
const SECTION_CLASS = 'box p-4 space-y-4'
const FIELD_GRID_CLASS = 'grid grid-cols-2 gap-4'

import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { useI18n } from 'vue-i18n'
import { Plus, X } from 'lucide-vue-next'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Button } from '@shared-ui/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb/index.js'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import {
  createFormSchema,
  TEMPLATE_CATEGORIES,
  HEADER_TYPES,
  BUTTON_TYPES
} from '@/features/admin/whatsapp/whatsappTemplateSchema.js'
import { WHATSAPP_TEMPLATE_LANGUAGES } from '@/features/admin/whatsapp/whatsappLanguages.js'
import { useInboxStore } from '@main/stores/inbox'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'
import {
  canEditWhatsAppTemplate,
  extractPlaceholders,
  placeholderLabel
} from '@/features/conversation/whatsappTemplate.js'

const props = defineProps({ id: { type: String, default: null } })

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const inboxStore = useInboxStore()
const emitter = useEmitter()

const isEditing = computed(() => Boolean(props.id))
const template = ref(null)
const isFetching = ref(isEditing.value)
const editUnavailable = computed(
  () => isEditing.value && (!template.value || !canEditWhatsAppTemplate(template.value))
)
const isLoading = ref(false)
const sampleValues = reactive({})
const buttons = ref([])

const breadcrumbLinks = computed(() => [
  { path: 'whatsapp-templates', label: t('admin.whatsappTemplates.title') },
  {
    path: '',
    label: isEditing.value ? t('globals.messages.edit') : t('globals.messages.newTemplate')
  }
])

const whatsappInboxes = computed(() =>
  inboxStore.inboxes.filter((inb) => inb.channel === 'whatsapp')
)

const form = useForm({
  validationSchema: computed(() => toTypedSchema(createFormSchema(t))),
  initialValues: {
    inbox_id: Number(route.query.inbox_id) || null,
    name: '',
    language: 'en_US',
    category: 'UTILITY',
    header_type: 'NONE',
    header_content: '',
    body_content: '',
    footer_content: '',
    sample_values: {},
    buttons: []
  }
})

const sampleErrors = reactive({})

const placeholders = computed(() => {
  const sources = [form.values.body_content || '']
  if (form.values.header_type === 'TEXT') {
    sources.push(form.values.header_content || '')
  }
  for (const btn of buttons.value) {
    if (btn.type === 'URL' && btn.url) sources.push(btn.url)
  }
  return extractPlaceholders(sources)
})

watch(placeholders, (current) => {
  const valid = new Set(current)
  for (const k of Object.keys(sampleValues)) {
    if (!valid.has(k)) delete sampleValues[k]
  }
})

const addButton = () => {
  if (buttons.value.length >= 3) return
  buttons.value.push({ type: 'URL', text: '', url: '', phone_number: '' })
}

const removeButton = (idx) => buttons.value.splice(idx, 1)

const cancel = () => {
  router.push({ name: 'whatsapp-templates', query: { inbox_id: form.values.inbox_id } })
}

const buttonErrors = reactive({})

const isButtonEmpty = (b) => !b.text?.trim() && !b.url?.trim() && !b.phone_number?.trim()

const validateButtons = () => {
  for (const k of Object.keys(buttonErrors)) delete buttonErrors[k]
  let ok = true
  buttons.value.forEach((b, idx) => {
    if (isButtonEmpty(b)) return
    let missing = ''
    if (!b.text?.trim()) missing = t('globals.terms.title')
    else if (b.type === 'URL' && !b.url?.trim()) missing = t('globals.terms.url', 1)
    else if (b.type === 'PHONE_NUMBER' && !b.phone_number?.trim())
      missing = t('globals.terms.phoneNumber', 1)
    if (missing) {
      buttonErrors[idx] = t('globals.messages.required', { name: missing })
      ok = false
    }
  })
  return ok
}

const validateSampleValues = () => {
  for (const k of Object.keys(sampleErrors)) delete sampleErrors[k]
  let ok = true
  for (const key of placeholders.value) {
    if (!(sampleValues[key] || '').trim()) {
      sampleErrors[key] = t('globals.messages.required', { name: placeholderLabel(key) })
      ok = false
    }
  }
  return ok
}

watch(sampleValues, () => {
  for (const key of placeholders.value) {
    if ((sampleValues[key] || '').trim() && sampleErrors[key]) delete sampleErrors[key]
  }
})

const onSubmit = form.handleSubmit(async (values) => {
  if (isLoading.value || isFetching.value || editUnavailable.value) return
  if (!validateSampleValues() || !validateButtons()) return
  try {
    isLoading.value = true
    const payload = {
      ...values,
      header_type: values.header_type === 'NONE' ? null : values.header_type,
      header_content: values.header_type === 'TEXT' ? values.header_content : null,
      footer_content: values.footer_content || null,
      sample_values: { ...sampleValues },
      buttons: buttons.value
        .filter((b) => !isButtonEmpty(b))
        .map((b) => ({
          type: b.type,
          text: b.text,
          url: b.type === 'URL' ? b.url : undefined,
          phone_number: b.type === 'PHONE_NUMBER' ? b.phone_number : undefined
        }))
    }
    if (isEditing.value) {
      await api.updateWhatsAppTemplate(props.id, payload)
    } else {
      await api.createWhatsAppTemplate(payload)
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('admin.whatsappTemplates.submitted')
    })
    router.push({ name: 'whatsapp-templates', query: { inbox_id: values.inbox_id } })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})

onMounted(async () => {
  try {
    await inboxStore.fetchInboxes()
    if (!isEditing.value) return
    const response = await api.getWhatsAppTemplate(props.id)
    template.value = response.data.data
    const saved = template.value
    form.setValues(
      {
        inbox_id: saved.inbox_id,
        name: saved.name,
        language: saved.language,
        category: saved.category,
        header_type: saved.header_type || 'NONE',
        header_content: saved.header_content || '',
        body_content: saved.body_content,
        footer_content: saved.footer_content || ''
      },
      false
    )
    buttons.value = (saved.buttons || []).map((button) => ({ ...button }))
    await nextTick()
    Object.assign(sampleValues, saved.sample_values || {})
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isFetching.value = false
  }
})
</script>
