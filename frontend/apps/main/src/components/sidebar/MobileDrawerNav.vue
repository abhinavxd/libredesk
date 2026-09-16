<template>
  <SidebarGroup v-if="isMobile" class="border-b border-sidebar-border pb-2">
    <SidebarGroupContent>
      <SidebarMenu>
        <PrimaryNavItems variant="drawer" />
      </SidebarMenu>
    </SidebarGroupContent>
  </SidebarGroup>
</template>

<script setup>
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  useSidebar
} from '@shared-ui/components/ui/sidebar'
import PrimaryNavItems from './PrimaryNavItems.vue'

const { isMobile } = useSidebar()
</script>
