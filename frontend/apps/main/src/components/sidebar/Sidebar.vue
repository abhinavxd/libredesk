<script setup>
import {
  adminNavItems,
  reportsNavItems,
  accountNavItems,
  contactNavItems
} from '@main/constants/navigation'
import { useRoute } from 'vue-router'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger
} from '@shared-ui/components/ui/collapsible'
import BetaBadge from '@main/components/BetaBadge.vue'
import SidebarSectionHeader from './SidebarSectionHeader.vue'
import { Button } from '@shared-ui/components/ui/button'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
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
  UserRoundX,
  Inbox,
  AtSign
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
import { useI18n } from '@shared-ui/i18n'
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
const i18n = useI18n()
const emit = defineEmits(['createView', 'editView', 'deleteView', 'createConversation'])

const isActiveParent = (parentHref) => {
  return route.path.startsWith(parentHref)
}

const isInboxRoute = (path) => {
  return path.startsWith('/inboxes')
}

const sidebarProviderRef = ref(null)

// An open mobile drawer treats taps on the view form as outside clicks and closes it.
const closeMobileSidebar = () => sidebarProviderRef.value?.setOpenMobile(false)

const openCreateViewDialog = () => {
  closeMobileSidebar()
  emit('createView')
}

const editView = (view) => {
  closeMobileSidebar()
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
const toggleAdminCollapsible = (id) => {
  openAdminCollapsible.value = openAdminCollapsible.value === id ? null : id
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
      openAdminCollapsible.value = activeItem.titleMessage.id
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
    ref="sidebarProviderRef"
    class="relative h-full min-h-0 gap-1 overflow-hidden has-[[data-collapsible=offcanvas]]:gap-0"
    style="--sidebar-width: 14rem"
    :default-open="sidebarOpen"
    v-on:update:open="sidebarOpen = $event"
  >
    <!-- Contacts sidebar -->
    <template
      v-if="route.matched.some((record) => record.name && record.name.startsWith('contact'))"
    >
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarSectionHeader :title="i18n.ngettext('Contact', 'Contacts', 2)" />
        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredContactsNavItems" :key="item.href || item.titleMessage.id">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <component :is="navIconMap[item.icon]" v-if="item.icon" />
                    <span>{{ i18n._(item.allLabelMessage) }}</span>
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
        <SidebarSectionHeader :title="i18n.ngettext('Report', 'Reports', 2)" />
        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredReportsNavItems" :key="item.href || item.titleMessage.id">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <component :is="navIconMap[item.icon]" v-if="item.icon" />
                    <span>{{ i18n._(item.titleMessage) }}</span>
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
        <SidebarSectionHeader :title="i18n.ngettext('Admin', 'Admins', 1)">
          <template #suffix>
            <span>({{ settingsStore.settings['app.version'] }})</span>
          </template>
        </SidebarSectionHeader>
        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredAdminNavItems" :key="item.href || item.titleMessage.id">
                <SidebarMenuButton
                  v-if="!item.children"
                  :isActive="isActiveParent(item.href)"
                  asChild
                >
                  <router-link :to="item.href">
                    <span>{{ i18n._(item.titleMessage) }}</span>
                  </router-link>
                </SidebarMenuButton>

                <Collapsible
                  v-else
                  class="group/collapsible"
                  :open="openAdminCollapsible === item.titleMessage.id"
                  @update:open="toggleAdminCollapsible(item.titleMessage.id)"
                >
                  <CollapsibleTrigger as-child>
                    <SidebarMenuButton :isActive="isActiveParent(item.href)">
                      <span>{{ i18n._(item.titleMessage, item.isTitlePlural === true ? 2 : 1) }}</span>
                      <BetaBadge v-if="item.badge" class="ml-1.5">
                        {{ item.badge }}
                      </BetaBadge>
                      <ChevronRight
                        class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                      />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <SidebarMenuSub>
                      <SidebarMenuSubItem v-for="child in item.children" :key="child.href || child.titleMessage.id">
                        <SidebarMenuButton size="sm" :isActive="isActiveParent(child.href)" asChild>
                          <router-link :to="child.href">
                            <component :is="navIconMap[child.icon]" v-if="child.icon" />
                            <span>{{
                              i18n._(child.titleMessage, child.isTitlePlural === true ? 2 : 1)
                            }}</span>
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
        <SidebarSectionHeader :title="i18n.ngettext('Account', 'Accounts', 1)" />
        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in accountNavItems" :key="item.href || item.titleMessage.id">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <component :is="navIconMap[item.icon]" v-if="item.icon" />
                    <span>{{ i18n._(item.titleMessage) }}</span>
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
        <SidebarSectionHeader :title="i18n.ngettext('Inbox', 'Inboxes', 1)">
          <template #actions>
            <Button variant="ghost" size="icon" as-child class="size-8 text-muted-foreground">
              <router-link :to="{ name: 'search' }" :aria-label="i18n._('Search')">
                <Search size="18" stroke-width="2.5" aria-hidden="true" />
              </router-link>
            </Button>
          </template>
        </SidebarSectionHeader>

        <SidebarContent>
          <MobileDrawerNav />
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton @click="emit('createConversation')">
                  <Plus />
                  <span>{{ i18n._('New conversation') }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton
                  :isActive="isActiveParent('/inboxes/assigned')"
                  @click="navigateToInbox('assigned')"
                >
                  <User />
                  <span class="flex-1 truncate">{{ i18n.ngettext('My Inbox', 'My Inboxes', 1) }}</span>
                  <SidebarCountBadge
                    :count="conversationStore.sidebarCounts.assigned"
                    :ariaLabel="
                      i18n.ngettext('{count} open conversation in my inbox', '{count} open conversations in my inbox', conversationStore.sidebarCounts.assigned)
                    "
                  />
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton
                  :isActive="isActiveParent('/inboxes/mentioned')"
                  @click="navigateToInbox('mentioned')"
                >
                  <AtSign />
                  <span class="flex-1 truncate">
                    {{ i18n.ngettext('Mention', 'Mentions', 2) }}
                  </span>
                  <SidebarCountBadge
                    :count="conversationStore.sidebarCounts.mentioned"
                    :ariaLabel="
                      i18n.ngettext('{count} open conversation with mentions', '{count} open conversations with mentions', conversationStore.sidebarCounts.mentioned)
                    "
                  />
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton
                  :isActive="isActiveParent('/inboxes/unassigned')"
                  @click="navigateToInbox('unassigned')"
                >
                  <UserRoundX />
                  <span class="flex-1 truncate">
                    {{ i18n._('Unassigned') }}
                  </span>
                  <SidebarCountBadge
                    :count="conversationStore.sidebarCounts.unassigned"
                    :ariaLabel="
                      i18n.ngettext('{count} open unassigned conversation', '{count} open unassigned conversations', conversationStore.sidebarCounts.unassigned)
                    "
                  />
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton
                  :isActive="isActiveParent('/inboxes/all')"
                  @click="navigateToInbox('all')"
                >
                  <Inbox />
                  <span class="flex-1 truncate">
                    {{ i18n._('All') }}
                  </span>
                  <SidebarCountBadge
                    :count="conversationStore.sidebarCounts.all"
                    :ariaLabel="
                      i18n.ngettext('{count} open conversation in all', '{count} open conversations in all', conversationStore.sidebarCounts.all)
                    "
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
                        {{ i18n.ngettext('Team Inbox', 'Team Inboxes', 2) }}
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
                          {{ team.emoji
                          }}<span class="flex-1 truncate" :title="team.name">{{ team.name }}</span>
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>

              <!-- Views -->
              <Collapsible
                class="group/collapsible"
                defaultOpen
                v-model:open="viewInboxOpen"
                v-if="userStore.can(permissions.VIEW_MANAGE)"
              >
                <SidebarMenuItem>
                  <CollapsibleTrigger asChild>
                    <SidebarMenuButton class="group/item !p-2">
                      <span class="sidebar-section-label">
                        {{ i18n.ngettext('View', 'Views', 2) }}
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
                        v-for="view in userViews"
                        :key="view.id"
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
                            :ariaLabel="
                              i18n.ngettext('{count} conversation in this view', '{count} conversations in this view', viewSidebarCount(view.id))
                            "
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
                              <span>{{ i18n._('Edit') }}</span>
                            </DropdownMenuItem>
                            <DropdownMenuItem @click="() => openDeleteConfirmation(view)">
                              <span>{{ i18n._('Delete') }}</span>
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
                        {{ i18n.ngettext('Shared view', 'Shared views', 2) }}
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
                          <span class="flex-1 truncate" :title="view.name">{{ view.name }}</span>
                          <SidebarCountBadge
                            :count="viewSidebarCount(view.id)"
                            :ariaLabel="
                              i18n.ngettext('{count} conversation in this view', '{count} conversations in this view', viewSidebarCount(view.id))
                            "
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
        <AlertDialogTitle>{{ i18n._('Are you absolutely sure?') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ i18n._('This action cannot be undone. This will permanently delete this view.') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ i18n._('Cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="handleDeleteView">
          {{ i18n._('Delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<style scoped>
:deep(.sidebar-secondary) {
  @apply absolute inset-y-0 h-full overflow-hidden rounded-xl border border-sidebar-border;
}

:deep(.group\/sidebar-wrapper) {
  min-height: auto !important;
  height: 100%;
}
</style>
