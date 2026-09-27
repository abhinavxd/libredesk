<template>
  <form @submit.prevent="createConversation" novalidate class="flex flex-col flex-1 min-h-0">
    <div class="shrink-0">
      <div :class="[ROW_CLASS, 'flex items-center gap-2']">
        <span :class="ROW_LABEL_CLASS">{{ $t('globals.terms.from') }}</span>
        <Select v-model="inboxId">
          <SelectTrigger :class="ROW_INPUT_CLASS" :aria-label="t('globals.terms.inbox')">
            <SelectValue :placeholder="t('placeholders.selectInbox')" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem
                v-for="option in inboxStore.whatsappOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      <div :class="ROW_CLASS">
        <div class="flex items-center gap-2">
          <span :class="ROW_LABEL_CLASS">{{ $t('globals.terms.to') }}</span>
          <div class="w-fit shrink-0">
            <ComboBox
              v-model="phoneCountryCode"
              :items="allCountries"
              :placeholder="t('globals.terms.select')"
              :buttonClass="phoneCountryCode ? COUNTRY_CODE_CLASS : `${COUNTRY_CODE_CLASS} text-muted-foreground/60`"
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
                <div class="flex items-center gap-1.5">
                  <span v-if="selected" class="text-base">{{ selected.emoji }}</span>
                  <span v-if="selected && selected.calling_code" class="text-sm">{{
                    selected.calling_code
                  }}</span>
                </div>
              </template>
            </ComboBox>
          </div>
          <Input
            ref="phoneInputRef"
            type="tel"
            v-model="phoneNumber"
            :class="ROW_INPUT_CLASS"
            inputmode="numeric"
            :aria-label="t('globals.terms.phoneNumber')"
            :placeholder="t('globals.messages.searchOrEnterNumber')"
            @input="handleSearchContacts"
            @keydown="handleSearchKeydown"
            @blur="clearSearchResults"
            autocomplete="off"
          />
        </div>

        <ContactSearchResults
          :results="searchResults"
          :highlighted-index="highlightedIndex"
          @select="selectContact"
        >
          <template #default="{ contact }">
            <div class="min-w-0">
              <p class="font-medium">{{ contact.first_name }} {{ contact.last_name }}</p>
              <p v-if="contact.phone_number" class="text-xs text-muted-foreground truncate">
                {{ contact.phone_number }}
              </p>
              <p v-if="contact.email" class="text-xs text-muted-foreground truncate">
                {{ contact.email }}
              </p>
              <div
                v-if="contact.external_user_id"
                class="flex items-center gap-1 text-xs text-muted-foreground"
              >
                <IdCard :size="12" class="flex-shrink-0" />
                <span class="truncate">{{ contact.external_user_id }}</span>
              </div>
            </div>
          </template>
        </ContactSearchResults>
      </div>

      <div :class="[ROW_CLASS, 'flex items-center gap-2']">
        <span :class="ROW_LABEL_CLASS">{{ $t('globals.terms.name', 1) }}</span>
        <Input
          type="text"
          v-model="firstName"
          :class="ROW_INPUT_CLASS"
          :placeholder="t('globals.terms.firstName')"
          :aria-label="t('globals.terms.firstName')"
          :disabled="!!selectedContact"
        />
        <Input
          type="text"
          v-model="lastName"
          :class="ROW_INPUT_CLASS"
          :placeholder="t('globals.terms.lastName')"
          :aria-label="t('globals.terms.lastName')"
          :disabled="!!selectedContact"
        />
      </div>

      <div :class="[ROW_CLASS, 'grid grid-cols-1 sm:grid-cols-2 sm:gap-4']">
        <div class="flex items-center gap-2 min-w-0">
          <span :class="ROW_LABEL_CLASS">{{ $t('globals.terms.team', 1) }}</span>
          <SelectTeamCombobox
            v-model="teamId"
            include-none
            :button-class="isUnset(teamId) ? ROW_COMBOBOX_EMPTY_CLASS : ROW_COMBOBOX_CLASS"
          />
        </div>
        <div class="flex items-center gap-2 min-w-0">
          <span :class="ROW_LABEL_CLASS">{{ $t('globals.terms.agent', 1) }}</span>
          <SelectAgentCombobox
            v-model="agentId"
            include-none
            :button-class="isUnset(agentId) ? ROW_COMBOBOX_EMPTY_CLASS : ROW_COMBOBOX_CLASS"
          />
        </div>
      </div>
    </div>

    <Alert v-if="existingConversation?.exists" class="m-3 w-auto">
      <AlertDescription class="flex items-center justify-between gap-4">
        <span>{{ existingConversationMessage }}</span>
        <Button
          v-if="existingConversation.uuid"
          type="button"
          size="sm"
          variant="outline"
          class="shrink-0"
          @click="openExistingConversation"
        >
          {{ $t('globals.messages.openConversation') }}
        </Button>
      </AlertDescription>
    </Alert>

    <div v-else class="flex-1 flex flex-col min-h-0 px-3 pt-3">
      <p class="text-sm text-muted-foreground mb-2">{{ $t('globals.terms.template', 1) }}</p>

      <template v-if="inboxId">
        <WhatsAppTemplatePicker
          fill
          class="flex-1"
          :approved-templates="approvedTemplates"
          :selected-template="selectedTemplate"
          :template-params="templateParams"
          :placeholders="placeholders"
          :url-button-params="urlButtonParams"
          :rendered-preview="renderedPreview"
          :is-fetching="isFetchingTemplates"
          @pick="pickTemplate"
          @back="selectedTemplate = null"
          @update:param="(key, v) => (templateParams[key] = v)"
        />
      </template>
    </div>

    <div v-if="!existingConversation?.exists" class="flex justify-end px-3 py-2 shrink-0">
      <Button type="submit" :disabled="!canSubmit || loading" :isLoading="loading">
        {{ $t('globals.messages.sendTemplate') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { IdCard } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { Alert, AlertDescription } from '@shared-ui/components/ui/alert'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useInboxStore } from '@main/stores/inbox'
import { useUserStore } from '@main/stores/user'
import { countryCallingOptions as allCountries } from '@shared-ui/constants/countries.js'
import { useWhatsAppTemplatePicker } from './useWhatsAppTemplatePicker.js'
import WhatsAppTemplatePicker from './WhatsAppTemplatePicker.vue'
import { useContactSearch } from './useContactSearch.js'
import ContactSearchResults from './ContactSearchResults.vue'
import api from '@/api'
import { useNewConversationDraft } from './useNewConversationDraft.js'
import {
  ROW_CLASS,
  ROW_LABEL_CLASS,
  ROW_INPUT_CLASS,
  ROW_COMBOBOX_CLASS,
  ROW_COMBOBOX_EMPTY_CLASS,
  isUnset
} from '@/features/conversation/composerRowClasses.js'

const COUNTRY_CODE_CLASS = 'h-9 min-w-16 border-0 bg-transparent px-0 font-normal shadow-none hover:bg-transparent'

const emit = defineEmits(['close'])
const props = defineProps({
  initialContact: { type: Object, default: null }
})

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()
const inboxStore = useInboxStore()
const userStore = useUserStore()

const {
  selectedTemplate,
  templateParams,
  approvedTemplates,
  placeholders,
  urlButtonParams,
  allParamsFilled,
  renderedPreview,
  isFetchingTemplates,
  pickTemplate,
  fetchTemplates
} = useWhatsAppTemplatePicker()

const inboxId = ref('')
const teamId = ref(null)
const agentId = ref(userStore.userID ? String(userStore.userID) : 'none')
const loading = ref(false)

const selectedContact = ref(null)
const phoneInputRef = ref(null)

const firstName = ref('')
const lastName = ref('')
const phoneCountryCode = ref('')
const phoneNumber = ref('')

const { searchResults, highlightedIndex, handleSearchContacts, handleSearchKeydown, selectContact, clearSearchResults } =
  useContactSearch({
    getQuery: () => phoneNumber.value,
    filterResults: (c) => c.phone_number,
    onSelect: (contact) => {
      selectedContact.value = contact
      phoneNumber.value = contact.phone_number || ''
      phoneCountryCode.value = contact.phone_number_country_code || ''
      firstName.value = contact.first_name || ''
      lastName.value = contact.last_name || ''
    }
  })

const draft = useNewConversationDraft('whatsapp')
let templateDraft = draft.value
const DRAFT_FIELDS = { inboxId, teamId, agentId, phoneCountryCode, phoneNumber, firstName, lastName }

onMounted(() => {
  if (draft.value) {
    selectedContact.value = draft.value.contact || null
    Object.entries(DRAFT_FIELDS).forEach(([key, field]) => {
      if (key in draft.value) field.value = draft.value[key]
    })
  }
  if (props.initialContact?.phone_number) selectContact(props.initialContact)
})

defineExpose({ focus: () => phoneInputRef.value?.$el?.focus() })

watch(inboxId, (id) => {
  const selection = templateDraft?.inboxId === id
    ? { templateId: templateDraft.templateId, params: templateDraft.templateParams }
    : {}
  templateDraft = null
  fetchTemplates(id, selection)
})

watch([selectedContact, ...Object.values(DRAFT_FIELDS), selectedTemplate, templateParams, isFetchingTemplates], () => {
  if (isFetchingTemplates.value) return
  const hasContent = selectedTemplate.value || [phoneNumber, firstName, lastName].some((f) => f.value?.trim())
  draft.value = hasContent
    ? {
        contact: selectedContact.value,
        templateId: selectedTemplate.value?.id ?? null,
        templateParams: { ...templateParams },
        ...Object.fromEntries(Object.entries(DRAFT_FIELDS).map(([k, f]) => [k, f.value]))
      }
    : null
}, { deep: true })

watch(
  () => props.initialContact,
  (contact) => {
    if (contact?.phone_number) selectContact(contact)
  }
)

const existingConversation = ref(null)
const existingConversationMessage = computed(() =>
  t(
    existingConversation.value?.uuid
      ? 'conversation.whatsapp.error.conversationExists'
      : 'conversation.whatsapp.error.conversationExistsNoAccess'
  )
)

watch([selectedContact, inboxId], async ([contact, inbox]) => {
  existingConversation.value = null
  if (!contact || !inbox) return
  try {
    const resp = await api.getWhatsAppOpenConversation(contact.id, Number(inbox))
    if (contact === selectedContact.value && inbox === inboxId.value) {
      existingConversation.value = resp.data.data
    }
  } catch {
    // A failed lookup leaves the form usable. The create call rejects a duplicate anyway.
  }
})

const goToConversation = (uuid) => {
  draft.value = null
  emit('close')
  router.push({ name: 'inbox-conversation', params: { uuid, type: 'assigned' } })
}

const openExistingConversation = () => goToConversation(existingConversation.value.uuid)

const hasContact = computed(() => {
  if (selectedContact.value) return true
  return (
    firstName.value.trim() !== '' &&
    phoneNumber.value.trim() !== '' &&
    phoneCountryCode.value !== ''
  )
})

const canSubmit = computed(
  () =>
    !!inboxId.value &&
    !!selectedTemplate.value &&
    allParamsFilled.value &&
    hasContact.value &&
    !existingConversation.value?.exists
)

watch([phoneNumber, phoneCountryCode], ([num, code]) => {
  if (!selectedContact.value) return
  if (
    num !== (selectedContact.value.phone_number || '') ||
    code !== (selectedContact.value.phone_number_country_code || '')
  ) {
    selectedContact.value = null
    firstName.value = ''
    lastName.value = ''
  }
})

const createConversation = async () => {
  if (!canSubmit.value) return
  loading.value = true
  try {
    const payload = {
      inbox_id: Number(inboxId.value),
      team_id: teamId.value && teamId.value !== 'none' ? Number(teamId.value) : null,
      agent_id: agentId.value && agentId.value !== 'none' ? Number(agentId.value) : null,
      whatsapp_template_id: selectedTemplate.value.id,
      whatsapp_template_params: { ...templateParams }
    }
    if (selectedContact.value) {
      payload.contact_id = selectedContact.value.id
    } else {
      payload.first_name = firstName.value
      payload.last_name = lastName.value
      payload.phone_number = phoneNumber.value
      payload.phone_number_country_code = phoneCountryCode.value
    }
    await api.createConversation(payload)
    draft.value = null
    emit('close')
  } catch (error) {
    const err = handleHTTPError(error)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: err.message
    })
    if (err.data?.conversation_uuid) goToConversation(err.data.conversation_uuid)
  } finally {
    loading.value = false
  }
}
</script>
