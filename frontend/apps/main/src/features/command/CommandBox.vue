<template>
  <CommandDialog
    :open="open"
    v-model:search-term="searchTerm"
    v-model:selected-value="highlightedValue"
    :filter-function="filterFunction"
    @update:open="onOpenChange"
    :class="[
      'z-[51] !top-[44%] !w-[calc(100%-1.5rem)] !min-w-0 gap-0 rounded-lg border-border/80 bg-popover shadow-lg [&>button]:right-3 [&>button]:top-3 [&>button]:rounded-md [&>button]:bg-muted/70 [&>button]:opacity-60 [&>button]:hover:opacity-100',
      isMacroMode ? '!max-w-5xl' : '!max-w-2xl'
    ]"
    command-class="rounded-lg bg-popover [&_[cmdk-input-wrapper]]:h-14 [&_[cmdk-input-wrapper]]:border-border/70 [&_[cmdk-input-wrapper]]:px-4 [&_[cmdk-input-wrapper]_svg]:text-muted-foreground [&_[cmdk-input]]:h-14 [&_[cmdk-input]]:pr-10 [&_[cmdk-input]]:text-base [&_[cmdk-group]]:py-2 [&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:pb-2 [&_[cmdk-group-heading]]:pt-1 [&_[cmdk-group-heading]]:text-xs [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-wider [&_[cmdk-item]]:mx-0.5 [&_[cmdk-item]]:rounded-md [&_[cmdk-item]]:px-3 [&_[cmdk-item]]:py-2 [&_[cmdk-item]]:transition-colors [&_[cmdk-item]]:duration-150"
  >
    <div
      v-if="breadcrumb"
      class="flex items-center gap-2 border-b border-border/70 px-4 py-2 text-xs text-muted-foreground"
    >
      <button
        type="button"
        class="flex items-center gap-1 rounded-md px-1.5 py-0.5 hover:bg-muted hover:text-foreground"
        @click="goBack"
      >
        <ChevronLeft class="h-3.5 w-3.5" />
        {{ $t('globals.messages.back') }}
      </button>
      <span class="rounded-md bg-muted px-2 py-0.5 font-medium text-foreground">
        {{ breadcrumb }}
      </span>
    </div>

    <CommandInput :placeholder="placeholder" :loading="loading" @keydown="onInputKeydown" />
    <CommandList
      :key="parent || 'root'"
      :class="
        isMacroMode
          ? 'h-[50vh] min-h-[50vh] min-w-[50vw] overflow-hidden [&>div]:h-full'
          : 'h-auto min-h-[220px] max-h-[min(52vh,440px)]'
      "
    >
      <CommandEmpty v-if="!loading">
        <p class="text-sm text-muted-foreground">{{ emptyText }}</p>
      </CommandEmpty>

      <MacroPicker
        v-if="isMacroMode"
        :search-term="searchTerm"
        :highlighted-value="highlightedValue"
        :macro-context="macroContext"
        @applied="closePalette"
      />

      <template v-else>
        <CommandGroup
          v-for="group in groups"
          :key="group.section"
          :heading="group.label"
        >
          <CommandItem
            v-for="command in group.commands"
            :key="command.id"
            :value="command.id"
            :class="['gap-2.5', { 'text-destructive': command.destructive }]"
            @select="onSelect($event, command)"
          >
            <component
              :is="command.icon"
              v-if="command.icon"
              class="!h-4 !w-4 shrink-0 text-muted-foreground"
            />
            <span class="min-w-0 truncate">{{ command.label }}</span>
            <span v-if="command.hint" class="min-w-0 truncate text-xs text-muted-foreground">
              {{ command.hint }}
            </span>
            <span class="ml-auto flex shrink-0 items-center gap-1 pl-3">
              <template v-if="command.shortcut">
                <kbd v-for="key in command.shortcut" :key="key" :class="KBD_CLASS">
                  {{ KEY_LABELS[key] || key }}
                </kbd>
              </template>
              <ChevronRight
                v-if="command.group || command.navigateTo"
                class="!h-4 !w-4 text-muted-foreground"
              />
            </span>
          </CommandItem>
        </CommandGroup>
      </template>
    </CommandList>

    <div
      class="flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-border/70 bg-muted/30 px-4 py-2"
    >
      <span class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd :class="KBD_CLASS">Enter</kbd>
        {{ $t('globals.terms.select') }}
      </span>
      <span class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd :class="KBD_CLASS">&uarr;&darr;</kbd>
        {{ $t('command.navigate') }}
      </span>
      <span class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd :class="KBD_CLASS">Esc</kbd>
        {{ $t('globals.messages.close') }}
      </span>
      <span v-if="parent" class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd :class="KBD_CLASS">Backspace</kbd>
        {{ $t('globals.messages.back') }}
      </span>
    </div>
  </CommandDialog>

  <SnoozeDatePicker v-model:open="showSnoozeDatePicker" />
</template>

<script setup>
const KBD_CLASS =
  'inline-flex h-5 items-center rounded-sm border bg-background px-1.5 font-mono text-xs font-medium text-foreground shadow-sm'
const ASYNC_SEARCH_DEBOUNCE = 250

