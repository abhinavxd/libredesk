<template>
  <DropdownMenu v-if="hasMultipleThemes">
    <Tooltip>
      <TooltipTrigger as-child>
        <DropdownMenuTrigger as-child>
          <SidebarMenuButton>
            <Palette />
          </SidebarMenuButton>
        </DropdownMenuTrigger>
      </TooltipTrigger>
      <TooltipContent side="right">
        <p>{{ t('theme.switcher') }}</p>
      </TooltipContent>
    </Tooltip>
    <DropdownMenuContent side="right" align="end">
      <DropdownMenuItem
        v-for="t in THEMES"
        :key="t.id"
        @click="setTheme(t.id)"
        :class="{ 'bg-accent': currentTheme === t.id }"
      >
        <Check v-if="currentTheme === t.id" class="mr-2 h-4 w-4" />
        <span v-else class="mr-2 h-4 w-4" />
        {{ t.label }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { Palette, Check } from 'lucide-vue-next'
import { SidebarMenuButton } from '@shared-ui/components/ui/sidebar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useTheme } from '@main/composables/useTheme'

const { t } = useI18n()
const { THEMES, currentTheme, setTheme, hasMultipleThemes } = useTheme()
</script>
