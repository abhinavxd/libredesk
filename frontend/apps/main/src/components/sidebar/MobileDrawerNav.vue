<template>
  <!-- The top-level destinations plus notifications and the user menu. On
       desktop these live in the icon rail; on mobile the rail is not rendered,
       so they are folded into the top of this drawer. -->
  <SidebarGroup v-if="isMobile" class="border-b border-sidebar-border pb-2">
    <SidebarGroupContent>
      <SidebarMenu>
        <PrimaryNavItems variant="drawer" />
        <SidebarMenuItem>
          <NotificationBell />
        </SidebarMenuItem>
        <SidebarMenuItem>
          <SidebarNavUser />
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarGroupContent>
  </SidebarGroup>
</template>

<script setup>
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  useSidebar
} from '@shared-ui/components/ui/sidebar'
import PrimaryNavItems from './PrimaryNavItems.vue'
import NotificationBell from './NotificationBell.vue'
import SidebarNavUser from './SidebarNavUser.vue'

const { isMobile } = useSidebar()
</script>
