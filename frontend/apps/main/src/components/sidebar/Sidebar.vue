<script setup>
import { adminNavItems } from '../../constants/navigation'
import { useRoute, useRouter } from 'vue-router'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger
} from '@shared-ui/components/ui/collapsible'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider
} from '@shared-ui/components/ui/sidebar'
import { ArrowLeft, ChevronRight, Mail, ShieldCheck, SquarePen } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useStorage } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { useUserStore } from '@main/stores/user'
import { useAddressStore } from '@main/stores/address'
import { useConversationStore } from '@main/stores/conversation'
import { useReviewStore } from '@main/stores/review'
import { useComposeStore } from '@main/stores/compose'
import { permissions as perms } from '@main/constants/permissions'
import { useAddressNavigation } from '@main/composables/useAddressNavigation'
import { Button } from '@shared-ui/components/ui/button'
import { navIconMap } from '@main/constants/navIcons'
import { filterNavItems } from '@main/utils/nav-permissions'
import { addressLabel } from '@main/utils/address-display'
import MobileDrawerFooter from './MobileDrawerFooter.vue'
import MobileSidebarSwipeArea from './MobileSidebarSwipeArea.vue'
import SidebarCountBadge from './SidebarCountBadge.vue'
import BrandLogo from '@main/components/brand/BrandLogo.vue'

const userStore = useUserStore()
const addressStore = useAddressStore()
const conversationStore = useConversationStore()
const reviewStore = useReviewStore()
const composeStore = useComposeStore()
const settingsStore = useAppSettingsStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { navigateToAddress } = useAddressNavigation()

const isActiveParent = (parentHref) => route.path.startsWith(parentHref)
const isMailRoute = (path) =>
  path.startsWith('/addresses') ||
  path.startsWith('/conversation') ||
  path.startsWith('/reviews')
const canCompose = computed(() => userStore.can(perms.CONVERSATIONS_CREATE))
const isActiveAddress = (addressID) => String(route.params.addressID) === String(addressID)
const filteredAdminNavItems = computed(() => filterNavItems(adminNavItems, userStore.can))

const openAdminCollapsible = ref(null)
const toggleAdminCollapsible = (titleKey) => {
  openAdminCollapsible.value = openAdminCollapsible.value === titleKey ? null : titleKey
}
watch(
  [() => route.path, filteredAdminNavItems],
  () => {
    const activeItem = filteredAdminNavItems.value.find((item) => {
      if (!item.children) return isActiveParent(item.href)
      return item.children.some((child) => isActiveParent(child.href))
    })
    if (activeItem) openAdminCollapsible.value = activeItem.titleKey
  },
  { immediate: true }
)

const sidebarOpen = useStorage('mainSidebarOpen', true)
const addressesOpen = useStorage('addressesSectionOpen', true)

onMounted(() => {
  addressStore.fetchAddresses()
  conversationStore.fetchSidebarCounts({ force: true })
  reviewStore.fetchCounts()
})
</script>

