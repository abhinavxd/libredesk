<script setup>
import {
  adminNavItems,
  reportsNavItems,
  accountNavItems,
  contactNavItems
} from '../../constants/navigation'
import { useRoute } from 'vue-router'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@shared-ui/components/ui/collapsible'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
  SidebarProvider
} from '@shared-ui/components/ui/sidebar'
import { useAppSettingsStore } from '@main/stores/appSettings'
import {
  ChevronRight,
  EllipsisVertical,
  User,
  Search,
  Plus,
  CircleDashed,
  List,
  AtSign,
} from 'lucide-vue-next'

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import MobileDrawerNav from './MobileDrawerNav.vue'
import MobileDrawerFooter from './MobileDrawerFooter.vue'
import SidebarCountBadge from './SidebarCountBadge.vue'
import { filterNavItems } from '@main/utils/nav-permissions'
import { permissions } from '@main/constants/permissions'
import { useStorage } from '@vueuse/core'
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@main/stores/user'
import { useConversationStore } from '@main/stores/conversation'
import { navIconMap } from '@main/constants/navIcons'
import { useInboxNavigation } from '@main/composables/useInboxNavigation'

defineProps({
  userTeams: { type: Array, default: () => [] },
  userViews: { type: Array, default: () => [] },
  sharedViews: { type: Array, default: () => [] }
})
const userStore = useUserStore()
const conversationStore = useConversationStore()
const settingsStore = useAppSettingsStore()
const route = useRoute()
const { t } = useI18n()
const emit = defineEmits(['createView', 'editView', 'deleteView', 'createConversation'])

const isActiveParent = (parentHref) => {
  return route.path.startsWith(parentHref)
}

const isInboxRoute = (path) => {
  return path.startsWith('/inboxes')
}

const openCreateViewDialog = () => {
  emit('createView')
}

const editView = (view) => {
  emit('editView', view)
}

const openDeleteConfirmation = (view) => {
  viewToDelete.value = view
  isDeleteOpen.value = true
}

const handleDeleteView = () => {
  if (viewToDelete.value) {
    emit('deleteView', viewToDelete.value)
    isDeleteOpen.value = false
    viewToDelete.value = null
  }
}

const { navigateToInbox, navigateToTeamInbox, navigateToViewInbox } = useInboxNavigation()

const filteredAdminNavItems = computed(() => filterNavItems(adminNavItems, userStore.can))
const filteredReportsNavItems = computed(() => filterNavItems(reportsNavItems, userStore.can))
const filteredContactsNavItems = computed(() => filterNavItems(contactNavItems, userStore.can))

// For auto opening admin collapsibles when a child route is active
const openAdminCollapsible = ref(null)
const toggleAdminCollapsible = (titleKey) => {
  openAdminCollapsible.value = openAdminCollapsible.value === titleKey ? null : titleKey
}
// Watch for route changes and update the active collapsible
watch(
  [() => route.path, filteredAdminNavItems],
  () => {
    const activeItem = filteredAdminNavItems.value.find((item) => {
      if (!item.children) return isActiveParent(item.href)
      return item.children.some((child) => isActiveParent(child.href))
    })
    if (activeItem) {
      openAdminCollapsible.value = activeItem.titleKey
    }
  },
  { immediate: true }
)

// Sidebar open state in local storage
const sidebarOpen = useStorage('mainSidebarOpen', true)
const teamInboxOpen = useStorage('teamInboxOpen', true)
const viewInboxOpen = useStorage('viewInboxOpen', true)
const sharedViewInboxOpen = useStorage('sharedViewInboxOpen', true)

// Track delete confirmation dialog state
const isDeleteOpen = ref(false)
const viewToDelete = ref(null)

const viewSidebarCount = (viewID) => {
  return conversationStore.sidebarCounts.views?.[viewID] ?? 0
}

onMounted(() => {
  conversationStore.fetchSidebarCounts({ force: true })
})
</script>

