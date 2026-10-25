<script setup>
const EMAIL_CLASS = 'text-sm text-muted-foreground break-all'
const METADATA_CLASS = 'text-xs text-muted-foreground break-all'
const RESULT_LABEL_CLASS = 'mb-1 text-xs text-muted-foreground'
const ERROR_CLASS = 'text-sm text-destructive'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeftRightIcon, AlertTriangleIcon, UserRoundIcon } from 'lucide-vue-next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter
} from '@shared-ui/components/ui/dialog'
import { Button } from '@shared-ui/components/ui/button'
import { Label } from '@shared-ui/components/ui/label'
import { Badge } from '@shared-ui/components/ui/badge'
import { RadioGroup, RadioGroupItem } from '@shared-ui/components/ui/radio-group'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { handleHTTPError } from '@shared-ui/utils/http'
import countries from '@shared-ui/constants/countries'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'

const props = defineProps({ open: Boolean, contact: { type: Object, required: true } })
const emit = defineEmits(['update:open', 'merged'])
const { t } = useI18n()
const emitter = useEmitter()
const selectedId = ref('')
const preview = ref(null)
const choices = ref({})
const loading = ref(false)
const merging = ref(false)
const error = ref('')
const stale = ref(false)
const submitted = ref(false)
const missing = computed(
  () => preview.value?.fields.filter((field) => field.conflict && !choices.value[field.key]) || []
)
const names = computed(() =>
  preview.value ? [preview.value.keep, preview.value.remove].map(name) : []
)
const fieldOrder = { first_name: 0, last_name: 1, email: 2, phone: 3, country: 4, avatar_url: 5 }
const visibleFields = computed(
  () =>
    preview.value?.fields
      .filter((field) => !empty(field.keep, field.key) || !empty(field.remove, field.key))
      .sort(
        (a, b) => (fieldOrder[a.key] ?? 6) - (fieldOrder[b.key] ?? 6) || a.key.localeCompare(b.key)
      ) || []
)
const labels = {
  first_name: 'globals.terms.firstName',
  last_name: 'globals.terms.lastName',
  email: 'globals.terms.email',
  phone: 'globals.terms.phoneNumber',
  country: 'globals.terms.country',
  avatar_url: 'contact.merge.photo'
}

function name(contact) {
  return [contact.first_name, contact.last_name].filter(Boolean).join(' ')
}

function empty(value, key) {
  if (key === 'phone') return !value?.number
  return value == null || value === ''
}

function text(value, key) {
  if (empty(value, key)) return t('contact.merge.noValue')
  if (key === 'phone') {
    const country = countries.find((country) => country.iso_2 === value.country_code)
    return [country?.calling_code, value.number].filter(Boolean).join(' ')
  }
  if (key === 'country') return countries.find((country) => country.iso_2 === value)?.name || value
  return typeof value === 'object' ? JSON.stringify(value) : String(value)
}

function result(field) {
  if (field.conflict) return choices.value[field.key] ? field[choices.value[field.key]] : null
  return empty(field.keep, field.key) ? field.remove : field.keep
}

async function search(query) {
  error.value = ''
  if (query.trim().length < 3) return []
  let data
  try {
    ;({ data } = await api.searchMergeContacts({ query: query.trim() }))
  } catch (err) {
    error.value = handleHTTPError(err).message
    return []
  }
  return data.data
    .filter((contact) => contact.id !== props.contact.id)
    .map((contact) => ({
      value: String(contact.id),
      label: [name(contact), contact.email, contact.external_user_id].filter(Boolean).join(' · ')
    }))
}

async function loadPreview(keepId = props.contact.id, removeId = Number(selectedId.value)) {
  loading.value = true
  error.value = ''
  stale.value = false
  submitted.value = false
  choices.value = {}
  try {
    const { data } = await api.previewContactMerge(keepId, removeId)
    preview.value = data.data
  } catch (err) {
    preview.value = null
    error.value = handleHTTPError(err).message
  } finally {
    loading.value = false
  }
}

function close(open) {
  if (!merging.value) emit('update:open', open)
}

