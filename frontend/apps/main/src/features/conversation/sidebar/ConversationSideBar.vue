<template>
  <Tabs v-model="activeTab" class="flex flex-col h-full">
    <TabsList class="grid grid-cols-2 mx-4 mt-3">
      <TabsTrigger value="details" class="text-xs">{{ $t('copilot.details') }}</TabsTrigger>
      <TabsTrigger value="copilot" class="text-xs">{{ appSettingsStore.copilotName }}</TabsTrigger>
    </TabsList>

    <TabsContent value="details" class="mt-0">
      <ConversationSideBarContact class="p-4" />
      <Accordion type="multiple" collapsible v-model="accordionState">
        <AccordionItem value="actions" class="accordion-item">
          <AccordionTrigger class="accordion-trigger">
            {{ $t('globals.terms.action', 2) }}
          </AccordionTrigger>

          <!-- Agent, team, priority, and tags assignment -->
          <AccordionContent class="accordion-content--actions">
            <div>
              <SelectComboBox
                v-model="conversationStore.current.assigned_user_id"
                :items="[{ value: 'none', label: t('globals.terms.none') }, ...usersStore.options]"
                :placeholder="t('placeholders.selectAgent')"
                @select="selectAgent"
                type="user"
              />
            </div>

            <div>
              <SelectComboBox
                v-model="conversationStore.current.assigned_team_id"
                :items="[{ value: 'none', label: t('globals.terms.none') }, ...teamsStore.options]"
                :placeholder="t('placeholders.selectTeam')"
                @select="selectTeam"
                type="team"
              />
            </div>

            <div>
              <SelectComboBox
                v-model="conversationStore.current.priority_id"
                :items="priorityOptions"
                :placeholder="t('placeholders.selectPriority')"
                @select="selectPriority"
                type="priority"
              />
            </div>

            <div v-if="conversationStore.current">
              <SelectTag
                :model-value="conversationStore.current.tags || []"
                @update:modelValue="onTagsChange"
                :items="tags.map((tag) => ({ label: tag, value: tag }))"
                :placeholder="t('placeholders.selectTags')"
              />
              <div class="mt-2 flex flex-wrap items-center gap-1">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  class="h-6 gap-1 px-2 text-xs text-muted-foreground"
                  :disabled="isSuggestingTags"
                  @click="suggestTags"
                >
                  <Loader2 v-if="isSuggestingTags" class="h-3 w-3 animate-spin" />
                  <Sparkles v-else class="h-3 w-3" />
                  {{ t('conversation.sidebar.suggestTags') }}
                </Button>
                <button
                  v-for="suggestion in tagSuggestions"
                  :key="suggestion"
                  type="button"
                  class="inline-flex items-center gap-1 rounded-md bg-accent px-2 py-0.5 text-xs text-accent-foreground hover:bg-accent/80"
                  @click="applySuggestedTag(suggestion)"
                >
                  <Plus class="h-3 w-3" />
                  {{ suggestion }}
                </button>
              </div>
            </div>
          </AccordionContent>
        </AccordionItem>

        <!-- Information -->
        <AccordionItem value="information" class="accordion-item">
          <AccordionTrigger class="accordion-trigger">
            {{ $t('conversation.sidebar.information') }}
          </AccordionTrigger>
          <AccordionContent class="accordion-content">
            <ConversationInfo />
          </AccordionContent>
        </AccordionItem>

        <!-- Contact attributes -->
        <AccordionItem
          value="contact_attributes"
          class="accordion-item"
          v-if="customAttributeStore.contactAttributeOptions.length > 0"
        >
          <AccordionTrigger class="accordion-trigger">
            {{ $t('conversation.sidebar.contactAttributes') }}
          </AccordionTrigger>
          <AccordionContent class="accordion-content">
            <CustomAttributes
              :loading="conversationStore.current.loading"
              :attributes="customAttributeStore.contactAttributeOptions"
              :customAttributes="conversationStore.current?.contact?.custom_attributes || {}"
              @update:setattributes="updateContactCustomAttributes"
            />
          </AccordionContent>
        </AccordionItem>

        <!-- Page visits (livechat only) -->
        <AccordionItem
          value="page_visits"
          class="accordion-item"
          v-if="conversationStore.current?.inbox_channel === 'livechat'"
        >
          <AccordionTrigger class="accordion-trigger">
            {{ $t('conversation.sidebar.lastVisitedPages') }}
          </AccordionTrigger>
          <AccordionContent class="accordion-content">
            <ConversationSideBarPageVisits />
          </AccordionContent>
        </AccordionItem>

        <!-- Contact notes -->
        <AccordionItem
          value="contact_notes"
          class="accordion-item"
          v-if="conversationStore.current?.contact?.id && userStore.can('contact_notes:read')"
        >
          <AccordionTrigger class="accordion-trigger">
            {{ $t('globals.terms.note', 2) }}
          </AccordionTrigger>
          <AccordionContent class="accordion-content">
            <ContactNotes :contact-id="conversationStore.current.contact.id" compact />
          </AccordionContent>
        </AccordionItem>

        <!-- Previous conversations -->
        <AccordionItem value="previous_conversations" class="accordion-item">
          <AccordionTrigger class="accordion-trigger">
            {{ $t('conversation.sidebar.previousConvo') }}
          </AccordionTrigger>
          <AccordionContent class="accordion-content">
            <PreviousConversations />
          </AccordionContent>
        </AccordionItem>
      </Accordion>
    </TabsContent>

    <TabsContent value="copilot" class="flex-1 min-h-0 mt-0">
      <CopilotPanel />
    </TabsContent>
  </Tabs>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { Sparkles, Loader2, Plus } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useConversationStore } from '@/stores/conversation'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useTagStore } from '@/stores/tag'
