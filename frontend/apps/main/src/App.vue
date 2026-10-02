<template>
  <div class="flex h-dvh w-full bg-canvas p-1 text-foreground md:p-1.5">
    <div class="min-w-0 flex-1">
      <Sidebar>
        <div class="flex h-full flex-col overflow-hidden rounded-lg bg-background">
          <ConnectionBanner />
          <AdminBanner v-if="route.path.startsWith('/admin')" />
          <PageHeader />
          <RouterView class="flex-grow" />
        </div>
      </Sidebar>
    </div>
  </div>

  <Command />
  <ComposeDialog />
</template>

<script setup>
import { onMounted, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { useEventListener } from '@vueuse/core'
import { toast as sooner } from 'vue-sonner'
import { retireNotificationWorker } from './composables/retireNotificationWorker'
import { useAddressTitle } from './composables/useAddressTitle'
import { useIdleDetection } from './composables/useIdleDetection'
import { useEmitter } from './composables/useEmitter'
import { useUserStore } from './stores/user'
import { useUsersStore } from './stores/users'
import { useAddressStore } from './stores/address'
import { useConversationStore } from './stores/conversation'
import { initWS } from './websocket.js'
import { EMITTER_EVENTS } from './constants/emitterEvents.js'
import PageHeader from './components/layout/PageHeader.vue'
import AdminBanner from '@/components/banner/AdminBanner.vue'
import ConnectionBanner from '@/components/banner/ConnectionBanner.vue'
import Sidebar from '@main/components/sidebar/Sidebar.vue'
import Command from '@/features/command/CommandBox.vue'
import ComposeDialog from '@/features/compose/ComposeDialog.vue'

const route = useRoute()
const emitter = useEmitter()
const userStore = useUserStore()
const usersStore = useUsersStore()
const addressStore = useAddressStore()
const conversationStore = useConversationStore()
watch(
  () => route.path.replace(/\/conversation\/.*$/, ''),
  (path) => {
    if (path.startsWith('/addresses')) conversationStore.fetchSidebarCounts()
  }
)

initWS()
useIdleDetection()
useAddressTitle()
useEventListener(window, 'focus', () => conversationStore.resyncMail())
useEventListener(window, 'online', () => conversationStore.retryDraftSave())

onMounted(async () => {
  retireNotificationWorker()
  initToaster()
  if (!userStore.userID) await userStore.getCurrentUser()
  await Promise.allSettled([
    addressStore.fetchAddresses(),
    conversationStore.fetchStatuses(),
    conversationStore.fetchAllDrafts(),
    usersStore.fetchUsers()
  ])
})

function initToaster() {
  emitter.on(EMITTER_EVENTS.SHOW_TOAST, (message) => {
    if (!message.description) return
    if (message.variant === 'destructive') sooner.error(message.description)
    else if (message.variant === 'warning') sooner.warning(message.description)
    else if (message.variant === 'info') sooner.info(message.description)
    else sooner.success(message.description)
  })
}
</script>

<style scoped>
:deep(.group\/sidebar-wrapper) {
  min-height: auto !important;
  height: 100%;
}
</style>
