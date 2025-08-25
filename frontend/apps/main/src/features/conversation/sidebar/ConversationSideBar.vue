<template>
  <div>
    <ConversationSideBarContact class="p-4" />
    <Accordion type="multiple" collapsible v-model="accordionState">
      <AccordionItem value="actions" class="border-0 mb-2">
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded mx-2">
          {{ $t('globals.terms.action', 2) }}
        </AccordionTrigger>

        <!-- `Agent, team, and priority assignment -->
        <AccordionContent class="space-y-4 p-4">
          <!-- Agent assignment -->
          <SelectComboBox
            v-model="conversationStore.current.assigned_user_id"
            :items="[{ value: 'none', label: 'None' }, ...usersStore.options]"
            :placeholder="t('globals.messages.select', { name: t('globals.terms.agent').toLowerCase() })"
            @select="selectAgent"
            type="user"
          />

          <!-- Team assignment -->
          <SelectComboBox
            v-model="conversationStore.current.assigned_team_id"
            :items="[{ value: 'none', label: 'None' }, ...teamsStore.options]"
            :placeholder="t('globals.messages.select', { name: t('globals.terms.team').toLowerCase() })"
            @select="selectTeam"
            type="team"
          />

          <!-- Priority assignment -->
          <SelectComboBox
            v-model="conversationStore.current.priority_id"
            :items="priorityOptions"
            :placeholder="t('globals.messages.select', { name: t('globals.terms.priority').toLowerCase() })"
            @select="selectPriority"
            type="priority"
          />

          <!-- Tags assignment -->
          <SelectTag
            v-if="conversationStore.current"
            v-model="conversationStore.current.tags"
            :items="tags.map((tag) => ({ label: tag, value: tag }))"
            :placeholder="t('globals.messages.select', { name: t('globals.terms.tag', 2).toLowerCase() })"
          />
        </AccordionContent>
      </AccordionItem>

      <!-- Information -->
      <AccordionItem value="information" class="border-0 mb-2">
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded mx-2">
          {{ $t('conversation.sidebar.information') }}
        </AccordionTrigger>
        <AccordionContent class="p-4">
          <ConversationInfo />
        </AccordionContent>
      </AccordionItem>

      <!-- Contact attributes -->
      <AccordionItem
        value="contact_attributes"
        class="border-0 mb-2"
        v-if="customAttributeStore.contactAttributeOptions.length > 0"
      >
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded mx-2">
          {{ $t('conversation.sidebar.contactAttributes') }}
        </AccordionTrigger>
        <AccordionContent class="p-4">
          <CustomAttributes
            :loading="conversationStore.current.loading"
            :attributes="customAttributeStore.contactAttributeOptions"
            :customAttributes="conversationStore.current?.contact?.custom_attributes || {}"
            @update:setattributes="updateContactCustomAttributes"
          />
        </AccordionContent>
      </AccordionItem>

      <!-- Previous conversations -->
      <AccordionItem value="previous_conversations" class="border-0 mb-2">
        <AccordionTrigger class="bg-muted px-4 py-3 text-sm font-medium rounded mx-2">
          {{ $t('conversation.sidebar.previousConvo') }}
        </AccordionTrigger>
        <AccordionContent class="p-4">
          <PreviousConversations />
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue'
import { useConversationStore } from '../../../stores/conversation'
import { useUsersStore } from '../../../stores/users'
import { useTeamStore } from '../../../stores/team'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@shared-ui/components/ui/accordion'
import ConversationInfo from './ConversationInfo.vue'
import ConversationSideBarContact from '@/features/conversation/sidebar/ConversationSideBarContact.vue'
import { SelectTag } from '@shared-ui/components/ui/select'
import { handleHTTPError } from '../../../utils/http'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '../../../stores/customAttributes'
import PreviousConversations from '@/features/conversation/sidebar/PreviousConversations.vue'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import api from '../../../api'

const customAttributeStore = useCustomAttributeStore()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamsStore = useTeamStore()
const tags = ref([])
// Save the accordion state in local storage
const accordionState = useStorage('conversation-sidebar-accordion', [])
const { t } = useI18n()
let isConversationChange = false
customAttributeStore.fetchCustomAttributes()

// Watch for changes in the current conversation and set the flag
watch(
  () => conversationStore.current,
  (newConversation, oldConversation) => {
    // Set the flag when the conversation changes
    if (newConversation?.uuid !== oldConversation?.uuid) {
      isConversationChange = true
    }
  },
  { immediate: true }
)

onMounted(async () => {
  await fetchTags()
})

// Watch for changes in the tags and upsert the tags
watch(
  () => conversationStore.current?.tags,
  (newTags, oldTags) => {
    // Skip if the tags change is due to a conversation change.
    if (isConversationChange) {
      isConversationChange = false
      return
    }

    // Skip if the tags are the same (deep comparison)
    if (
      Array.isArray(newTags) &&
      Array.isArray(oldTags) &&
      newTags.length === oldTags.length &&
      newTags.every((item) => oldTags.includes(item))
    ) {
      return
    }

    conversationStore.upsertTags({
      tags: newTags
    })
  },
  { immediate: false }
)

const priorityOptions = computed(() => conversationStore.priorityOptions)

const fetchTags = async () => {
  try {
    const resp = await api.getTags()
    tags.value = resp.data.data.map((item) => item.name)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
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
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.attribute')
      })
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
