<script setup>
import {
  adminNavItems,
  reportsNavItems,
  accountNavItems,
  contactNavItems,
  articlesNavItems
} from '@/constants/navigation'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
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
  SidebarProvider,
  SidebarRail
} from '@/components/ui/sidebar'
import { useAppSettingsStore } from '@/stores/appSettings'
import {
  ChevronRight,
  EllipsisVertical,
  User,
  Search,
  Plus,
  CircleDashed,
  List
} from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { filterNavItems } from '@/utils/nav-permissions'
import { useStorage } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { useConversationStore } from '@/stores/conversation'

defineProps({
  userTeams: { type: Array, default: () => [] },
  userViews: { type: Array, default: () => [] }
})
const userStore = useUserStore()
const conversationStore = useConversationStore()
const settingsStore = useAppSettingsStore()
const route = useRoute()
const router = useRouter()
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

const deleteView = (view) => {
  emit('deleteView', view)
}

// Navigation methods with conversation retention
const navigateToInbox = (type) => {
  if (conversationStore.hasConversationOpen && conversationStore.conversation.data?.uuid) {
    router.push({
      name: 'inbox-conversation',
      params: {
        type,
        uuid: conversationStore.conversation.data.uuid
      }
    })
  } else {
    router.push({
      name: 'inbox',
      params: { type }
    })
  }
}

const navigateToTeamInbox = (teamID) => {
  if (conversationStore.hasConversationOpen && conversationStore.conversation.data?.uuid) {
    router.push({
      name: 'team-inbox-conversation',
      params: {
        teamID,
        uuid: conversationStore.conversation.data.uuid
      }
    })
  } else {
    router.push({
      name: 'team-inbox',
      params: { teamID }
    })
  }
}

const navigateToViewInbox = (viewID) => {
  if (conversationStore.hasConversationOpen && conversationStore.conversation.data?.uuid) {
    router.push({
      name: 'view-inbox-conversation',
      params: {
        viewID,
        uuid: conversationStore.conversation.data.uuid
      }
    })
  } else {
    router.push({
      name: 'view-inbox',
      params: { viewID }
    })
  }
}

const filteredAdminNavItems = computed(() => filterNavItems(adminNavItems, userStore.can))
const filteredReportsNavItems = computed(() => filterNavItems(reportsNavItems, userStore.can))
const filteredContactsNavItems = computed(() => filterNavItems(contactNavItems, userStore.can))
const filteredArticlesNavItems = computed(() => filterNavItems(articlesNavItems, userStore.can))


// For auto opening admin collapsibles when a child route is active
const openAdminCollapsible = ref(null)
const toggleAdminCollapsible = (titleKey) => {
  openAdminCollapsible.value = openAdminCollapsible.value === titleKey ? null : titleKey
}

// For auto opening articles collapsibles when a child route is active
const openArticlesCollapsible = ref(null)
const toggleArticlesCollapsible = (titleKey) => {
  openArticlesCollapsible.value = openArticlesCollapsible.value === titleKey ? null : titleKey
} 


// Watch for route changes and update the active collapsible
const updateActiveCollapsible = (filteredItems, openCollapsible) => {
  const activeItem = filteredItems.value.find((item) => {
    if (!item.children) return isActiveParent(item.href)
    return item.children.some((child) => isActiveParent(child.href))
  })
  if (activeItem) {
    openCollapsible.value = activeItem.titleKey
  }
}
watch(
  [() => route.path, filteredAdminNavItems],
  () => updateActiveCollapsible(filteredAdminNavItems, openAdminCollapsible),
  { immediate: true }
)
watch(
  [() => route.path, filteredArticlesNavItems],
  () => updateActiveCollapsible(filteredArticlesNavItems, openArticlesCollapsible),
  { immediate: true }
)

