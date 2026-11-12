<template>
  <div data-admin-toolbar class="flex min-w-0 flex-wrap items-center justify-between gap-3">
    <div v-if="$slots.left" class="min-w-0 max-w-full overflow-x-auto p-1 -m-1">
      <slot name="left" />
    </div>
    <div
      v-if="searchable || $slots.actions"
      class="ml-auto flex min-w-0 max-w-full flex-wrap items-center justify-end gap-3"
    >
      <SearchInput
        v-if="searchable"
        v-model="search"
        class="admin-toolbar-search max-w-full"
        :class="search ? 'w-60' : 'w-32 focus-within:w-60'"
        :data-pointer-opened="pointerOpened"
        @pointerdown.capture="pointerOpened = true"
        @keydown.capture="pointerOpened = false"
        @focusout="pointerOpened = false"
        :placeholder="i18n._('Search')"
        :clear-label="i18n._('Clear search')"
      />
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import SearchInput from '@shared-ui/components/SearchInput.vue'

defineProps({ searchable: { type: Boolean, default: false } })
const search = defineModel('search', { type: String, default: '' })
const pointerOpened = ref(false)
</script>

<style scoped>
@media (prefers-reduced-motion: no-preference) and (hover: hover) and (pointer: fine) {
  .admin-toolbar-search[data-pointer-opened='true'] {
    transition: width 150ms cubic-bezier(0.23, 1, 0.32, 1);
  }
}
</style>
