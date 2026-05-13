<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t('contact.createContact') }}</DialogTitle>
      </DialogHeader>

      <form @submit.prevent="onSubmit" class="space-y-6 pt-2">
        <!-- Avatar -->
        <div class="flex justify-center">
          <AvatarUpload
            @upload="onAvatarUpload"
            @remove="onAvatarRemove"
            :src="avatarPreview"
            :initials="initials"
            :label="t('globals.messages.upload')"
          />
        </div>

        <!-- First name / Last name -->
        <div class="flex flex-wrap gap-4">
          <div class="flex-1">
            <FormField v-slot="{ componentField }" name="first_name">
              <FormItem class="flex flex-col">
                <FormLabel>{{ t('globals.terms.firstName') }}</FormLabel>
                <FormControl><Input v-bind="componentField" type="text" /></FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>
          <div class="flex-1">
            <FormField v-slot="{ componentField }" name="last_name">
              <FormItem class="flex flex-col">
                <FormLabel>{{ t('globals.terms.lastName') }}</FormLabel>
                <FormControl><Input v-bind="componentField" type="text" /></FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>
        </div>

        <!-- Email -->
        <FormField v-slot="{ componentField }" name="email">
          <FormItem class="flex flex-col">
            <FormLabel>{{ t('globals.terms.email') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="email" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <!-- Phone -->
        <div class="flex flex-col">
          <div class="flex items-end">
            <FormField v-slot="{ componentField }" name="phone_number_country_code">
              <FormItem class="w-max">
                <FormLabel class="whitespace-nowrap">{{ t('globals.terms.phoneNumber') }}</FormLabel>
                <FormControl>
                  <ComboBox
                    v-bind="componentField"
                    :items="allCountries"
                    :placeholder="t('globals.terms.select')"
                    :buttonClass="'rounded-r-none border-r-0'"
                  >
                    <template #item="{ item }">
                      <div class="flex items-center gap-2">
                        <div class="w-7 h-7 flex items-center justify-center">
                          <span v-if="item.emoji">{{ item.emoji }}</span>
                        </div>
                        <span class="text-sm">{{ item.label }} ({{ item.calling_code }})</span>
                      </div>
                    </template>
                    <template #selected="{ selected }">
                      <div class="flex items-center gap-1">
                        <span v-if="selected" class="text-lg">{{ selected.emoji }}</span>
                        <span v-if="selected?.calling_code" class="text-xs text-muted-foreground">({{ selected.calling_code }})</span>
                      </div>
                    </template>
                  </ComboBox>
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
            <div class="flex-1">
              <FormField v-slot="{ componentField }" name="phone_number">
                <FormItem class="relative">
                  <FormControl>
                    <Input type="tel" v-bind="componentField" class="rounded-l-none" inputmode="numeric" />
                    <FormMessage class="absolute top-full mt-1 text-sm" />
                  </FormControl>
                </FormItem>
              </FormField>
            </div>
          </div>
        </div>

        <!-- Country -->
        <FormField v-slot="{ componentField }" name="country">
          <FormItem class="flex flex-col">
            <FormLabel>{{ t('globals.terms.country') }}</FormLabel>
            <FormControl>
              <ComboBox
                v-bind="componentField"
                :items="countryOptions"
                :placeholder="t('globals.terms.select')"
              >
                <template #item="{ item }">
                  <div class="flex items-center gap-2">
                    <span v-if="item.emoji">{{ item.emoji }}</span>
                    <span class="text-sm">{{ item.label }}</span>
                  </div>
                </template>
                <template #selected="{ selected }">
                  <div class="flex items-center gap-1">
                    <span v-if="selected" class="text-lg">{{ selected.emoji }}</span>
                    <span v-if="selected" class="text-sm">{{ selected.label }}</span>
                  </div>
                </template>
              </ComboBox>
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <div class="flex justify-end gap-2 pt-2">
          <Button type="button" variant="outline" @click="$emit('update:open', false)">
            {{ t('globals.messages.cancel') }}
          </Button>
          <Button type="submit" :isLoading="loading" :disabled="loading">
            {{ t('contact.createContact') }}
          </Button>
        </div>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import {
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import { AvatarUpload } from '@shared-ui/components/ui/avatar'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'
import countries from '../../constants/countries.js'
import api from '../../api'
import { createContactFormSchema } from './formSchema.js'
import { useEmitter } from '../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const props = defineProps({
  open: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:open', 'created'])

const { t } = useI18n()
const emitter = useEmitter()
const loading = ref(false)
const avatarFile = ref(null)
const avatarPreview = ref('')

const allCountries = countries.map((c) => ({
  label: c.name,
  value: c.iso_2,
  emoji: c.emoji,
  calling_code: c.calling_code
}))

const countryOptions = countries.map((c) => ({
  label: c.name,
  value: c.iso_2,
  emoji: c.emoji
}))

const form = useForm({
  validationSchema: toTypedSchema(createContactFormSchema(t))
})

const initials = computed(() => {
  const first = form.values.first_name?.[0] || ''
  const last = form.values.last_name?.[0] || ''
  return `${first}${last}`.toUpperCase()
})

function onAvatarUpload(file) {
  avatarFile.value = file
  avatarPreview.value = URL.createObjectURL(file)
}

function onAvatarRemove() {
  avatarFile.value = null
  avatarPreview.value = ''
}

function resetForm() {
  form.resetForm()
  avatarFile.value = null
  avatarPreview.value = ''
}

watch(() => props.open, (val) => {
  if (!val) resetForm()
})

const onSubmit = form.handleSubmit(async (values) => {
  loading.value = true
  try {
    const formData = new FormData()
    if (values.first_name) formData.append('first_name', values.first_name)
    if (values.last_name) formData.append('last_name', values.last_name)
    if (values.email) formData.append('email', values.email)
    if (values.phone_number) formData.append('phone_number', values.phone_number)
    if (values.phone_number_country_code) formData.append('phone_number_country_code', values.phone_number_country_code)
    if (values.country) formData.append('country', values.country)
    if (avatarFile.value) formData.append('files', avatarFile.value)

    const { data } = await api.createContact(formData)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('contact.createdSuccessfully')
    })
    emit('update:open', false)
    emit('created', data.data)
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    loading.value = false
  }
})
</script>