<template>
  <SidebarProvider
    style="--sidebar-width: 18rem"
    :default-open="sidebarOpen"
    v-on:update:open="sidebarOpen = $event"
  >
    <template v-if="route.matched.some((record) => record.name && record.name.startsWith('admin'))">
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <Button
            variant="default"
            class="h-10 w-full justify-start bg-success px-3 text-success-foreground hover:bg-success/90"
            @click="router.push('/addresses')"
          >
            <ArrowLeft class="h-4 w-4" aria-hidden="true" />
            Back to addresses
          </Button>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="flex w-full flex-col items-start justify-between px-1">
                <span class="text-xl font-semibold">{{ t('globals.terms.admin') }}</span>
                <div
                  v-if="settingsStore.settings['app.version']"
                  class="text-xs text-muted-foreground"
                >
                  {{ settingsStore.settings['app.version'] }}
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
                  <router-link :to="item.href"
                    ><span>{{ t(item.titleKey) }}</span></router-link
                  >
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
                        class="ml-1.5 shrink-0 rounded-full border-warning/50 bg-warning/10 px-[5.5px] py-[3px] text-[9px] font-medium leading-none tracking-[0.07em] text-warning-600 uppercase"
                        >{{ item.badge }}</Badge
                      >
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
                            <span>{{
                              t(child.titleKey, child.isTitleKeyPlural === true ? 2 : 1)
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

    <template v-if="route.path && isMailRoute(route.path)">
      <Sidebar collapsible="offcanvas" class="sidebar-secondary">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <div class="flex w-full items-center px-1">
                <div class="min-w-0 text-xl font-semibold">
                  <BrandLogo :name="settingsStore.siteTitle" :logo="settingsStore.siteLogo" />
                </div>
              </div>
            </SidebarMenuItem>
          </SidebarMenu>
          <Button
            v-if="canCompose"
            variant="default"
            class="h-10 w-full justify-start bg-success px-3 text-success-foreground hover:bg-success/90 max-md:h-12"
            @click="composeStore.open()"
          >
            <SquarePen class="h-4 w-4" aria-hidden="true" />
            {{ t('compose.button') }}
          </Button>
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup v-if="reviewStore.enabled" class="pb-0">
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  :isActive="route.path.startsWith('/reviews')"
                  class="!h-auto min-h-10 px-2.5 py-2 max-md:min-h-12"
                  asChild
                >
                  <router-link :to="{ name: 'reviews' }">
                    <ShieldCheck class="h-4 w-4 text-review" aria-hidden="true" />
                    <span class="flex-1 truncate text-base font-medium max-md:text-lg">{{ t('review.title') }}</span>
                    <SidebarCountBadge
                      tone="review"
                      :count="reviewStore.badgeCount"
                      :ariaLabel="t('review.badgeLabel', { count: reviewStore.badgeCount })"
                    />
                  </router-link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
          <SidebarGroup>
            <Collapsible class="group/collapsible" v-model:open="addressesOpen">
              <SidebarMenu>
                <SidebarMenuItem>
                  <CollapsibleTrigger asChild>
                    <SidebarMenuButton class="!p-2">
                      <span class="text-sm font-medium text-muted-foreground">Addresses</span>
                      <ChevronRight
                        class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
                      />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <SidebarMenu class="max-md:gap-2">
                      <SidebarMenuItem v-for="address in addressStore.addresses" :key="address.id">
                        <SidebarMenuButton
                          size="default"
                          :isActive="isActiveAddress(address.id)"
                          :title="addressLabel(address) + ' · ' + address.address"
                          :class="[
                            '!h-auto min-h-12 items-start px-2.5 py-2.5 max-md:min-h-16 max-md:rounded-xl max-md:border max-md:border-sidebar-border/80 max-md:bg-sidebar-accent/80 max-md:px-3 max-md:py-3 max-md:shadow-sm',
                            {
                              'opacity-60': !address.enabled,
                              'max-md:border-success/50 max-md:bg-success/10': isActiveAddress(
                                address.id
                              )
                            }
                          ]"
                          @click="navigateToAddress(address.id)"
                        >
                          <Mail class="mt-0.5 h-4 w-4 max-md:mt-1 max-md:h-5 max-md:w-5" />
                          <span class="min-w-0 flex-1 truncate">
                            <span
                              class="block truncate text-base leading-6 font-medium max-md:text-lg"
                            >
                              {{ addressLabel(address) }}
                            </span>
                            <span
                              class="block truncate text-sm leading-5 text-muted-foreground max-md:text-base"
                            >
                              {{ address.address }}
                            </span>
                          </span>
                          <SidebarCountBadge
                            :count="conversationStore.sidebarCounts.addresses?.[address.id] || 0"
                            :ariaLabel="
                              String(conversationStore.sidebarCounts.addresses?.[address.id] || 0) +
                              ' unread messages for ' +
                              address.address
                            "
                          />
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    </SidebarMenu>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </SidebarMenu>
            </Collapsible>
          </SidebarGroup>
        </SidebarContent>
        <MobileDrawerFooter />
      </Sidebar>
    </template>

    <MobileSidebarSwipeArea>
      <SidebarInset class="!h-full !min-h-0 bg-canvas"><slot /></SidebarInset>
    </MobileSidebarSwipeArea>
  </SidebarProvider>
</template>

<style scoped>
:deep(.sidebar-secondary) {
  @apply ml-0 overflow-hidden rounded-lg border border-sidebar-border;
  bottom: 0.35rem !important;
  height: auto !important;
  left: 0.35rem;
  top: 0.4rem !important;
}

:deep(.group\/sidebar-wrapper) {
  min-height: auto !important;
  height: 100%;
}
</style>
