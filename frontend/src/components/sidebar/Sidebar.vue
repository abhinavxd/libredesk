<script setup>
import {
  adminNavItems,
  reportsNavItems,
  accountNavItems,
  contactNavItems
} from '@/constants/navigation'
import { RouterLink, useRoute } from 'vue-router'
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

defineProps({
  userTeams: { type: Array, default: () => [] },
  userViews: { type: Array, default: () => [] }
})
const userStore = useUserStore()
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

const deleteView = (view) => {
  emit('deleteView', view)
}

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
                  <router-link :to="{ name: 'inbox', params: { type: 'assigned' } }">
                    <User />
                    <span>{{ t('globals.terms.myInbox') }}</span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton asChild :isActive="isActiveParent('/inboxes/unassigned')">
                  <router-link :to="{ name: 'inbox', params: { type: 'unassigned' } }">
                    <CircleDashed />
                    <span>
                      {{ t('globals.terms.unassigned') }}
                    </span>
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>

              <SidebarMenuItem>
                <SidebarMenuButton asChild :isActive="isActiveParent('/inboxes/all')">
                  <router-link :to="{ name: 'inbox', params: { type: 'all' } }">
                    <List />
                    <span>
                      {{ t('globals.messages.all') }}
                    </span>
                  </router-link>
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
                          <router-link :to="{ name: 'team-inbox', params: { teamID: team.id } }">
                            {{ team.emoji }}<span>{{ team.name }}</span>
                          </router-link>
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
                          <router-link :to="{ name: 'view-inbox', params: { viewID: view.id } }">
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
                          </router-link>
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

    <!-- Main Content Area -->
    <SidebarInset>
      <slot></slot>
    </SidebarInset>
  </SidebarProvider>
</template>
