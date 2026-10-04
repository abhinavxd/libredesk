<template>
  <Tabs v-model="theme">
    <TabsList class="h-8 p-0.5">
      <TabsTrigger
        value="light"
        class="px-2"
        data-preview-theme="light"
        :aria-label="$t('globals.terms.light')"
      >
        <Sun class="size-3.5" />
      </TabsTrigger>
      <TabsTrigger
        value="dark"
        class="px-2"
        data-preview-theme="dark"
        :aria-label="$t('globals.terms.dark')"
      >
        <Moon class="size-3.5" />
      </TabsTrigger>
    </TabsList>
  </Tabs>
</template>

<script setup>
import { Tabs, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { Sun, Moon } from 'lucide-vue-next'

const theme = defineModel({ type: String, default: 'light' })
</script>
