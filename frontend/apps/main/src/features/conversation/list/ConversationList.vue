<template>
  <div class="h-full flex flex-col">
    <!-- Header -->
    <div class="flex min-w-0 items-center gap-2 px-2 h-12 border-b shrink-0">
      <SidebarTrigger class="cursor-pointer" />
      <span class="min-w-0 flex-1 truncate text-xl font-semibold" :title="title">{{ title }}</span>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button type="button" variant="ghost" size="icon" :aria-label="t('address.actions')">
            <MoreHorizontal class="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem :disabled="markingRead" @select="markAllAsRead">
            <MailCheck class="mr-2 h-4 w-4" />{{ t('address.markAllAsRead') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <!-- Bulk Action Toolbar (when items selected) -->
    <ConversationBulkActionToolbar v-if="hasSelection && canBulkAct" />

    <!-- Filters (hidden when bulk selecting) -->
    <div v-else class="p-2 flex justify-between items-center">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" class="w-30">
            <div>
              <span class="mr-1">{{ conversationStore.conversations.total }}</span>
              <span>{{ conversationStore.getListStatus }}</span>
            </div>
            <ChevronDown class="w-4 h-4 ml-2 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem
            v-for="status in conversationStore.statusOptions"
            :key="status.value"
            @click="handleStatusChange(status)"
          >
            {{ status.label }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <!-- Sort dropdown-menu -->
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" class="w-30">
            {{ conversationStore.getListSortField }}
            <ChevronDown class="w-4 h-4 ml-2 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem @click="handleSortChange('oldest')">
            {{ $t('conversation.sort.oldestActivity') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('newest')">
            {{ $t('conversation.sort.newestActivity') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_first')">
            {{ $t('conversation.sort.startedFirst') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('started_last')">
            {{ $t('conversation.sort.startedLast') }}
          </DropdownMenuItem>
          <DropdownMenuItem @click="handleSortChange('waiting_longest')">
            {{ $t('conversation.sort.waitingLongest') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <!-- Content -->
    <div class="relative min-h-0 flex-grow overflow-hidden">
      <div
        v-if="pullDistance > 0"
        class="pointer-events-none absolute inset-x-0 top-0 z-10 flex h-14 items-center justify-center gap-2 text-sm font-medium text-muted-foreground"
        :class="{ 'transition-transform duration-200 ease-out': !isPulling }"
        :style="indicatorStyle"
        role="status"
        aria-live="polite"
      >
        <Loader2 v-if="refreshing" class="h-4 w-4 animate-spin text-primary" />
        <RefreshCw v-else class="h-4 w-4 text-primary" />
        <span>{{ indicatorText }}</span>
      </div>

      <div
        ref="scrollElement"
        class="h-full overflow-y-auto overscroll-contain"
        @touchstart.passive="startPull"
        @touchmove="movePull"
        @touchend="finishPull"
        @touchcancel="resetPull"
      >
        <div
          :class="{ 'transition-transform duration-200 ease-out': !isPulling }"
          :style="contentStyle"
        >
          <EmptyList
            v-if="showEmpty"
            key="empty"
            class="px-4 py-8"
            :title="t('conversation.noConversationsFound')"
            :message="t('conversation.tryAdjustingFilters')"
            :icon="Inbox"
          />

          <EmptyList
            v-if="hasErrored"
            key="error"
            class="px-4 py-8"
            :title="t('conversation.couldNotFetch')"
            :message="conversationStore.conversations.errorMessage"
            :icon="MessageCircleWarning"
          />

          <TransitionGroup
            enter-active-class="transition-all duration-300 ease-in-out"
            enter-from-class="opacity-0 transform translate-y-4"
            enter-to-class="opacity-100 transform translate-y-0"
            leave-active-class="transition-all duration-300 ease-in-out"
            leave-from-class="opacity-100 transform translate-y-0"
            leave-to-class="opacity-0 transform translate-y-4"
          >
            <div
              v-if="!hasErrored && !conversationStore.conversations.loading"
              key="list"
              class="divide-y divide-border"
              :class="{ 'border-b border-border': hasConversations }"
            >
              <ConversationListItem
                v-for="conversation in conversationStore.conversationsList"
                :key="conversation.uuid"
                :conversation="conversation"
                :currentConversation="conversationStore.current"
                :contactFullName="conversationStore.getContactFullName(conversation.uuid)"
                class="transition-colors duration-200"
              />
            </div>

            <div v-if="conversationStore.conversations.loading" key="loading">
              <ConversationListItemSkeleton v-for="i in 12" :key="i" :index="i - 1" />
            </div>
          </TransitionGroup>

          <!-- Load More -->
          <div
            v-if="!hasErrored && (conversationStore.conversations.hasMore || hasConversations)"
            class="flex justify-center items-center p-5"
          >
            <Button
              v-if="conversationStore.conversations.hasMore"
              variant="outline"
              @click="conversationStore.fetchNextConversations"
              :disabled="conversationStore.conversations.fetching"
              class="max-md:h-11 transition-all duration-200 ease-in-out transform hover:scale-105"
            >
              <Loader2
                v-if="conversationStore.conversations.fetching"
                class="mr-2 h-4 w-4 animate-spin"
              />
              {{
                conversationStore.conversations.fetching
                  ? t('globals.terms.loading')
                  : t('globals.terms.loadMore')
              }}
            </Button>
            <p
              class="text-sm text-muted-foreground"
              v-else-if="conversationStore.conversationsList.length > 10"
            >
              {{ $t('conversation.allLoaded') }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Inbox, MessageCircleWarning, ChevronDown, Loader2, RefreshCw, MoreHorizontal, MailCheck } from 'lucide-vue-next'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useAddressStore } from '@/stores/address'
import { useConversationStore } from '@/stores/conversation'
import { useBulkActionPermissions } from '@/composables/useBulkActionPermissions'
import { usePullToRefresh } from '@/composables/usePullToRefresh'
import EmptyList from '@/features/conversation/list/ConversationEmptyList.vue'
import ConversationBulkActionToolbar from '@/features/conversation/list/ConversationBulkActionToolbar.vue'
import ConversationListItem from '@/features/conversation/list/ConversationListItem.vue'
import ConversationListItemSkeleton from '@/features/conversation/list/ConversationListItemSkeleton.vue'

const conversationStore = useConversationStore()
const addressStore = useAddressStore()
const { canBulkAct } = useBulkActionPermissions()
const route = useRoute()
const { t } = useI18n()
const emitter = useEmitter()
const markingRead = ref(false)

async function markAllAsRead() {
  const addressID = Number(route.params.addressID)
  if (!addressID || markingRead.value) return
  const addressName = addressStore.get(addressID)?.address || title.value
  markingRead.value = true
  try {
    await conversationStore.markAddressAsRead(addressID)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('address.markedAllAsRead', { address: addressName })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive', description: handleHTTPError(error).message
    })
  } finally {
    markingRead.value = false
  }
}

const hasSelection = computed(() => conversationStore.selectedCount > 0)

async function refreshCurrentAddress() {
  const addressID = Number(route.params.addressID)
  if (!addressID) return

  await Promise.all([
    addressStore.fetchAddresses(true),
    conversationStore.fetchConversationsList(false, addressID, 1),
    conversationStore.fetchSidebarCounts({ force: true })
  ])
}

const {
  contentStyle,
  finishPull,
  indicatorStyle,
  indicatorText,
  isPulling,
  movePull,
  pullDistance,
  refreshing,
  resetPull,
  scrollElement,
  startPull
} = usePullToRefresh(refreshCurrentAddress)

const title = computed(() => {
  const address = addressStore.get(route.params.addressID)
  return address?.address || 'Addresses'
})

const handleStatusChange = (status) => {
  conversationStore.setListStatus(status.label)
}

const handleSortChange = (order) => {
  conversationStore.setListSortField(order)
}

const hasConversations = computed(() => conversationStore.conversationsList.length !== 0)
const hasErrored = computed(() => !!conversationStore.conversations.errorMessage)
const showEmpty = computed(
  () =>
    !hasConversations.value &&
    !hasErrored.value &&
    !conversationStore.conversations.loading &&
    conversationStore.conversations.initialized
)
</script>
