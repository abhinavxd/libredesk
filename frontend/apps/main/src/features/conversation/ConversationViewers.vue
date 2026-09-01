<template>
  <div
    v-if="viewers.length"
    class="conversation-viewers flex items-center"
    :aria-label="t('zendesk.viewersAria', viewers.length, { count: viewers.length })"
  >
    <div class="flex items-center">
      <Tooltip v-for="(viewer, index) in visibleViewers" :key="viewer.id">
        <TooltipTrigger as-child>
          <Avatar
            class="conversation-viewer-avatar size-7 border-2 border-background bg-muted"
            :style="{ zIndex: visibleViewers.length - index, marginLeft: index ? '-0.45rem' : '0' }"
          >
            <AvatarImage :src="viewer.avatar_url" :alt="viewer.name" />
            <AvatarFallback class="text-[10px] font-medium">
              {{ viewer.initials }}
            </AvatarFallback>
          </Avatar>
        </TooltipTrigger>
        <TooltipContent>
          {{ t('zendesk.viewingTicket', { name: viewer.name }) }}
        </TooltipContent>
      </Tooltip>
      <Tooltip v-if="overflowCount">
        <TooltipTrigger as-child>
          <div
            class="conversation-viewer-avatar conversation-viewer-more size-7 border-2 border-background"
            :style="{ zIndex: 0, marginLeft: '-0.45rem' }"
          >
            +{{ overflowCount }}
          </div>
        </TooltipTrigger>
        <TooltipContent>
          {{ overflowNames }}
        </TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useConversationStore } from '@main/stores/conversation'
import { useUsersStore } from '@main/stores/users'

const MAX_VISIBLE = 3

const { t } = useI18n()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()

const viewers = computed(() =>
  conversationStore.otherViewersOnCurrent.map((id) => {
    const user = usersStore.users.find((u) => Number(u.id) === Number(id))
    const first = user?.first_name || ''
    const last = user?.last_name || ''
    return {
      id,
      name: `${first} ${last}`.trim() || t('zendesk.viewerFallback', { id }),
      avatar_url: user?.avatar_url || '',
      initials: `${first.charAt(0) || 'A'}${last.charAt(0) || ''}`.toUpperCase()
    }
  })
)

const visibleViewers = computed(() => viewers.value.slice(0, MAX_VISIBLE))
const overflowCount = computed(() => Math.max(0, viewers.value.length - MAX_VISIBLE))
const overflowNames = computed(() =>
  viewers.value
    .slice(MAX_VISIBLE)
    .map((v) => v.name)
    .join(', ')
)
</script>

<style scoped>
.conversation-viewer-avatar {
  box-shadow: 0 0 0 2px hsl(142 71% 45%);
}

.conversation-viewer-more {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  background: hsl(var(--muted));
  font-size: 10px;
  font-weight: 600;
  color: hsl(var(--muted-foreground));
}
</style>
