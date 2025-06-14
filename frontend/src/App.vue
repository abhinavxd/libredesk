<template>
  <div class="flex w-full h-screen text-foreground">
    <!-- Icon sidebar always visible -->
    <SidebarProvider style="--sidebar-width: 3rem" class="w-auto z-50">
      <ShadcnSidebar collapsible="none" class="border-r">
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <Tooltip>
                    <TooltipTrigger as-child>
                      <SidebarMenuButton asChild :isActive="route.path.startsWith('/inboxes')">
                        <router-link :to="{ name: 'inboxes' }">
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
          <SidebarNavUser />
        </SidebarFooter>
      </ShadcnSidebar>
    </SidebarProvider>

    <!-- Main sidebar that collapses -->
    <div class="flex-1">
      <Sidebar
        :userTeams="userStore.teams"
        :userViews="userViews"
        @create-view="openCreateViewForm = true"
        @edit-view="editView"
        @delete-view="deleteView"
        @create-conversation="() => (openCreateConversationDialog = true)"
      >
        <div class="flex flex-col h-screen">
          <!-- Show app update only in admin routes -->
          <AppUpdate v-if="route.path.startsWith('/admin')" />

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
import { onMounted, ref } from 'vue'
import { RouterView } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { initWS } from '@/websocket.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { useConversationStore } from './stores/conversation'
import { useInboxStore } from '@/stores/inbox'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { useSlaStore } from '@/stores/sla'
import { useMacroStore } from '@/stores/macro'
import { useTagStore } from '@/stores/tag'
import { useCustomAttributeStore } from '@/stores/customAttributes'
import { useIdleDetection } from '@/composables/useIdleDetection'
import PageHeader from './components/layout/PageHeader.vue'
import ViewForm from '@/features/view/ViewForm.vue'
import AppUpdate from '@/components/update/AppUpdate.vue'
import api from '@/api'
import { toast as sooner } from 'vue-sonner'
import Sidebar from '@/components/sidebar/Sidebar.vue'
import Command from '@/features/command/CommandBox.vue'
import CreateConversation from '@/features/conversation/CreateConversation.vue'
import { Inbox, Shield, FileLineChart, BookUser } from 'lucide-vue-next'
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
} from '@/components/ui/sidebar'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import SidebarNavUser from '@/components/sidebar/SidebarNavUser.vue'

const route = useRoute()
const emitter = useEmitter()
const userStore = useUserStore()
const conversationStore = useConversationStore()
const usersStore = useUsersStore()
const teamStore = useTeamStore()
const inboxStore = useInboxStore()
const slaStore = useSlaStore()
const macroStore = useMacroStore()
const tagStore = useTagStore()
const customAttributeStore = useCustomAttributeStore()
const userViews = ref([])
const view = ref({})
const openCreateViewForm = ref(false)
const openCreateConversationDialog = ref(false)
const { t } = useI18n()

initWS()
useIdleDetection()

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
    conversationStore.fetchStatuses(),
    conversationStore.fetchPriorities(),
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
      description: t('globals.messages.deletedSuccessfully', {
        name: t('globals.terms.view')
      })
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
