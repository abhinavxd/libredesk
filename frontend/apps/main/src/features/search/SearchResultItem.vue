<template>
  <router-link
    :to="{ name: 'inbox-conversation', params: { uuid: conversationUUID, type: 'assigned' } }"
    class="grid grid-cols-[auto_minmax(0,1fr)_auto] gap-3 px-4 py-3 hover:bg-accent/40 transition-colors"
  >
    <Avatar class="w-9 h-9 rounded-full mt-0.5">
      <AvatarImage :src="item.contact.avatar_url || ''" class="object-cover" />
      <AvatarFallback>{{ initials(item.contact.first_name) }}</AvatarFallback>
    </Avatar>

    <div class="min-w-0 space-y-1">
      <div class="flex items-baseline gap-2 text-sm min-w-0">
        <span class="font-medium text-foreground truncate">
          <HighlightedText :text="contactName" :term="term" />
        </span>
        <span v-if="item.contact.email" class="text-muted-foreground truncate">
          <HighlightedText :text="item.contact.email" :term="term" />
        </span>
      </div>

      <p class="text-sm truncate" :class="subject ? 'text-foreground font-medium' : 'text-muted-foreground italic'">
        <HighlightedText v-if="subject" :text="subject" :term="term" />
        <template v-else>{{ t('globals.terms.noSubject') }}</template>
      </p>

      <p v-if="snippet" class="text-sm text-muted-foreground line-clamp-2 break-words">
        <template v-if="!isConversation">
          <span class="text-foreground">{{ senderName }}</span>
          <Badge v-if="item.private" variant="secondary" class="mx-1.5 text-xs font-normal">
            {{ t('globals.terms.privateNote') }}
          </Badge>
          <Badge v-else-if="item.type === 'outgoing'" variant="outline" class="mx-1.5 text-xs font-normal">
            {{ t('globals.terms.reply') }}
          </Badge>
          <span v-else>: </span>
        </template>
        <HighlightedText :text="snippet" :term="term" />
      </p>

      <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground pt-0.5">
        <span class="tabular-nums">#{{ referenceNumber }}</span>
        <span v-if="item.inbox_name" class="inline-flex items-center gap-1 min-w-0">
          <component :is="item.inbox_channel === 'livechat' ? MessageSquare : Mail" class="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
          <span class="truncate">{{ item.inbox_name }}</span>
        </span>
        <span class="inline-flex items-center gap-1 min-w-0">
          <UserRound class="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
          <span class="truncate">{{ assigneeName || t('globals.terms.unassigned') }}</span>
        </span>
        <span v-if="item.team_name" class="inline-flex items-center gap-1 min-w-0">
          <UsersRound class="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
          <span class="truncate">{{ item.team_name }}</span>
        </span>
        <span v-if="item.tags?.length" class="inline-flex items-center gap-1 flex-wrap">
          <Tag class="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
          <span v-for="tag in item.tags" :key="tag" class="rounded-md bg-secondary px-1.5 py-0.5 text-secondary-foreground">
            {{ tag }}
          </span>
        </span>
      </div>
    </div>

    <div class="flex flex-col items-end gap-1.5 shrink-0">
      <time
        :datetime="timestamp"
        :title="format(new Date(timestamp), 'MMM d, yyyy HH:mm')"
        class="text-xs text-muted-foreground whitespace-nowrap tabular-nums"
      >
        {{ getRelativeTime(timestamp) }}
      </time>
      <Badge v-if="status" variant="outline" class="text-xs font-normal">{{ status }}</Badge>
      <span v-if="item.priority" class="inline-flex items-center gap-1 text-xs text-muted-foreground">
        <PriorityMarker :priority="item.priority" />
        {{ item.priority }}
      </span>
    </div>
  </router-link>
</template>

<script setup>
import { computed } from 'vue'
import { format } from 'date-fns'
import { Mail, MessageSquare, Tag, UserRound, UsersRound } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Badge } from '@shared-ui/components/ui/badge'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import PriorityMarker from '@main/features/conversation/PriorityMarker.vue'
import HighlightedText from './HighlightedText.vue'

const props = defineProps({
  item: { type: Object, required: true },
  type: { type: String, required: true },
  term: { type: String, default: '' }
})

const { t } = useI18n()

const fullName = (person) => [person?.first_name, person?.last_name].filter(Boolean).join(' ')
const initials = (name) => (name || '?').substring(0, 2).toUpperCase()

const isConversation = computed(() => props.type === 'conversations')
const conversationUUID = computed(() => (isConversation.value ? props.item.uuid : props.item.conversation_uuid))
const referenceNumber = computed(() =>
  isConversation.value ? props.item.reference_number : props.item.conversation_reference_number
)
const status = computed(() => (isConversation.value ? props.item.status : props.item.conversation_status))
const subject = computed(() => (isConversation.value ? props.item.subject : props.item.conversation_subject) || '')
const snippet = computed(() => (isConversation.value ? props.item.last_message : props.item.snippet) || '')
const timestamp = computed(() =>
  isConversation.value ? props.item.last_message_at || props.item.created_at : props.item.created_at
)
const contactName = computed(() => fullName(props.item.contact))
const senderName = computed(() => fullName(props.item.sender) || contactName.value)
const assigneeName = computed(() => fullName(props.item.assignee))
</script>
