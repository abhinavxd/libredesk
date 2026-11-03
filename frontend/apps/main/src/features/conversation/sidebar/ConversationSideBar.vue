<template>
  <Tabs v-model="activeTab" class="flex h-full min-h-0 flex-col" @keydown.capture="copilotPointerOpened = false">
    <div class="flex h-14 shrink-0 items-center border-b border-border px-3">
      <TabsList class="h-full w-full justify-start gap-6 rounded-none bg-transparent p-0">
        <TabsTrigger value="details" :class="SIDEBAR_TAB_CLASS">{{ $t('copilot.details') }}</TabsTrigger>
        <TabsTrigger value="copilot" :class="SIDEBAR_TAB_CLASS" @pointerdown="copilotPointerOpened = activeTab !== 'copilot'">{{ COPILOT_NAME }}</TabsTrigger>
      </TabsList>
    </div>

    <TabsContent value="details" class="mt-0 min-h-0 flex-1 overflow-y-auto overscroll-contain">
      <ConversationSideBarContact class="border-b border-border p-4">
        <template #actions>
          <Tooltip>
            <TooltipTrigger as-child>
              <Button
                ref="sectionEditorButton"
                type="button"
                variant="ghost"
                size="icon"
                class="h-11 w-11 shrink-0 text-muted-foreground hover:text-foreground focus-visible:text-foreground lg:h-7 lg:w-7 [&_svg]:size-3.5"
                :aria-label="t('globals.messages.reorderSections')"
                :aria-pressed="editingSections"
                @click="toggleSectionEditor"
              >
                <Pencil aria-hidden="true" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{{ t('globals.messages.reorderSections') }}</TooltipContent>
          </Tooltip>
        </template>
      </ConversationSideBarContact>
      <div v-if="editingSections" class="space-y-3 border-b border-border p-4">
        <Draggable
          v-model="editableSections"
          item-key="id"
          handle=".drag-handle"
          :animation="reducedMotion === 'reduce' ? 0 : 200"
          :force-fallback="true"
          fallback-on-body
          :fallback-tolerance="3"
          ghost-class="drag-ghost"
          tag="ol"
          class="space-y-1"
        >
          <template #item="{ element: section, index }">
              <li class="group/reorder flex items-center gap-2 rounded-md border border-border bg-background p-2">
              <span class="drag-handle cursor-move text-muted-foreground" aria-hidden="true">
                <GripVertical class="size-4" />
              </span>
              <span class="min-w-0 flex-1 text-sm">{{ section.label }}</span>
              <ReorderButtons
                :index="index"
                :length="editableSections.length"
                @move="moveSection(index, $event)"
              />
            </li>
          </template>
        </Draggable>
        <div class="flex justify-end gap-2">
          <Button type="button" variant="outline" size="sm" @click="closeSectionEditor">
            {{ t('globals.messages.cancel') }}
          </Button>
          <Button type="button" size="sm" @click="saveSectionOrder">
            {{ t('globals.messages.save') }}
          </Button>
        </div>
      </div>
      <Accordion v-show="!editingSections" type="multiple" collapsible v-model="accordionState">
        <AccordionItem
          v-for="section in visibleSections"
          :key="section.id"
          :value="section.id"
          class="accordion-item"
        >
          <AccordionTrigger class="accordion-trigger">{{ section.label }}</AccordionTrigger>

          <AccordionContent v-if="section.id === 'actions'" class="accordion-content--actions">
            <div>
              <SelectAgentCombobox
                v-model="conversationStore.current.assigned_user_id"
                include-none
                current-user-first
                @select="selectAgent"
              />
            </div>

            <div>
              <SelectTeamCombobox
                :model-value="conversationStore.current.assigned_team_id"
                include-none
                @select="selectTeam"
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
              <SelectTagCombobox
                multiple
                compact
                keep-open-on-select
                :model-value="conversationStore.current.tags || []"
                @update:modelValue="onTagsChange"
              >
                <template #action>
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        class="h-7 w-7 text-muted-foreground"
                        :aria-label="t('conversation.sidebar.suggestTags')"
                        :disabled="isSuggestingTags"
                        @click="suggestTags"
                      >
                        <Loader2 v-if="isSuggestingTags" class="animate-spin" aria-hidden="true" />
                        <Sparkles v-else aria-hidden="true" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{{ t('conversation.sidebar.suggestTags') }}</TooltipContent>
                  </Tooltip>
                </template>
              </SelectTagCombobox>
              <div v-if="tagSuggestions.length" class="mt-2 flex flex-wrap items-center gap-1">
                <Button
                  v-for="suggestion in tagSuggestions"
                  :key="suggestion"
                  type="button"
                  variant="secondary"
                  size="xs"
                  class="max-w-full gap-1 px-2 text-xs"
                  @click="applySuggestedTag(suggestion)"
                >
                  <Plus class="h-3 w-3" aria-hidden="true" />
                  <span class="truncate">{{ suggestion }}</span>
                </Button>
              </div>
            </div>
          </AccordionContent>

          <AccordionContent v-else-if="section.id === 'information'" class="accordion-content">
            <ConversationInfo />
          </AccordionContent>

          <AccordionContent
            v-else-if="section.id === 'contact_attributes'"
            class="accordion-content"
          >
            <CustomAttributes
              :loading="conversationStore.current.loading"
              :attributes="customAttributeStore.contactAttributeOptions"
              :customAttributes="conversationStore.current?.contact?.custom_attributes || {}"
              @update:setattributes="updateContactCustomAttributes"
            />
          </AccordionContent>

          <AccordionContent v-else-if="section.id === 'page_visits'" class="accordion-content">
            <ConversationSideBarPageVisits />
          </AccordionContent>

          <AccordionContent v-else-if="section.id === 'contact_notes'" class="accordion-content">
            <ContactNotes :contact-id="conversationStore.current.contact.id" compact />
          </AccordionContent>

          <AccordionContent
            v-else-if="section.id === 'previous_conversations'"
            class="accordion-content"
          >
            <PreviousConversations />
          </AccordionContent>
        </AccordionItem>
      </Accordion>
    </TabsContent>

    <TabsContent value="copilot" class="flex-1 min-h-0 mt-0" :class="{ 'copilot-entry': copilotPointerOpened }">
      <CopilotPanel />
    </TabsContent>
  </Tabs>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { Sparkles, Loader2, Plus, Pencil, GripVertical } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useConversationStore } from '@/stores/conversation'
