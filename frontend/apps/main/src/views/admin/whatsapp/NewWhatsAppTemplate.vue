<template>
  <AdminSplitLayout>
    <template #content>
      <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
      </div>

      <form @submit="onSubmit" class="space-y-6 w-full">
        <FormField v-slot="{ componentField }" name="inbox_id">
          <FormItem>
            <FormLabel>{{ $t('globals.terms.inbox') }}</FormLabel>
            <FormControl>
              <Select v-bind="componentField">
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

        <div class="grid grid-cols-2 gap-4">
          <FormField v-slot="{ componentField }" name="name">
            <FormItem>
              <FormLabel>{{ $t('admin.whatsappTemplates.name') }}</FormLabel>
              <FormControl>
                <Input type="text" placeholder="order_status" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{
                $t('admin.whatsappTemplates.name.description')
              }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="language">
            <FormItem>
              <FormLabel>{{ $t('globals.terms.language') }}</FormLabel>
              <FormControl>
                <Input type="text" placeholder="en_US" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{
                $t('admin.whatsappTemplates.language.description')
              }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>

        <FormField v-slot="{ componentField }" name="category">
          <FormItem>
            <FormLabel>{{ $t('admin.whatsappTemplates.category') }}</FormLabel>
            <FormControl>
              <Select v-bind="componentField">
                <SelectTrigger>
                  <SelectValue :placeholder="$t('admin.whatsappTemplates.selectCategory')" />
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

        <div class="box p-4 space-y-4">
          <h3 class="font-semibold">{{ $t('admin.whatsappTemplates.header') }}</h3>

          <FormField v-slot="{ componentField }" name="header_type">
            <FormItem>
              <FormLabel>{{ $t('admin.whatsappTemplates.headerType') }}</FormLabel>
              <FormControl>
                <Select v-bind="componentField">
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="h in HEADER_TYPES" :key="h" :value="h">{{ h }}</SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField
            v-if="form.values.header_type === 'TEXT'"
            v-slot="{ componentField }"
            name="header_content"
          >
            <FormItem>
              <FormLabel>{{ $t('admin.whatsappTemplates.headerText') }}</FormLabel>
              <FormControl>
                <Input type="text" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{
                $t('admin.whatsappTemplates.headerText.description')
              }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>

        <div class="box p-4 space-y-4">
          <h3 class="font-semibold">{{ $t('admin.whatsappTemplates.body') }}</h3>

          <FormField v-slot="{ componentField }" name="body_content">
            <FormItem>
              <FormLabel>{{ $t('admin.whatsappTemplates.bodyText') }}</FormLabel>
              <FormControl>
                <Textarea
                  :placeholder="$t('admin.whatsappTemplates.bodyText.placeholder')"
                  rows="4"
                  v-bind="componentField"
                />
              </FormControl>
              <FormDescription>{{
                $t('admin.whatsappTemplates.bodyText.description')
              }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="footer_content">
            <FormItem>
              <FormLabel>{{ $t('admin.whatsappTemplates.footer') }}</FormLabel>
              <FormControl>
                <Input type="text" maxlength="60" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{
                $t('admin.whatsappTemplates.footer.description')
              }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>

        <div v-if="placeholders.length" class="box p-4 space-y-4">
          <h3 class="font-semibold">{{ $t('admin.whatsappTemplates.sampleValues') }}</h3>
          <p class="text-xs text-muted-foreground">
            {{ $t('admin.whatsappTemplates.sampleValues.description') }}
          </p>
          <div v-for="key in placeholders" :key="key" class="grid grid-cols-3 gap-3 items-start">
            <label class="text-sm font-mono pt-2">{{ placeholderLabel(key) }}</label>
            <div class="col-span-2">
              <Input
                v-model="sampleValues[key]"
                :placeholder="$t('admin.whatsappTemplates.sampleValues.placeholder')"
              />
              <p v-if="sampleErrors[key]" class="text-sm text-destructive mt-1">
                {{ sampleErrors[key] }}
              </p>
            </div>
          </div>
        </div>

        <div class="box p-4 space-y-4">
          <div class="flex items-center justify-between">
            <h3 class="font-semibold">{{ $t('globals.terms.buttons') }}</h3>
            <Button
              type="button"
              variant="outline"
              size="sm"
              :disabled="buttons.length >= 3"
              @click="addButton"
            >
              <Plus class="size-4" />
              {{ $t('admin.whatsappTemplates.addButton') }}
            </Button>
          </div>

          <div v-for="(btn, idx) in buttons" :key="idx" class="grid grid-cols-12 gap-2 items-start">
            <Select v-model="btn.type">
              <SelectTrigger class="col-span-4">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="t in BUTTON_TYPES" :key="t" :value="t">{{ t }}</SelectItem>
              </SelectContent>
            </Select>
            <Input v-model="btn.text" :placeholder="$t('globals.terms.title')" class="col-span-3" />
            <Input
              v-if="btn.type === 'URL'"
              v-model="btn.url"
              placeholder="https://example.com/"
              class="col-span-4"
            />
            <Input
              v-else-if="btn.type === 'PHONE_NUMBER'"
              v-model="btn.phone_number"
              placeholder="+1234567890"
              class="col-span-4"
            />
            <div v-else class="col-span-4"></div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              class="col-span-1"
              @click="removeButton(idx)"
            >
              <X class="size-4" />
            </Button>
          </div>
        </div>

        <div class="flex gap-2">
          <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
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
            {{ $t('admin.whatsappTemplates.newTemplate') }}
          </p>
          <p class="text-sm text-muted-foreground">
            {{ $t('admin.whatsappTemplates.help.create') }}
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
import { computed, onMounted, reactive, ref, watch } from 'vue'
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
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import {
  createFormSchema,
  TEMPLATE_CATEGORIES,
  HEADER_TYPES,
  BUTTON_TYPES
} from '@/features/admin/whatsapp/whatsappTemplateSchema.js'
import { useInboxStore } from '@main/stores/inbox'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'
import { extractPlaceholders, placeholderLabel } from '@/features/conversation/whatsappTemplate.js'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const inboxStore = useInboxStore()
const emitter = useEmitter()

const isLoading = ref(false)
const sampleValues = reactive({})
const buttons = ref([])

const breadcrumbLinks = [
  { path: 'whatsapp-templates', label: t('admin.whatsappTemplates.title') },
  { path: '', label: t('admin.whatsappTemplates.newTemplate') }
]

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
  router.push({ name: 'whatsapp-templates' })
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
  if (!validateSampleValues()) return
  try {
    isLoading.value = true
    const payload = {
      ...values,
      header_type: values.header_type === 'NONE' ? null : values.header_type,
      header_content: values.header_type === 'TEXT' ? values.header_content : null,
      footer_content: values.footer_content || null,
      sample_values: { ...sampleValues },
      buttons: buttons.value
        .filter((b) => b.text && (b.type === 'QUICK_REPLY' || b.url || b.phone_number))
        .map((b) => ({
          type: b.type,
          text: b.text,
          url: b.type === 'URL' ? b.url : undefined,
          phone_number: b.type === 'PHONE_NUMBER' ? b.phone_number : undefined
        }))
    }
    await api.createWhatsAppTemplate(payload)
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

onMounted(() => {
  inboxStore.fetchInboxes()
})
</script>
