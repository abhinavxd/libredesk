<template>
  <SmallScreenOverlay v-if="showSmallScreenOverlay" @dismiss="dismissSmallScreen" />

  <div class="flex w-full h-screen text-foreground bg-canvas p-1.5">
    <!-- Icon sidebar always visible -->
    <SidebarProvider style="--sidebar-width: 3rem" class="w-auto z-50">
      <ShadcnSidebar collapsible="none" class="border rounded-lg overflow-hidden">
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <SidebarMenuButton asChild :isActive="route.path.startsWith('/inboxes')">
                        <router-link :to="lastInboxPath || { name: 'inboxes' }">
                          <Inbox />
                        </router-link>
                      </SidebarMenuButton>
                    </TooltipTrigger>
                    <TooltipContent side="right">
                      <p>{{ t('globals.terms.inbox', 2) }}</p>
                    </TooltipContent>
                  </Tooltip>
                </SidebarMenuItem>
                <SidebarMenuItem v-if="userStore.can('contacts:read_all')">
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <SidebarMenuButton asChild :isActive="route.path.startsWith('/contacts')">
                        <router-link :to="{ name: 'contacts' }">
                          <BookUser />
                        </router-link>
                      </SidebarMenuButton>
                    </TooltipTrigger>
                    <TooltipContent side="right">
                      <p>{{ t('globals.terms.contact', 2) }}</p>
                    </TooltipContent>
                  </Tooltip>
                </SidebarMenuItem>
                <SidebarMenuItem v-if="userStore.hasReportTabPermissions">
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <SidebarMenuButton asChild :isActive="route.path.startsWith('/reports')">
                        <router-link :to="{ name: 'reports' }">
                          <FileLineChart />
                        </router-link>
                      </SidebarMenuButton>
                    </TooltipTrigger>
                    <TooltipContent side="right">
                      <p>{{ t('globals.terms.report', 2) }}</p>
                    </TooltipContent>
                  </Tooltip>
                </SidebarMenuItem>
                <SidebarMenuItem v-if="userStore.hasAdminTabPermissions">
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <SidebarMenuButton asChild :isActive="route.path.startsWith('/admin')">
                        <router-link
                          :to="{
                            name: userStore.can('general_settings:manage') ? 'general' : 'admin'
                          }"
                        >
                          <Shield />
                        </router-link>
                      </SidebarMenuButton>
                    </TooltipTrigger>
                    <TooltipContent side="right">
                      <p>{{ t('globals.terms.admin') }}</p>
                    </TooltipContent>
                  </Tooltip>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter>
          <SidebarMenu>
            <SidebarMenuItem>
              <Tooltip>
                <TooltipTrigger as-child>
                  <NotificationBell />
                </TooltipTrigger>
                <TooltipContent side="right">
                  <p>{{ t('globals.terms.notification', 2) }}</p>
                </TooltipContent>
              </Tooltip>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <ThemeSwitcher />
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarNavUser />
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>
      </ShadcnSidebar>
    </SidebarProvider>

    <!-- Main sidebar that collapses -->
    <div class="flex-1 min-w-0">
      <Sidebar
        :userTeams="userStore.teams"
        :userViews="userViews"
        :sharedViews="sharedViewStore.sharedViewList"
        @create-view="openCreateViewForm = true"
        @edit-view="editView"
        @delete-view="deleteView"
        @create-conversation="() => (openCreateConversationDialog = true)"
      >
        <div class="flex flex-col h-full rounded-lg overflow-hidden bg-background">
          <!-- Show admin banner only in admin routes -->
          <AdminBanner v-if="route.path.startsWith('/admin')" />

          <!-- Common header for all pages -->
          <PageHeader />

          <!-- Main content -->
          <RouterView class="flex-grow" />
        </div>
        <ViewForm v-model:openDialog="openCreateViewForm" v-model:view="view" />
      </Sidebar>
    </div>
  </div>

  <!-- Command box -->
  <Command />

  <!-- Create conversation dialog -->
  <CreateConversation v-model="openCreateConversationDialog" v-if="openCreateConversationDialog" />
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { useStorage } from '@vueuse/core'
import { RouterView } from 'vue-router'
import { useUserStore } from './stores/user'
import { initWS } from './websocket.js'
import { EMITTER_EVENTS } from './constants/emitterEvents.js'
import { useEmitter } from './composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useConversationStore } from './stores/conversation'
import { useInboxStore } from './stores/inbox'
import { useUsersStore } from './stores/users'
import { useTeamStore } from './stores/team'
import { useSlaStore } from './stores/sla'
import { useMacroStore } from './stores/macro'
import { useSharedViewStore } from './stores/sharedView'
import { useTagStore } from './stores/tag'
import { useCustomAttributeStore } from './stores/customAttributes'
import { useIdleDetection } from './composables/useIdleDetection'
import { useNotificationStore } from './stores/notification'
import { initAudioContext } from '@shared-ui/composables/useNotificationSound'
import PageHeader from './components/layout/PageHeader.vue'
import ViewForm from '@/features/view/ViewForm.vue'
import AdminBanner from '@/components/banner/AdminBanner.vue'
import { toast as sooner } from 'vue-sonner'
import Sidebar from '@main/components/sidebar/Sidebar.vue'
import Command from '@/features/command/CommandBox.vue'
import CreateConversation from '@/features/conversation/CreateConversation.vue'
import { Inbox, Shield, FileLineChart, BookUser } from 'lucide-vue-next'
import SmallScreenOverlay from '@/components/SmallScreenOverlay.vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import {
  Sidebar as ShadcnSidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarMenu,
  SidebarGroupContent,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider
} from '@shared-ui/components/ui/sidebar'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import SidebarNavUser from '@main/components/sidebar/SidebarNavUser.vue'
import NotificationBell from '@main/components/sidebar/NotificationBell.vue'
import ThemeSwitcher from '@main/components/sidebar/ThemeSwitcher.vue'
import { useTheme } from '@main/composables/useTheme'
import api from '@main/api'