import { useUserStore } from '@/stores/user'
import { COPILOT_NAME } from '@/constants/copilot'
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
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS, CONVERSATION_ACTIONS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { useStorage, usePreferredReducedMotion } from '@vueuse/core'
import Draggable from 'vuedraggable'
import ReorderButtons from '@shared-ui/components/ReorderButtons.vue'
import CustomAttributes from '@/features/conversation/sidebar/CustomAttributes.vue'
import { useCustomAttributeStore } from '@/stores/customAttributes'
import ContactNotes from '@/features/contact/ContactNotes.vue'
import PreviousConversations from '@/features/conversation/sidebar/PreviousConversations.vue'
import ConversationSideBarPageVisits from '@/features/conversation/sidebar/ConversationSideBarPageVisits.vue'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@main/components/combobox/SelectTeamCombobox.vue'
import SelectTagCombobox from '@main/components/combobox/SelectTagCombobox.vue'
import { TAG_ACTION } from '@/constants/conversation'
import api from '@/api'

const SIDEBAR_TAB_CLASS = 'h-full rounded-none border-b-2 border-transparent px-0 text-xs font-normal focus-visible:ring-inset focus-visible:ring-offset-0 data-[state=active]:border-foreground data-[state=active]:bg-transparent data-[state=active]:font-medium data-[state=active]:shadow-none'

const customAttributeStore = useCustomAttributeStore()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const userStore = useUserStore()
const accordionState = useStorage('conversation-sidebar-accordion', [])
const activeTab = useStorage('conversation-sidebar-tab', 'details')
const copilotPointerOpened = ref(false)
const { t } = useI18n()
customAttributeStore.fetchCustomAttributes()

