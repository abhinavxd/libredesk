<template>
  <form @submit.prevent="createConversation" class="flex flex-col flex-1 overflow-hidden">
    <div class="space-y-4 flex-shrink-0">
      <div class="grid grid-cols-4 gap-4">
        <div class="space-y-2 relative col-span-2 min-w-0">
          <label class="text-sm font-medium">{{ $t('conversation.whatsapp.number') }}</label>
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
              autocomplete="off"
            />
          </div>

          <div
            v-if="searchResults.length"
            class="absolute w-full z-50 mt-1 rounded-md border bg-popover p-1 text-popover-foreground shadow-md"
          >
            <ul class="max-h-60 overflow-y-auto" role="listbox">
              <li
                v-for="(contact, index) in searchResults"
                :key="contact.id"
                @click="selectContact(contact)"
                role="option"
                :aria-selected="index === highlightedIndex"
                class="relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors duration-200"
                :class="
                  index === highlightedIndex
                    ? 'bg-accent text-accent-foreground'
                    : 'hover:bg-accent hover:text-accent-foreground'
                "
              >
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
              </li>
            </ul>
          </div>
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
          <SelectComboBox
            v-model="teamId"
            :items="[{ value: 'none', label: t('globals.terms.none') }, ...teamStore.options]"
            :placeholder="t('placeholders.selectTeam')"
            type="team"
          />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium">
            {{ $t('actions.assignAgent') }} ({{ $t('globals.terms.optional') }})
          </label>
          <SelectComboBox
            v-model="agentId"
            :items="[{ value: 'none', label: t('globals.terms.none') }, ...uStore.options]"
            :placeholder="t('placeholders.selectAgent')"
            type="user"
          />
        </div>
      </div>
    </div>

    <div class="flex-1 flex flex-col min-h-0 mt-4">
      <label class="text-sm font-medium mb-2">{{ $t('conversation.whatsapp.template') }}</label>

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
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
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
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useInboxStore } from '@main/stores/inbox'
import { useUsersStore } from '@main/stores/users'
import { useUserStore } from '@main/stores/user'
import { useTeamStore } from '@main/stores/team'
import { useCountriesStore } from '@main/stores/countries'
import { useWhatsAppTemplatePicker } from './useWhatsAppTemplatePicker.js'
import WhatsAppTemplatePicker from './WhatsAppTemplatePicker.vue'
import api from '@/api'

const emit = defineEmits(['close'])

const { t } = useI18n()
const emitter = useEmitter()
const inboxStore = useInboxStore()
const uStore = useUsersStore()
const userStore = useUserStore()
const teamStore = useTeamStore()
const countriesStore = useCountriesStore()
const { allCountries } = storeToRefs(countriesStore)

const {
  templates,
  selectedTemplate,
  templateParams,
  approvedTemplates,
  placeholders,
  urlButtonParams,
  allParamsFilled,
  renderedPreview,
  pickTemplate,
  reset
} = useWhatsAppTemplatePicker()

const inboxId = ref('')
const teamId = ref('none')
const agentId = ref(userStore.userID ? String(userStore.userID) : 'none')
const loading = ref(false)

const searchResults = ref([])
const highlightedIndex = ref(-1)
const selectedContact = ref(null)
let timeoutId = null

const firstName = ref('')
const lastName = ref('')
const phoneCountryCode = ref('')
const phoneNumber = ref('')
const isFetchingTemplates = ref(false)

onMounted(() => countriesStore.fetchCountries())
onUnmounted(() => clearTimeout(timeoutId))

const fetchTemplates = async () => {
  reset()
  if (!inboxId.value) return
  try {
    isFetchingTemplates.value = true
    const resp = await api.getWhatsAppTemplates(inboxId.value)
    templates.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isFetchingTemplates.value = false
  }
}

watch(inboxId, fetchTemplates)

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

const handleSearchContacts = () => {
  clearTimeout(timeoutId)
  timeoutId = setTimeout(async () => {
    const query = phoneNumber.value.trim()
    if (query.length < 3) {
      searchResults.value.splice(0)
      return
    }
    try {
      const resp = await api.searchContacts({ query })
      searchResults.value = [...resp.data.data]
      highlightedIndex.value = -1
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
      searchResults.value.splice(0)
    }
  }, 300)
}

const handleSearchKeydown = (e) => {
  if (!searchResults.value.length) return
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    highlightedIndex.value = Math.min(highlightedIndex.value + 1, searchResults.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    highlightedIndex.value = Math.max(highlightedIndex.value - 1, 0)
  } else if (e.key === 'Enter' && highlightedIndex.value >= 0) {
    e.preventDefault()
    selectContact(searchResults.value[highlightedIndex.value])
  } else if (e.key === 'Escape') {
    searchResults.value.splice(0)
    highlightedIndex.value = -1
  }
}

const selectContact = (contact) => {
  selectedContact.value = contact
  phoneNumber.value = contact.phone_number || ''
  phoneCountryCode.value = contact.phone_number_country_code || ''
  firstName.value = contact.first_name || ''
  lastName.value = contact.last_name || ''
  searchResults.value.splice(0)
  highlightedIndex.value = -1
}

watch(phoneNumber, (val) => {
  if (selectedContact.value && val !== (selectedContact.value.phone_number || '')) {
    selectedContact.value = null
    firstName.value = ''
    lastName.value = ''
  }
})

watch(phoneCountryCode, (val) => {
  if (selectedContact.value && val !== (selectedContact.value.phone_number_country_code || '')) {
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