async function merge() {
  if (merging.value || loading.value || stale.value) return
  submitted.value = true
  if (missing.value.length) {
    document
      .getElementById(`merge-field-${preview.value.fields.indexOf(missing.value[0])}-keep`)
      ?.focus()
    return
  }
  merging.value = true
  error.value = ''
  try {
    const { data } = await api.mergeContacts(preview.value.keep.id, {
      remove_id: preview.value.remove.id,
      version: preview.value.version,
      choices: choices.value
    })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('contact.merge.success') })
    emit('merged', data.data.id)
    emit('update:open', false)
  } catch (err) {
    error.value = handleHTTPError(err).message
    stale.value = err.response?.status === 409
  } finally {
    merging.value = false
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="close">
    <DialogContent
      class="sm:max-w-4xl"
      :hide-close-button="merging"
      @escape-key-down="merging && $event.preventDefault()"
      @interact-outside="merging && $event.preventDefault()"
    >
      <DialogHeader>
        <DialogTitle>{{ t('contact.merge.title') }}</DialogTitle>
        <DialogDescription>{{ t('contact.merge.description') }}</DialogDescription>
      </DialogHeader>
      <div v-if="!preview" class="space-y-4">
        <div class="rounded-md border p-4">
          <p class="text-sm font-medium">{{ name(contact) }}</p>
          <p :class="EMAIL_CLASS">{{ contact.email }}</p>
          <p v-if="contact.external_user_id" :class="METADATA_CLASS">
            {{ contact.external_user_id }}
          </p>
        </div>
        <p id="merge-search-label" class="text-sm font-medium">
          {{ t('contact.merge.findContact') }}
        </p>
        <ComboBox
          aria-labelledby="merge-search-label"
          v-model="selectedId"
          :items="[]"
          :search="search"
          :disabled="loading"
          :default-label="t('contact.merge.search')"
          :placeholder="t('contact.merge.search')"
          @select="loadPreview()"
        />
      </div>
      <Spinner v-if="loading" />
      <form v-if="preview" class="min-w-0 space-y-5" @submit.prevent="merge">
        <div class="grid gap-3 sm:grid-cols-[1fr_auto_1fr] sm:items-center">
          <div
            v-for="(profile, index) in [preview.keep, preview.remove]"
            :key="profile.id"
            class="min-w-0 rounded-md border p-4"
            :class="index === 1 && 'sm:col-start-3 sm:row-start-1'"
          >
            <Badge variant="secondary">{{
              t(index === 0 ? 'contact.merge.keep' : 'contact.merge.remove')
            }}</Badge>
            <p class="mt-2 text-sm font-medium break-words">{{ name(profile) }}</p>
            <p :class="EMAIL_CLASS">{{ profile.email }}</p>
            <p :class="METADATA_CLASS">
              {{ t('contact.merge.profileID', { id: profile.id }) }}
            </p>
            <p v-if="profile.external_user_id" :class="METADATA_CLASS">
              {{ profile.external_user_id }}
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="icon"
            class="sm:col-start-2 sm:row-start-1"
            :disabled="loading || merging"
            :aria-label="t('contact.merge.swap')"
            @click="loadPreview(preview.remove.id, preview.keep.id)"
          >
            <ArrowLeftRightIcon class="size-4" aria-hidden="true" />
          </Button>
        </div>
        <div class="rounded-md border bg-muted/40 p-4 text-sm space-y-1">
          <p>
            {{
              t('contact.merge.history', {
                conversations: preview.keep.conversations + preview.remove.conversations,
                notes: preview.keep.notes + preview.remove.notes
              })
            }}
          </p>
          <p class="text-muted-foreground">{{ t('contact.merge.connections') }}</p>
          <p v-if="!preview.keep.enabled || !preview.remove.enabled">
            {{ t('contact.merge.staysBlocked') }}
          </p>
          <p
            v-if="preview.keep.external_user_id || preview.remove.external_user_id"
            class="break-all"
          >
            {{
              t('contact.merge.customerID', {
                id: preview.keep.external_user_id || preview.remove.external_user_id
              })
            }}
          </p>
          <div
            v-for="identity in [
              ...(preview.keep.channel_identities || []),
              ...(preview.remove.channel_identities || [])
            ]"
            :key="identity.channel + identity.identifier"
            class="break-all text-muted-foreground"
          >
            {{ identity.channel }}: {{ identity.identifier }}
          </div>
        </div>
        <p class="text-sm text-muted-foreground">{{ t('contact.merge.chooseDetails') }}</p>
        <div class="divide-y rounded-md border">
          <fieldset v-for="field in visibleFields" :key="field.key" class="min-w-0 p-4">
            <legend class="float-left mb-2 w-full text-sm font-medium">
              {{ labels[field.key] ? t(labels[field.key]) : field.key.slice(7) }}
            </legend>
            <RadioGroup
              v-if="field.conflict"
              v-model="choices[field.key]"
              :name="`merge-${field.key}`"
              :disabled="loading || merging"
              :aria-label="labels[field.key] ? t(labels[field.key]) : field.key.slice(7)"
              class="clear-both grid gap-3 sm:grid-cols-3"
            >
              <Label
                v-for="side in ['keep', 'remove']"
                :key="side"
                :for="`merge-field-${preview.fields.indexOf(field)}-${side}`"
                class="flex min-w-0 cursor-pointer items-start gap-2 rounded-md border p-3 font-normal"
                :class="choices[field.key] === side && 'border-primary bg-accent'"
              >
                <RadioGroupItem
                  :id="`merge-field-${preview.fields.indexOf(field)}-${side}`"
                  :value="side"
                  class="mt-0.5 shrink-0"
                />
                <span class="min-w-0 space-y-1">
                  <span class="block text-xs text-muted-foreground break-words">{{
                    side === 'keep' ? names[0] : names[1]
                  }}</span>
                  <Avatar v-if="field.key === 'avatar_url'" class="size-10"
                    ><AvatarImage
                      :src="field[side]"
                      :alt="side === 'keep' ? names[0] : names[1]" /><AvatarFallback
                      ><UserRoundIcon aria-hidden="true" /></AvatarFallback
                  ></Avatar>
                  <span v-else class="block text-sm [overflow-wrap:anywhere]">{{
                    text(field[side], field.key)
                  }}</span>
                </span>
              </Label>
              <div class="min-w-0 rounded-md bg-muted/40 p-3 text-sm">
                <p :class="RESULT_LABEL_CLASS">{{ t('contact.merge.result') }}</p>
                <p v-if="!choices[field.key]" class="text-muted-foreground">
                  {{ t('contact.merge.chooseValue') }}
                </p>
                <Avatar v-else-if="field.key === 'avatar_url'" class="size-10"
                  ><AvatarImage
                    :src="result(field)"
                    :alt="t('contact.merge.photo')" /><AvatarFallback
                    ><UserRoundIcon aria-hidden="true" /></AvatarFallback
                ></Avatar>
                <p v-else class="[overflow-wrap:anywhere]">{{ text(result(field), field.key) }}</p>
              </div>
            </RadioGroup>
            <div v-else class="clear-both grid gap-3 text-sm sm:grid-cols-3">
              <div
                v-for="(value, index) in [field.keep, field.remove, result(field)]"
                :key="index"
                class="min-w-0 rounded-md bg-muted/40 p-3"
              >
                <p :class="RESULT_LABEL_CLASS">
                  {{ index === 2 ? t('contact.merge.result') : names[index] }}
                </p>
                <Avatar v-if="field.key === 'avatar_url' && value" class="size-10"
                  ><AvatarImage :src="value" :alt="t('contact.merge.photo')" /><AvatarFallback
                    ><UserRoundIcon aria-hidden="true" /></AvatarFallback
                ></Avatar>
                <p v-else class="[overflow-wrap:anywhere]">{{ text(value, field.key) }}</p>
              </div>
            </div>
          </fieldset>
        </div>
        <p v-if="submitted && missing.length" role="alert" :class="ERROR_CLASS">
          {{ t('contact.merge.chooseConflicts') }}
        </p>
        <div
          role="note"
          class="flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/5 p-4 text-sm"
        >
          <AlertTriangleIcon class="mt-0.5 size-4 shrink-0 text-destructive" aria-hidden="true" />
          <p>{{ t('contact.merge.warning', { name: names[1] }) }}</p>
        </div>
        <p v-if="error" role="alert" :class="ERROR_CLASS">{{ error }}</p>
        <DialogFooter>
          <Button type="button" variant="outline" :disabled="merging" @click="close(false)">{{
            t('globals.messages.cancel')
          }}</Button>
          <Button
            v-if="stale"
            type="button"
            :disabled="loading"
            @click="loadPreview(preview.keep.id, preview.remove.id)"
            >{{ t('contact.merge.refresh') }}</Button
          >
          <Button
            v-else
            type="submit"
            :disabled="merging || loading"
            :is-loading="merging"
            class="whitespace-normal h-auto min-h-9 text-left break-words"
            >{{ t('contact.merge.confirm', { name: names[0] }) }}</Button
          >
        </DialogFooter>
      </form>
      <p v-else-if="error" role="alert" :class="ERROR_CLASS">{{ error }}</p>
    </DialogContent>
  </Dialog>
</template>