import { useUserStore } from '@/stores/user'
import { useAppSettingsStore } from '@/stores/appSettings'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@shared-ui/components/ui/accordion'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@shared-ui/components/ui/tabs'
import ConversationInfo from './ConversationInfo.vue'
import ConversationSideBarContact from '@/features/conversation/sidebar/ConversationSideBarContact.vue'
import CopilotPanel from '@/features/conversation/sidebar/CopilotPanel.vue'
import { SelectTag } from '@shared-ui/components/ui/select'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '../../../stores/customAttributes'
import ContactNotes from '@/features/contact/ContactNotes.vue'
import PreviousConversations from '@/features/conversation/sidebar/PreviousConversations.vue'
import ConversationSideBarPageVisits from '@/features/conversation/sidebar/ConversationSideBarPageVisits.vue'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import { TAG_ACTION } from '@/constants/conversation'
import api from '../../../api'

const customAttributeStore = useCustomAttributeStore()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tagStore = useTagStore()
const userStore = useUserStore()
const appSettingsStore = useAppSettingsStore()
const tags = ref([])
const accordionState = useStorage('conversation-sidebar-accordion', [])
const activeTab = useStorage('conversation-sidebar-tab', 'details')
const { t } = useI18n()
customAttributeStore.fetchCustomAttributes()

onMounted(async () => {
  await fetchTags()
})

const onTagsChange = (newTags) => {
  const conv = conversationStore.current
  if (!conv) return
  const current = conv.tags || []
  if (newTags.length === current.length && newTags.every((t) => current.includes(t))) return
  conversationStore.updateConversationTags(conv.uuid, TAG_ACTION.SET, newTags)
}

const isSuggestingTags = ref(false)
const tagSuggestions = ref([])

// Suggestions belong to one conversation; drop them when the sidebar switches so they never leak.
watch(
  () => conversationStore.current?.uuid,
  () => {
    tagSuggestions.value = []
  }
)

const suggestTags = async () => {
  const conv = conversationStore.current
  if (!conv || isSuggestingTags.value) return
  const uuid = conv.uuid
  isSuggestingTags.value = true
  try {
    const resp = await api.aiSuggestTags({ conversation_uuid: uuid })
    if (conversationStore.current?.uuid !== uuid) return
    const current = conversationStore.current.tags || []
    const suggestions = (resp.data.data || []).filter((tag) => !current.includes(tag))
    if (suggestions.length === 0) {
      tagSuggestions.value = []
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('conversation.sidebar.noTagSuggestions')
      })
      return
    }
    tagSuggestions.value = suggestions
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSuggestingTags.value = false
  }
}

const applySuggestedTag = (tag) => {
  const conv = conversationStore.current
  if (!conv) return
  const current = conv.tags || []
  if (!current.includes(tag)) onTagsChange([...current, tag])
  tagSuggestions.value = tagSuggestions.value.filter((suggestion) => suggestion !== tag)
}

const priorityOptions = computed(() => conversationStore.priorityOptions)

const fetchTags = async () => {
  await tagStore.fetchTags()
  tags.value = tagStore.tags.map((item) => item.name)
}

const handleAssignedUserChange = (id) => {
  conversationStore.updateAssignee('user', {
    assignee_id: parseInt(id)
  })
}

const handleAssignedTeamChange = (id) => {
  conversationStore.updateAssignee('team', {
    assignee_id: parseInt(id)
  })
}

const handleRemoveAssignee = (type) => {
  conversationStore.removeAssignee(type)
}

const handlePriorityChange = (priority) => {
  conversationStore.updatePriority(priority)
}

const selectAgent = (agent) => {
  if (agent.value === 'none') {
    handleRemoveAssignee('user')
    return
  }
  conversationStore.current.assigned_user_id = agent.value
  handleAssignedUserChange(agent.value)
}

const selectTeam = (team) => {
  if (team.value === 'none') {
    handleRemoveAssignee('team')
    return
  }
  handleAssignedTeamChange(team.value)
}

const selectPriority = (priority) => {
  conversationStore.current.priority = priority.label
  conversationStore.current.priority_id = priority.value
  handlePriorityChange(priority.label)
}

const updateContactCustomAttributes = async (attributes) => {
  let previousAttributes = conversationStore.current.contact.custom_attributes
  try {
    conversationStore.current.contact.custom_attributes = attributes
    await api.updateContactCustomAttribute(conversationStore.current.uuid, attributes)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
    conversationStore.current.contact.custom_attributes = previousAttributes
  }
}
</script>

<style scoped>
:deep(.accordion-item) {
  @apply border-0 mb-2;
}

:deep(.accordion-trigger) {
  @apply bg-muted p-2 text-sm font-medium rounded mx-2;
}

:deep(.accordion-content) {
  @apply p-4;
}

:deep(.accordion-content--actions) {
  @apply space-y-3 px-4 pt-4 pb-0;
}
</style>
