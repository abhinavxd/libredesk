<template>
  <router-link
    :to="{ name: 'inbox-conversation', params: { uuid: conversationUUID, type: 'assigned' } }"
    class="flex gap-4 px-5 py-4 hover:bg-accent/40 transition-colors"
  >
    <Avatar class="w-10 h-10 rounded-full shrink-0 mt-0.5">
      <AvatarImage :src="item.contact.avatar_url || ''" class="object-cover" />
      <AvatarFallback>{{ initials(item.contact.first_name) }}</AvatarFallback>
    </Avatar>

    <div class="min-w-0 flex-1">
      <div class="flex items-baseline justify-between gap-4">
        <div class="flex items-baseline gap-2 min-w-0 text-sm">
          <span class="font-medium text-foreground truncate">
            <HighlightedText :text="contactName" :term="term" />
          </span>
          <span v-if="item.contact.email" class="text-muted-foreground truncate hidden sm:inline">
            <HighlightedText :text="item.contact.email" :term="term" />
          </span>
        </div>
        <time
          :datetime="timestamp"
          :title="format(new Date(timestamp), 'MMM d, yyyy HH:mm')"
          class="text-xs text-muted-foreground whitespace-nowrap tabular-nums shrink-0"
        >
          {{ getRelativeTime(timestamp) }}
        </time>
      </div>

      <p v-if="subject" class="mt-1 truncate text-base font-medium leading-snug text-foreground">
        <HighlightedText :text="subject" :term="term" />
      </p>

      <p
        v-if="snippet"
        class="mt-1 text-sm leading-relaxed text-muted-foreground break-words"
        :class="isConversation ? 'truncate' : 'line-clamp-2'"
      >
        <template v-if="!isConversation">
          <span class="text-foreground">{{ senderName }}</span>
          <span>: </span>
        </template>
        <HighlightedText :text="snippet" :term="term" />
      </p>

      <div
        class="mt-2.5 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs text-muted-foreground"
      >
        <Badge v-if="status" variant="outline" class="font-normal">{{ status }}</Badge>
        <span v-if="item.priority" class="inline-flex items-center gap-1">
          <PriorityMarker :priority="item.priority" />
          {{ item.priority }}
        </span>
        <span class="tabular-nums">#{{ referenceNumber }}</span>
        <span v-if="item.inbox_name" class="inline-flex items-center gap-1.5 min-w-0">
          <component
            :is="channelIcon"
            :class="METADATA_ICON_CLASS"
            aria-hidden="true"
          />
          <span class="truncate">{{ item.inbox_name }}</span>
        </span>
        <span class="inline-flex items-center gap-1.5 min-w-0">
          <UserRound :class="METADATA_ICON_CLASS" aria-hidden="true" />
          <span class="truncate">{{ assigneeName || t('globals.terms.unassigned') }}</span>
        </span>
        <span v-if="item.team_name" class="inline-flex items-center gap-1.5 min-w-0">
          <UsersRound :class="METADATA_ICON_CLASS" aria-hidden="true" />
          <span class="truncate">{{ item.team_name }}</span>
        </span>
        <span v-if="item.tags?.length" class="inline-flex items-center gap-1.5 flex-wrap">
          <Tag :class="METADATA_ICON_CLASS" aria-hidden="true" />
          <span
            v-for="tag in item.tags"
            :key="tag"
            class="rounded-md bg-secondary px-1.5 py-0.5 text-secondary-foreground"
          >
            {{ tag }}
          </span>
        </span>
      </div>
    </div>
  </router-link>
</template>

<script setup>
import { computed } from 'vue'
import { format } from 'date-fns'
import { Mail, Tag, UserRound, UsersRound } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Badge } from '@shared-ui/components/ui/badge'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import PriorityMarker from '@main/features/conversation/PriorityMarker.vue'
import { CHANNEL_ICONS } from '@main/constants/channelIcons.js'
import HighlightedText from './HighlightedText.vue'

const METADATA_ICON_CLASS = 'w-3.5 h-3.5 shrink-0'

const props = defineProps({
  item: { type: Object, required: true },
  type: { type: String, required: true },
  term: { type: String, default: '' }
})

const { t } = useI18n()

const fullName = (person) => [person?.first_name, person?.last_name].filter(Boolean).join(' ')
const initials = (name) => (name || '?').substring(0, 2).toUpperCase()

const isConversation = computed(() => props.type === 'conversations')
const conversationUUID = computed(() =>
  isConversation.value ? props.item.uuid : props.item.conversation_uuid
)
const referenceNumber = computed(() =>
  isConversation.value ? props.item.reference_number : props.item.conversation_reference_number
)
const status = computed(() =>
  isConversation.value ? props.item.status : props.item.conversation_status
)
const subject = computed(
  () => (isConversation.value ? props.item.subject : props.item.conversation_subject) || ''
)
const snippet = computed(
  () => (isConversation.value ? props.item.last_message : props.item.snippet) || ''
)
const timestamp = computed(() =>
  isConversation.value ? props.item.last_message_at || props.item.created_at : props.item.created_at
)
const channelIcon = computed(() => CHANNEL_ICONS[props.item.inbox_channel] || Mail)
const contactName = computed(() => fullName(props.item.contact))
const senderName = computed(() => fullName(props.item.sender) || contactName.value)
const assigneeName = computed(() => fullName(props.item.assignee))
</script>
