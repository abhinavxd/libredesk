<template>
  <div>
    <div class="flex items-center px-2 h-12">
      <SidebarTrigger class="cursor-pointer" />
      <Input
        v-model="model"
        :placeholder="t('globals.terms.search')"
        class="w-full border-none shadow-none focus:ring-0 focus:ring-offset-0"
      />
    </div>
    <Separator />
  </div>
</template>

<script setup>
import { Separator } from '@shared-ui/components/ui/separator'
import { Input } from '@shared-ui/components/ui/input'
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useI18n } from 'vue-i18n'
const model = defineModel({
  type: String,
  default: '',
  required: false
})
const { t } = useI18n()
</script>

<style scoped>
.focus\:ring-0:focus {
  --tw-ring-offset-shadow: none;
  --tw-ring-shadow: none;
  box-shadow: none;
}
</style>