<template>
  <SidebarProvider
    style="--sidebar-width: 14rem"
    :default-open="sidebarOpen"
    v-on:update:open="sidebarOpen = $event"
  >
    <!-- Contacts sidebar -->
    <template
      v-if="route.matched.some((record) => record.name && record.name.startsWith('contact'))"
    >
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="px-1">
                <span class="font-semibold text-xl">
                  {{ t('globals.terms.contact', 2) }}
                </span>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>
        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredContactsNavItems" :key="item.titleKey">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <component :is="navIconMap[item.icon]" v-if="item.icon" />
                    <span>{{ t(item.allLabelKey) }}</span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <!-- Reports sidebar -->
    <template
      v-if="
        userStore.hasReportTabPermissions &&
        route.matched.some((record) => record.name && record.name.startsWith('reports'))
      "
    >
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="px-1">
                <span class="font-semibold text-xl">
                  {{ t('globals.terms.report', 2) }}
                </span>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>
        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredReportsNavItems" :key="item.titleKey">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <component :is="navIconMap[item.icon]" v-if="item.icon" />
                    <span>{{ t(item.titleKey) }}</span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <!-- Admin Sidebar -->
    <template v-if="route.matched.some((record) => record.name && record.name.startsWith('admin'))">
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="flex flex-col items-start justify-between w-full px-1">
                <span class="font-semibold text-xl">
                  {{ t('globals.terms.admin') }}
                </span>
                <!-- App version -->
                <div class="text-xs text-muted-foreground">
                  ({{ settingsStore.settings['app.version'] }})
                </div>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>
        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredAdminNavItems" :key="item.titleKey">
                <SidebarMenuButton
                  v-if="!item.children"
                  :isActive="isActiveParent(item.href)"
                  asChild
                >
                  <router-link :to="item.href">
                    <span>{{ t(item.titleKey) }}</span>
                  </router-link>
                </SidebarMenuButton>

                <Collapsible
                  v-else
                  class="group/collapsible"
                  :open="openAdminCollapsible === item.titleKey"
                  @update:open="toggleAdminCollapsible(item.titleKey)"
                >
                  <CollapsibleTrigger as-child>
                    <SidebarMenuButton :isActive="isActiveParent(item.href)">
                      <span>{{ t(item.titleKey, item.isTitleKeyPlural === true ? 2 : 1) }}</span>
                      <Badge
                        v-if="item.badge"
                        variant="outline"
                        class="ml-1.5 rounded-full uppercase tracking-[0.07em] font-medium text-[9px] leading-none px-[5.5px] py-[3px] bg-warning/10 text-warning-600 border-warning/50 shrink-0"
                      >
                        {{ item.badge }}
                      </Badge>
                      <ChevronRight
                        class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                      />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <SidebarMenuSub>
                      <SidebarMenuSubItem v-for="child in item.children" :key="child.titleKey">
                        <SidebarMenuButton size="sm" :isActive="isActiveParent(child.href)" asChild>
                          <router-link :to="child.href">
                            <component :is="navIconMap[child.icon]" v-if="child.icon" />
                            <span>{{ t(child.titleKey, child.isTitleKeyPlural === true ? 2 : 1) }}</span>
                          </router-link>
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </Collapsible>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <!-- Account sidebar -->
    <template v-if="isActiveParent('/account')">
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="px-1">
                <span class="font-semibold text-xl">
                  {{ t('globals.terms.account') }}
                </span>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>
        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in accountNavItems" :key="item.titleKey">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <component :is="navIconMap[item.icon]" v-if="item.icon" />
                    <span>{{ t(item.titleKey) }}</span>
                  </router-link>
                </SidebarMenuButton>
                <SidebarMenuAction>
                  <span class="sr-only">{{ item.description }}</span>
                </SidebarMenuAction>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <!-- Inbox sidebar -->
    <template v-if="route.path && isInboxRoute(route.path)">
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="flex items-center justify-between w-full px-1">
                <div class="font-semibold text-xl">
                  <span>{{ t('globals.terms.inbox') }}</span>
                </div>
                <div class="mr-1 mt-1 transition-colors">
                  <router-link :to="{ name: 'search' }">
                    <Search size="18" stroke-width="2.5" class="text-muted-foreground hover:text-foreground" />
                  </router-link>
                </div>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton @click="emit('createConversation')">
                    <Plus />
                    <span>{{ t('conversation.newConversation') }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton :isActive="isActiveParent('/inboxes/assigned')" @click="navigateToInbox('assigned')">
                    <User />
                    <span class="flex-1 truncate">{{ t('globals.terms.myInbox') }}</span>
                    <SidebarCountBadge
                      :count="conversationStore.sidebarCounts.assigned"
                      :ariaLabel="t('conversation.sidebarCounts.assigned', conversationStore.sidebarCounts.assigned)"
                    />
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton :isActive="isActiveParent('/inboxes/mentioned')" @click="navigateToInbox('mentioned')">
                    <AtSign />
                    <span class="flex-1 truncate">
                      {{ t('globals.terms.mention', 2) }}
                    </span>
                    <SidebarCountBadge
                      :count="conversationStore.sidebarCounts.mentioned"
                      :ariaLabel="t('conversation.sidebarCounts.mentioned', conversationStore.sidebarCounts.mentioned)"
                    />
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton :isActive="isActiveParent('/inboxes/unassigned')" @click="navigateToInbox('unassigned')">
                    <CircleDashed />
                    <span class="flex-1 truncate">
                      {{ t('globals.terms.unassigned') }}
                    </span>
                    <SidebarCountBadge
                      :count="conversationStore.sidebarCounts.unassigned"
                      :ariaLabel="t('conversation.sidebarCounts.unassigned', conversationStore.sidebarCounts.unassigned)"
                    />
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton :isActive="isActiveParent('/inboxes/all')" @click="navigateToInbox('all')">
                    <List />
                    <span class="flex-1 truncate">
                      {{ t('globals.messages.all') }}
                    </span>
                    <SidebarCountBadge
                      :count="conversationStore.sidebarCounts.all"
                      :ariaLabel="t('conversation.sidebarCounts.all', conversationStore.sidebarCounts.all)"
                    />
                </SidebarMenuButton>
              </SidebarMenuItem>

              <!-- Team Inboxes -->
              <Collapsible
                defaultOpen
                class="group/collapsible"
                v-if="userTeams.length"
                v-model:open="teamInboxOpen"
              >
                <SidebarMenuItem>
                  <CollapsibleTrigger as-child>
                    <SidebarMenuButton>
                        <span class="sidebar-section-label">
                          {{ t('globals.terms.teamInbox', 2) }}
                        </span>
                        <ChevronRight
                          class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                        />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <SidebarMenuSub>
                      <SidebarMenuSubItem v-for="team in userTeams" :key="team.id">
                        <SidebarMenuButton
                          size="sm"
                          :is-active="route.params.teamID == team.id"
                          @click="navigateToTeamInbox(team.id)"
                        >
                          {{ team.emoji }}<span class="flex-1 truncate" :title="team.name">{{ team.name }}</span>
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>

              <!-- Views -->
              <Collapsible class="group/collapsible" defaultOpen v-model:open="viewInboxOpen" v-if="userStore.can(permissions.VIEW_MANAGE)">
                <SidebarMenuItem>
                  <CollapsibleTrigger asChild>
                    <SidebarMenuButton class="group/item !p-2">
                        <span class="sidebar-section-label">
                          {{ t('globals.terms.view', 2) }}
                        </span>
                        <div>
                          <Plus
                            size="18"
                            @click.stop="openCreateViewDialog"
                            class="rounded-md cursor-pointer transition-colors duration-200 can-hover:opacity-0 can-hover:group-hover/item:opacity-100 hover:bg-sidebar-accent/50 text-muted-foreground hover:text-sidebar-accent-foreground p-1"
                          />
                        </div>
                        <ChevronRight
                          class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                          v-if="userViews.length"
                        />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>

                  <CollapsibleContent>
                    <SidebarMenuSub>
                      <SidebarMenuSubItem
                        v-for="view in userViews" :key="view.id"
                        class="group/view-item"
                      >
                        <SidebarMenuButton
                          size="sm"
                          class="group-has-[[data-sidebar=menu-action]]/menu-item:pr-7"
                          :isActive="route.params.viewID == view.id"
                          @click="navigateToViewInbox(view.id)"
                        >
                          <span class="flex-1 truncate" :title="view.name">{{ view.name }}</span>
                          <SidebarCountBadge
                            :count="viewSidebarCount(view.id)"
                            :ariaLabel="t('conversation.sidebarCounts.view', viewSidebarCount(view.id))"
                          />
                        </SidebarMenuButton>
                        <DropdownMenu>
                          <DropdownMenuTrigger as-child>
                            <SidebarMenuAction
                              class="mr-1 can-hover:opacity-0 can-hover:group-hover/view-item:opacity-100 data-[state=open]:opacity-100"
                              @click.prevent
                            >
                              <EllipsisVertical />
                            </SidebarMenuAction>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent>
                            <DropdownMenuItem @click="() => editView(view)">
                              <span>{{ t('globals.messages.edit') }}</span>
                            </DropdownMenuItem>
                            <DropdownMenuItem @click="() => openDeleteConfirmation(view)">
                              <span>{{ t('globals.messages.delete') }}</span>
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>

              <!-- Shared Views -->
              <Collapsible
                class="group/collapsible"
                defaultOpen
                v-model:open="sharedViewInboxOpen"
                v-if="sharedViews.length"
              >
                <SidebarMenuItem>
                  <CollapsibleTrigger asChild>
                    <SidebarMenuButton class="!p-2">
                        <span class="sidebar-section-label">
                          {{ t('globals.terms.sharedView', 2) }}
                        </span>
                        <ChevronRight
                          class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                        />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>

                  <CollapsibleContent>
                    <SidebarMenuSub>
                      <SidebarMenuSubItem v-for="view in sharedViews" :key="view.id">
                        <SidebarMenuButton
                          size="sm"
                          :isActive="route.params.viewID == view.id"
                          @click="navigateToViewInbox(view.id)"
                        >
                          <span class="flex-1 truncate" :title="view.name">{{
                            view.name
                          }}</span>
                          <SidebarCountBadge
                            :count="viewSidebarCount(view.id)"
                            :ariaLabel="t('conversation.sidebarCounts.view', viewSidebarCount(view.id))"
                          />
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <!-- Main Content Area -->
    <SidebarInset class="bg-canvas !min-h-0 !h-full">
      <slot></slot>
    </SidebarInset>
  </SidebarProvider>

  <!-- View Delete Confirmation Dialog -->
  <AlertDialog v-model:open="isDeleteOpen">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ t('confirm.deleteView') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="handleDeleteView">
          {{ t('globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<style scoped>
:deep(.sidebar-secondary) {
  @apply border border-sidebar-border ml-[3.2rem] rounded-lg overflow-hidden;
  top: 0.40rem !important;
  bottom: 0.35rem !important;
  height: auto !important;
}

/* Override SidebarProvider height */
:deep(.group\/sidebar-wrapper) {
  min-height: auto !important;
  height: 100%;
}

</style>
