<template>
  <form @submit.prevent="createConversation" novalidate class="flex flex-col flex-1 overflow-hidden">
    <div class="space-y-4 flex-shrink-0">
      <div class="grid grid-cols-4 gap-4">
        <div class="space-y-2 relative col-span-2 min-w-0">
          <label class="text-sm font-medium">{{ $t('globals.terms.phoneNumber') }}</label>
          <div class="flex items-end">
            <div class="w-fit shrink-0">
              <ComboBox
                v-model="phoneCountryCode"
                :items="allCountries"
                :placeholder="t('globals.terms.select')"
                :buttonClass="'rounded-r-none border-r-0 min-w-20'"
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
                    <span v-if="selected && selected.calling_code" class="text-sm font-medium">{{
                      selected.calling_code
                    }}</span>
                  </div>
                </template>
              </ComboBox>
            </div>
            <Input
              type="tel"
              v-model="phoneNumber"
              class="rounded-l-none flex-1"
              inputmode="numeric"
              :placeholder="t('conversation.whatsapp.numberPlaceholder')"
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

        <div class="space-y-2 min-w-0">
          <label class="text-sm font-medium">{{ $t('globals.terms.firstName') }}</label>
          <Input type="text" v-model="firstName" :disabled="!!selectedContact" />
        </div>
        <div class="space-y-2 min-w-0">
          <label class="text-sm font-medium">{{ $t('globals.terms.lastName') }}</label>
          <Input type="text" v-model="lastName" :disabled="!!selectedContact" />
        </div>
      </div>

      <div class="grid grid-cols-3 gap-4">
        <div class="space-y-2">
          <label class="text-sm font-medium">{{ $t('globals.terms.inbox') }}</label>
          <Select v-model="inboxId">
            <SelectTrigger>
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
        <div class="space-y-2">
          <label class="text-sm font-medium">
            {{ $t('actions.assignTeam') }} ({{ $t('globals.terms.optional') }})
          </label>
          <SelectTeamCombobox v-model="teamId" include-none />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium">
            {{ $t('actions.assignAgent') }} ({{ $t('globals.terms.optional') }})
          </label>
          <SelectAgentCombobox v-model="agentId" include-none />
        </div>
      </div>
    </div>

    <div class="flex-1 flex flex-col min-h-0 mt-4">
      <label class="text-sm font-medium mb-2">{{ $t('globals.terms.template', 1) }}</label>

      <p v-if="!inboxId" class="text-sm text-muted-foreground">
        {{ $t('conversation.whatsapp.selectInboxFirst') }}
      </p>
      <template v-else>
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

    <DialogFooter class="mt-4 pt-2 flex-shrink-0">
      <Button type="submit" :disabled="!canSubmit || loading" :isLoading="loading">
        {{ $t('globals.messages.submit') }}
      </Button>
    </DialogFooter>
  </form>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { IdCard } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { DialogFooter } from '@shared-ui/components/ui/dialog'
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

const emit = defineEmits(['close'])
const props = defineProps({
  initialContact: { type: Object, default: null }
})

const { t } = useI18n()
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
const teamId = ref('none')
const agentId = ref(userStore.userID ? String(userStore.userID) : 'none')
const loading = ref(false)

const selectedContact = ref(null)

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

onMounted(() => {
  if (props.initialContact?.phone_number) selectContact(props.initialContact)
})

watch(
  () => props.initialContact,
  (contact) => {
    if (contact?.phone_number) selectContact(contact)
  }
)

watch(inboxId, (id) => fetchTemplates(id))

const hasContact = computed(() => {
  if (selectedContact.value) return true
  return (
    firstName.value.trim() !== '' &&
    phoneNumber.value.trim() !== '' &&
    phoneCountryCode.value !== ''
  )
})

const canSubmit = computed(
  () => !!inboxId.value && !!selectedTemplate.value && allParamsFilled.value && hasContact.value
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
    emit('close')
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
}
</script>