const route = useRoute()
const emitter = useEmitter()

// Apply the persisted theme to <html data-theme="..."> on app start. Called
// here (not just inside ThemeSwitcher) so the attribute is set even when
// only one theme is registered and the switcher UI is hidden.
useTheme()

// Small screen overlay - shown once per session for screens < 768px.
const showSmallScreenOverlay = ref(window.screen.width < 768 && !sessionStorage.getItem('smallScreenDismissed'))
function dismissSmallScreen() {
  sessionStorage.setItem('smallScreenDismissed', '1')
  showSmallScreenOverlay.value = false
}

// Remember last inbox path so navigating back from admin/contacts/reports restores it
const lastInboxPath = useStorage('lastInboxPath', '')
watch(
  () => route.path,
  (path) => {
    if (path.startsWith('/inboxes') && path !== '/inboxes/search') {
      lastInboxPath.value = path.replace(/\/conversation\/[^/]+$/, '')
    }
  }
)
const userStore = useUserStore()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamStore = useTeamStore()
const inboxStore = useInboxStore()
const slaStore = useSlaStore()
const macroStore = useMacroStore()
const sharedViewStore = useSharedViewStore()
const tagStore = useTagStore()
const customAttributeStore = useCustomAttributeStore()
const userViews = ref([])
const view = ref({})
const openCreateViewForm = ref(false)
const openCreateConversationDialog = ref(false)
const { t } = useI18n()
const notificationStore = useNotificationStore()

// Update browser tab title with unread notification count.
// Watch both unreadCount and route so the prefix is preserved after navigation.
watch([() => notificationStore.unreadCount, () => route.fullPath], ([count]) => {
  const base = document.title.replace(/^\(\d+\)\s*/, '')
  document.title = count > 0 ? `(${count}) ${base}` : base
})

initWS()
useIdleDetection()

// Unlock audio on first user interaction (browser autoplay policy)
const unlockAudio = () => {
  initAudioContext()
  document.removeEventListener('click', unlockAudio)
  document.removeEventListener('touchstart', unlockAudio)
}
document.addEventListener('click', unlockAudio)
document.addEventListener('touchstart', unlockAudio)

onMounted(() => {
  initToaster()
  listenViewRefresh()
  initStores()
})

// Initialize data stores
const initStores = async () => {
  if (!userStore.userID) {
    await userStore.getCurrentUser()
  }
  await Promise.allSettled([
    getUserViews(),
    sharedViewStore.loadSharedViews(),
    conversationStore.fetchStatuses(),
    conversationStore.fetchPriorities(),
    conversationStore.fetchAllDrafts(),
    usersStore.fetchUsers(),
    teamStore.fetchTeams(),
    inboxStore.fetchInboxes(),
    slaStore.fetchSlas(),
    macroStore.loadMacros(),
    tagStore.fetchTags(),
    customAttributeStore.fetchCustomAttributes()
  ])
}

const editView = (v) => {
  view.value = { ...v }
  openCreateViewForm.value = true
}

const deleteView = async (view) => {
  try {
    await api.deleteView(view.id)
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'view' })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.deletedSuccessfully')
    })
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  }
}

const getUserViews = async () => {
  try {
    const response = await api.getCurrentUserViews()
    userViews.value = response.data.data
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  }
}

const initToaster = () => {
  emitter.on(EMITTER_EVENTS.SHOW_TOAST, (message) => {
    if (message.variant === 'destructive') {
      sooner.error(message.description)
    } else if (message.variant === 'warning') {
      sooner.warning(message.description)
    } else {
      sooner.success(message.description)
    }
  })
}

const listenViewRefresh = () => {
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshViews)
}

const refreshViews = (data) => {
  openCreateViewForm.value = false
  // TODO: move model to constants.
  if (data?.model === 'view') {
    getUserViews()
  }
}
</script>

<style scoped>
:deep(.group\/sidebar-wrapper) {
  min-height: auto !important;
  height: 100%;
}
</style>