const sections = computed(() => [
  { id: 'actions', label: t('globals.terms.action', 2) },
  { id: 'information', label: t('conversation.sidebar.information') },
  {
    id: 'contact_attributes',
    label: t('conversation.sidebar.contactAttributes'),
    visible: customAttributeStore.contactAttributeOptions.length > 0
  },
  {
    id: 'page_visits',
    label: t('conversation.sidebar.lastVisitedPages'),
    visible: conversationStore.current?.inbox_channel === 'livechat'
  },
  {
    id: 'contact_notes',
    label: t('globals.terms.note', 2),
    visible: !!conversationStore.current?.contact?.id,
    permitted: userStore.can('contact_notes:read')
  },
  { id: 'previous_conversations', label: t('globals.terms.conversation', 2) }
])
const sectionOrder = useStorage(
  computed(() => `conversation-sidebar-section-order:${userStore.userID}`),
  []
)
const normalizedOrder = computed(() => {
  const ids = sections.value.map((section) => section.id)
  const saved = Array.isArray(sectionOrder.value) ? sectionOrder.value : []
  return [...new Set([...saved.filter((id) => ids.includes(id)), ...ids])]
})
const visibleSections = computed(() =>
  normalizedOrder.value
    .map((id) => sections.value.find((section) => section.id === id))
    .filter((section) => section.visible !== false && section.permitted !== false)
)
const reducedMotion = usePreferredReducedMotion()
const editingSections = ref(false)
const sectionEditorButton = ref(null)
const draftOrder = ref([])
const editableSections = computed({
  get: () =>
    draftOrder.value
      .map((id) => sections.value.find((section) => section.id === id))
      .filter((section) => section && section.permitted !== false),
  set: (reordered) => {
    const editableIds = new Set(editableSections.value.map((section) => section.id))
    let index = 0
    draftOrder.value = draftOrder.value.map((id) =>
      editableIds.has(id) ? reordered[index++].id : id
    )
  }
})

const toggleSectionEditor = () => {
  if (!editingSections.value) draftOrder.value = [...normalizedOrder.value]
  editingSections.value = !editingSections.value
}

const moveSection = (index, direction) => {
  const reordered = [...editableSections.value]
  const [section] = reordered.splice(index, 1)
  reordered.splice(index + direction, 0, section)
  editableSections.value = reordered
}

const closeSectionEditor = async () => {
  editingSections.value = false
  await nextTick()
  sectionEditorButton.value?.$el?.focus()
}

const saveSectionOrder = () => {
  sectionOrder.value = [...draftOrder.value]
  closeSectionEditor()
}

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

const onPaletteAction = (action) => {
  if (action === CONVERSATION_ACTIONS.SUGGEST_TAGS) suggestTags()
}

onMounted(() => emitter.on(EMITTER_EVENTS.CONVERSATION_ACTION, onPaletteAction))
onUnmounted(() => emitter.off(EMITTER_EVENTS.CONVERSATION_ACTION, onPaletteAction))

const applySuggestedTag = (tag) => {
  const conv = conversationStore.current
  if (!conv) return
  const current = conv.tags || []
  if (!current.includes(tag)) onTagsChange([...current, tag])
  tagSuggestions.value = tagSuggestions.value.filter((suggestion) => suggestion !== tag)
}

const priorityOptions = computed(() => conversationStore.priorityOptions)

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
  const conv = conversationStore.current
  let previousAttributes = conv.contact.custom_attributes
  try {
    conv.contact.custom_attributes = attributes
    await api.updateContactCustomAttribute(conv.uuid, attributes)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
    conv.contact.custom_attributes = previousAttributes
  }
}
</script>

<style scoped>
.copilot-entry {
  @apply transition-opacity duration-100 ease-out;

  @starting-style {
    opacity: 0.85;
  }
}

@media (prefers-reduced-motion: reduce) {
  .copilot-entry {
    transition-duration: 60ms;

    @starting-style {
      opacity: 0.95;
    }
  }
}

:deep(.accordion-item) {
  @apply border-b border-border last:border-b-0;
}

:deep(.accordion-trigger) {
  @apply rounded-none px-4 py-3 text-sm font-medium can-hover:hover:bg-accent/40 hover:no-underline;
}

:deep(.accordion-content) {
  @apply p-4;
}

:deep(.accordion-content--actions) {
  @apply space-y-3 p-4;
}
</style>
