<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button type="button" variant="outline" class="w-full justify-start gap-2 font-normal">
        <svg
          v-if="model"
          class="size-4 shrink-0"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <use :href="`#${model}`" />
        </svg>
        <span v-if="model" class="truncate">{{ model }}</span>
        <span v-else class="text-muted-foreground">{{ t('globals.terms.none') }}</span>
      </Button>
    </PopoverTrigger>
    <PopoverContent class="w-72 p-3" align="start">
      <Input v-model="search" :placeholder="t('globals.terms.search')" class="h-8" />
      <div class="mt-2 h-56 overflow-y-auto">
        <div class="grid grid-cols-7 gap-1">
          <button
            v-for="name in filteredIcons"
            :key="name"
            type="button"
            :title="name"
            class="flex size-8 shrink-0 items-center justify-center rounded-md hover:bg-accent"
            :class="{ 'bg-accent': name === model }"
            @click="select(name)"
          >
            <svg
              class="size-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <use :href="`#${name}`" />
            </svg>
          </button>
        </div>
        <p
          v-if="iconNames.length && !filteredIcons.length"
          class="py-4 text-center text-sm text-muted-foreground"
        >
          {{ t('globals.messages.noResultsFound') }}
        </p>
        <p v-if="loadError" class="py-4 text-center text-sm text-muted-foreground">
          {{ t('globals.messages.somethingWentWrong') }}
        </p>
      </div>
      <Button
        v-if="model"
        type="button"
        variant="ghost"
        size="sm"
        class="mt-2 w-full"
        @click="select('')"
      >
        {{ t('globals.terms.remove') }}
      </Button>
    </PopoverContent>
  </Popover>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover/index.js'
import { loadLucideSprite } from './lucideSprite.js'

const { t } = useI18n()
const model = defineModel({ type: String, default: '' })

const open = ref(false)
const search = ref('')
const iconNames = ref([])
const loadError = ref(false)

onMounted(async () => {
  try {
    iconNames.value = await loadLucideSprite()
  } catch {
    loadError.value = true
  }
})

const filteredIcons = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return iconNames.value
  return iconNames.value.filter((name) => name.includes(q))
})

const select = (name) => {
  model.value = name
  open.value = false
}
</script>