// Sidebar open state in local storage
const sidebarOpen = useStorage('mainSidebarOpen', true)
const teamInboxOpen = useStorage('teamInboxOpen', true)
const viewInboxOpen = useStorage('viewInboxOpen', true)
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
      <Sidebar collapsible="offcanvas" class="border-r ml-12">
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
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredContactsNavItems" :key="item.titleKey">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <span>{{
                      t('globals.messages.all', {
                        name: t(item.titleKey, 2).toLowerCase()
                      })
                    }}</span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
        <SidebarRail />
      </Sidebar>
    </template>

    <!-- Reports sidebar -->
    <template
      v-if="
        userStore.hasReportTabPermissions &&
        route.matched.some((record) => record.name && record.name.startsWith('reports'))
      "
    >
      <Sidebar collapsible="offcanvas" class="border-r ml-12">
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
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredReportsNavItems" :key="item.titleKey">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
                    <span>{{ t(item.titleKey) }}</span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
        <SidebarRail />
      </Sidebar>
    </template>

    <!-- Admin Sidebar -->
    <template v-if="route.matched.some((record) => record.name && record.name.startsWith('admin'))">
      <Sidebar collapsible="offcanvas" class="border-r ml-12">
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
                            <span>{{ t(child.titleKey) }}</span>
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
        <SidebarRail />
      </Sidebar>
    </template>

    <!-- Account sidebar -->
    <template v-if="isActiveParent('/account')">
      <Sidebar collapsible="offcanvas" class="border-r ml-12">
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
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in accountNavItems" :key="item.titleKey">
                <SidebarMenuButton :isActive="isActiveParent(item.href)" asChild>
                  <router-link :to="item.href">
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
        <SidebarRail />
      </Sidebar>
    </template>

    <!-- Inbox sidebar -->
    <template v-if="route.path && isInboxRoute(route.path)">
      <Sidebar collapsible="offcanvas" class="border-r ml-12">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="flex items-center justify-between w-full px-1">
                <div class="font-semibold text-xl">
                  <span>{{ t('globals.terms.inbox') }}</span>
                </div>
                <div class="mr-1 mt-1 hover:scale-110 transition-transform">
                  <router-link :to="{ name: 'search' }">
                    <Search size="18" stroke-width="2.5" />
                  </router-link>
                </div>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild>
                  <a href="#" @click="emit('createConversation')">
                    <Plus />
                    <span
                      >{{
                        t('globals.messages.new', {
                          name: t('globals.terms.conversation').toLowerCase()
                        })
                      }}
                    </span>
                  </a>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild :isActive="isActiveParent('/inboxes/assigned')">
                  <a href="#" @click.prevent="navigateToInbox('assigned')">
                    <User />
                    <span>{{ t('globals.terms.myInbox') }}</span>
                  </a>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton asChild :isActive="isActiveParent('/inboxes/unassigned')">
                  <a href="#" @click.prevent="navigateToInbox('unassigned')">
                    <CircleDashed />
                    <span>
                      {{ t('globals.terms.unassigned') }}
                    </span>
                  </a>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton asChild :isActive="isActiveParent('/inboxes/all')">
                  <a href="#" @click.prevent="navigateToInbox('all')">
                    <List />
                    <span>
                      {{ t('globals.messages.all') }}
                    </span>
                  </a>
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
                    <SidebarMenuButton asChild>
                      <router-link to="#">
                        <!-- <Users /> -->
                        <span>
                          {{ t('globals.terms.teamInbox', 2) }}
                        </span>
                        <ChevronRight
                          class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                        />
                      </router-link>
                    </SidebarMenuButton>
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <SidebarMenuSub v-for="team in userTeams" :key="team.id">
                      <SidebarMenuSubItem>
                        <SidebarMenuButton
                          size="sm"
                          :is-active="route.params.teamID == team.id"
                          asChild
                        >
                          <a href="#" @click.prevent="navigateToTeamInbox(team.id)">
                            {{ team.emoji }}<span>{{ team.name }}</span>
                          </a>
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>

              <!-- Views -->
              <Collapsible class="group/collapsible" defaultOpen v-model:open="viewInboxOpen">
                <SidebarMenuItem>
                  <CollapsibleTrigger asChild>
                    <SidebarMenuButton asChild>
                      <router-link to="#" class="group/item !p-2">
                        <!-- <SlidersHorizontal /> -->
                        <span>
                          {{ t('globals.terms.view', 2) }}
                        </span>
                        <div>
                          <Plus
                            size="18"
                            @click.stop="openCreateViewDialog"
                            class="rounded cursor-pointer opacity-0 transition-all duration-200 group-hover/item:opacity-100 hover:bg-gray-200 hover:shadow-sm text-gray-600 hover:text-gray-800 transform hover:scale-105 active:scale-100 p-1"
                          />
                        </div>
                        <ChevronRight
                          class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                          v-if="userViews.length"
                        />
                      </router-link>
                    </SidebarMenuButton>
                  </CollapsibleTrigger>

                  <CollapsibleContent>
                    <SidebarMenuSub v-for="view in userViews" :key="view.id">
                      <SidebarMenuSubItem>
                        <SidebarMenuButton
                          size="sm"
                          :isActive="route.params.viewID == view.id"
                          asChild
                        >
                          <a href="#" @click.prevent="navigateToViewInbox(view.id)">
                            <span class="break-words w-32 truncate">{{ view.name }}</span>
                            <SidebarMenuAction :showOnHover="true" class="mr-3">
                              <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                  <EllipsisVertical />
                                </DropdownMenuTrigger>
                                <DropdownMenuContent>
                                  <DropdownMenuItem @click="() => editView(view)">
                                    <span>{{ t('globals.messages.edit') }}</span>
                                  </DropdownMenuItem>
                                  <DropdownMenuItem @click="() => deleteView(view)">
                                    <span>{{ t('globals.messages.delete') }}</span>
                                  </DropdownMenuItem>
                                </DropdownMenuContent>
                              </DropdownMenu>
                            </SidebarMenuAction>
                          </a>
                        </SidebarMenuButton>
                      </SidebarMenuSubItem>
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
      </Sidebar>
    </template>


    <!-- Articles Sidebar -->
    <template v-if="route.matched.some((record) => record.name && record.name.startsWith('article'))">
      <Sidebar collapsible="offcanvas" class="border-r ml-12">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="flex flex-col items-start justify-between w-full px-1">
                <span class="font-semibold text-xl">
                  {{ t('articles.support.article') }} 
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
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem v-for="item in filteredArticlesNavItems" :key="item.titleKey">
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
                  :open="openArticlesCollapsible === item.titleKey"
                  @update:open="toggleArticlesCollapsible(item.titleKey)"
                >
                  <CollapsibleTrigger as-child>
                    <SidebarMenuButton :isActive="isActiveParent(item.href)">
                      <span>{{ t(item.titleKey) }}</span>
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
                            <span>{{ t(child.titleKey) }}</span>
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
        <SidebarRail />
      </Sidebar>
    </template>

    <!-- Main Content Area -->
    <SidebarInset>
      <slot></slot>
    </SidebarInset>
  </SidebarProvider>
</template>