import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@shared-ui/components/ui/command'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import MacroPicker from './MacroPicker.vue'
import SnoozeDatePicker from './SnoozeDatePicker.vue'
import { MACROS_COMMAND, useCommandPalette } from './useCommandPalette'
import { useCommandRegistry, commandMatches } from './useCommandRegistry'
import { useGlobalShortcuts } from './useGlobalShortcuts'
import { SECTION_ORDER, SECTION_LABEL_KEYS, SECTION_LABEL_PLURAL } from './sections'
import { useNavigationCommands } from './providers/useNavigationCommands'
import { useCreateCommands } from './providers/useCreateCommands'
import { useAccountCommands } from './providers/useAccountCommands'
import { useConversationCommands } from './providers/useConversationCommands'
import { useListCommands } from './providers/useListCommands'
import { useBulkCommands } from './providers/useBulkCommands'
import { useContactCommands } from './providers/useContactCommands'
import { useEntitySearch, ENTITY_SEARCH_MIN_LENGTH } from './providers/useEntitySearch'

const { t } = useI18n()
const emitter = useEmitter()
const palette = useCommandPalette()
const { open, parent, searchTerm, macroContext, closePalette, setParent } = palette
const showSnoozeDatePicker = ref(false)
const highlightedValue = ref('')
const isMac = /Mac|iPhone|iPad/.test(navigator.platform)
const KEY_LABELS = isMac ? { Ctrl: '⌘', Alt: '⌥' } : {}

useGlobalShortcuts()

const openSnoozeDatePicker = () => {
  showSnoozeDatePicker.value = true
}

const registry = useCommandRegistry([
  useBulkCommands(),
  useConversationCommands({ openSnoozeDatePicker }),
  useContactCommands(),
  useListCommands(),
  useCreateCommands(),
  useNavigationCommands(),
  useAccountCommands()
])
const entitySearch = useEntitySearch()

const isMacroMode = computed(() => parent.value === MACROS_COMMAND)
const parentCommand = computed(() => registry.get(parent.value))
const breadcrumb = computed(() =>
  isMacroMode.value ? t('globals.terms.macro', 2) : parentCommand.value?.label || ''
)

const placeholder = computed(() => {
  if (isMacroMode.value) return t('command.searchMacros')
  return parentCommand.value?.placeholder || t('command.searchOrJumpTo')
})

// Results of the current async source: a group's own search, or the root entity search.
const asyncResults = ref([])
const loading = ref(false)
let searchSeq = 0

const asyncSource = () => {
  const term = searchTerm.value.trim()
  const group = parentCommand.value
  if (group?.search) return () => group.search(term)
  if (!parent.value && term.length >= ENTITY_SEARCH_MIN_LENGTH) {
    return () => entitySearch.search(term)
  }
  return null
}

const runAsyncSearch = async () => {
  const seq = ++searchSeq
  const source = asyncSource()
  if (!source) {
    asyncResults.value = []
    loading.value = false
    return
  }
  loading.value = true
  try {
    const results = await source()
    if (seq !== searchSeq) return
    asyncResults.value = results
  } catch (error) {
    if (seq !== searchSeq) return
    asyncResults.value = []
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    if (seq === searchSeq) loading.value = false
  }
}
const runAsyncSearchDebounced = useDebounceFn(runAsyncSearch, ASYNC_SEARCH_DEBOUNCE)

watch(
  () => [open.value, parent.value],
  ([isOpen]) => {
    searchSeq++
    asyncResults.value = []
    loading.value = false
    if (isOpen && !isMacroMode.value) runAsyncSearch()
  }
)
watch(
  () => searchTerm.value,
  () => {
    if (!open.value || isMacroMode.value) return
    // The request only fires after the debounce, stale results would stay selectable until then.
    searchSeq++
    asyncResults.value = []
    loading.value = Boolean(asyncSource())
    runAsyncSearchDebounced()
  }
)

const visibleCommands = computed(() => {
  if (isMacroMode.value) return []
  if (parentCommand.value?.search) return asyncResults.value
  return [...registry.childrenOf(parent.value), ...asyncResults.value]
})

const visibleById = computed(() => new Map(visibleCommands.value.map((c) => [c.id, c])))
const asyncIds = computed(() => new Set(asyncResults.value.map((c) => c.id)))

const groups = computed(() => {
  if (parent.value) {
    return visibleCommands.value.length
      ? [{ section: 'children', label: undefined, commands: visibleCommands.value }]
      : []
  }
  return SECTION_ORDER.map((section) => ({
    section,
    label: t(SECTION_LABEL_KEYS[section], SECTION_LABEL_PLURAL.has(section) ? 2 : 1),
    commands: visibleCommands.value.filter((c) => c.section === section)
  })).filter((group) => group.commands.length)
})

// Async results are already filtered by the server, static commands match on label and keywords.
const filterFunction = (values, term) => {
  if (isMacroMode.value) return values
  return values.filter((value) => {
    if (asyncIds.value.has(value)) return true
    const command = visibleById.value.get(value)
    return command ? commandMatches(command, term) : false
  })
}

const emptyText = computed(() =>
  !parent.value && searchTerm.value.trim().length < ENTITY_SEARCH_MIN_LENGTH
    ? t('command.noCommandAvailable')
    : t('globals.messages.noResultsFound')
)

const onSelect = async (event, command) => {
  // Without this radix writes the selected value into the search input.
  event.preventDefault()
  if (command.navigateTo) return setParent(command.navigateTo)
  if (command.group) return setParent(command.id)
  closePalette()
  await command.run?.()
}

const goBack = () => {
  if (isMacroMode.value) return setParent(null)
  setParent(parentCommand.value?.parent || null)
}

const onOpenChange = (isOpen) => {
  if (!isOpen) closePalette()
}

const onInputKeydown = (event) => {
  if (event.key === 'Backspace' && !event.target.value && parent.value) {
    event.preventDefault()
    goBack()
  }
}
</script>
