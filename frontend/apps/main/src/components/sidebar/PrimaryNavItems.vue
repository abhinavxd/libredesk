<template>
  <SidebarMenuItem v-for="item in items" :key="item.key">
    <Tooltip>
      <TooltipTrigger as-child>
        <SidebarMenuButton asChild :isActive="item.isActive">
          <router-link
            :to="item.to"
            :aria-label="item.label"
            :aria-current="item.isActive ? 'page' : undefined"
          >
            <component :is="item.icon" aria-hidden="true" />
            <span v-if="variant === 'drawer'">{{ item.label }}</span>
          </router-link>
        </SidebarMenuButton>
      </TooltipTrigger>
      <TooltipContent v-if="variant === 'rail'" side="right">
        <p>{{ item.label }}</p>
      </TooltipContent>
    </Tooltip>
  </SidebarMenuItem>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from '@shared-ui/i18n'
import { useStorage } from '@vueuse/core'
import { Inbox, Shield, FileLineChart, BookUser } from 'lucide-vue-next'
import { SidebarMenuButton, SidebarMenuItem } from '@shared-ui/components/ui/sidebar'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useIsMobile } from '@shared-ui/composables'
import { useUserStore } from '@main/stores/user'

defineProps({
  // 'rail' (desktop icon rail) or 'drawer' (mobile nav drawer).
  variant: { type: String, default: 'rail' }
})

const route = useRoute()
const i18n = useI18n()
const userStore = useUserStore()
const isMobile = useIsMobile()

const lastInboxPath = useStorage('lastInboxPath', '')

const inboxTarget = computed(() => {
  if (!lastInboxPath.value) return { name: 'inboxes' }
  if (!isMobile.value) return lastInboxPath.value
  return lastInboxPath.value.replace(/\/conversation\/[^/]+$/, '') || { name: 'inboxes' }
})

const items = computed(() =>
  [
    {
      key: 'inboxes',
      icon: Inbox,
      label: i18n._('{count, plural, one {Inbox} other {Inboxes}}', 2),
      to: inboxTarget.value,
      isActive: route.path.startsWith('/inboxes'),
      show: true
    },
    {
      key: 'contacts',
      icon: BookUser,
      label: i18n._('{count, plural, one {Contact} other {Contacts}}', 2),
      to: { name: 'contacts' },
      isActive: route.path.startsWith('/contacts'),
      show: userStore.can('contacts:read_all')
    },
    {
      key: 'reports',
      icon: FileLineChart,
      label: i18n._('{count, plural, one {Report} other {Reports}}', 2),
      to: { name: 'reports' },
      isActive: route.path.startsWith('/reports'),
      show: userStore.hasReportTabPermissions
    },
    {
      key: 'admin',
      icon: Shield,
      label: i18n._('{count, plural, one {Admin} other {Admins}}'),
      to: { name: userStore.can('general_settings:manage') ? 'general' : 'admin' },
      isActive: route.path.startsWith('/admin'),
      show: userStore.hasAdminTabPermissions
    }
  ].filter((item) => item.show)
)
</script>
