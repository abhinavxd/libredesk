<template>
  <SidebarFooter v-if="isMobile" class="border-t border-sidebar-border">
    <SidebarMenu>
      <SidebarMenuItem>
        <NotificationBell />
      </SidebarMenuItem>
      <SidebarMenuItem>
        <SidebarNavUser />
      </SidebarMenuItem>
    </SidebarMenu>
  </SidebarFooter>
</template>

<script setup>
import {
  SidebarFooter,
  SidebarMenu,
  SidebarMenuItem,
  useSidebar
} from '@shared-ui/components/ui/sidebar'
import NotificationBell from './NotificationBell.vue'
import SidebarNavUser from './SidebarNavUser.vue'

const { isMobile } = useSidebar()
</script>
