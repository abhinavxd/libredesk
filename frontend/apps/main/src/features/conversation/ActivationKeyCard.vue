<template>
  <div
    v-if="info.enabled"
    class="flex items-center gap-2 rounded-md border border-dashed border-border px-1.5 py-1"
    :class="{ 'opacity-60': disabled }"
  >
    <button
      type="button"
      class="flex min-w-0 flex-1 items-center gap-2 rounded-sm px-1.5 py-1 text-left text-sm transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:hover:bg-transparent"
      :disabled="!canAttach"
      :aria-label="$t('keyDistribution.card.attach')"
      @click="attach"
    >
      <KeyRound class="size-4 shrink-0 text-primary" aria-hidden="true" />
      <span class="shrink-0 font-medium">{{ $t('keyDistribution.card.attach') }}</span>
      <span class="truncate text-xs text-muted-foreground">{{ status }}</span>
    </button>
    <NativeSelect
      v-model.number="selectedAppId"
      class="w-40 shrink-0 sm:w-48"
      :aria-label="$t('keyDistribution.card.game')"
      :disabled="disabled || !info.apps.length"
      @update:modelValue="pickApp"
    >
      <option v-if="!info.apps.length" :value="0">{{ $t('keyDistribution.card.noGames') }}</option>
      <option v-for="app in info.apps" :key="app.id" :value="app.id">
        {{ app.available ? app.name : $t('keyDistribution.card.gameNoKeys', { name: app.name }) }}
      </option>
    </NativeSelect>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { KeyRound } from 'lucide-vue-next'
import { NativeSelect } from '@shared-ui/components/ui/native-select'
import { useActivationKeys } from '@main/composables/useActivationKeys'
import { readKeyPlaceholders } from '@main/utils/activation-keys'

// The composer's "Attach Game Key" card. It never sees a key: clicking it
// drops a placeholder chip into the editor, and the server puts a key in its
// place only in the email it sends.
const props = defineProps({
  // The editor's current HTML, to count the chips already placed.
  html: { type: String, default: '' },
  // The ConversationEditor instance (insertActivationKey / setActivationKeyApp).
  editor: { type: Object, default: null },
  disabled: { type: Boolean, default: false }
})

const { t } = useI18n()
const { info, refresh, rememberApp } = useActivationKeys()
const selectedAppId = ref(0)

const placeholders = computed(() => readKeyPlaceholders(props.html))
const selectedApp = computed(() => info.value.apps.find((app) => app.id === selectedAppId.value))
const atLimit = computed(() => placeholders.value.length >= info.value.max_keys_per_email)
const canAttach = computed(
  () => !props.disabled && !!props.editor && !!selectedApp.value?.available && !atLimit.value
)

const status = computed(() => {
  if (!info.value.apps.length) return t('keyDistribution.card.noGames')
  if (selectedApp.value && !selectedApp.value.available) return t('keyDistribution.card.noKeys')
  return t('keyDistribution.card.count', {
    count: placeholders.value.length,
    max: info.value.max_keys_per_email
  })
})

// Chips already in the email decide the game; otherwise the last game this
// person picked, otherwise the first one that has keys.
const syncSelection = () => {
  const apps = info.value.apps
  const has = (id) => apps.some((app) => app.id === id)
  const chipApp = placeholders.value[0]?.appId
  if (chipApp && has(chipApp)) {
    selectedAppId.value = chipApp
  } else if (!has(selectedAppId.value)) {
    const last = info.value.last_app_id
    selectedAppId.value = has(last) ? last : (apps.find((app) => app.available) || apps[0])?.id || 0
  }
}

watch([info, () => placeholders.value[0]?.appId], syncSelection, { immediate: true })

// Fires only for the person's own choice, not when syncSelection moves it.
const pickApp = (value) => {
  const appId = Number(value)
  if (!appId) return
  rememberApp(appId)
  // One game per email.
  if (placeholders.value.length) props.editor?.setActivationKeyApp(appId)
}

const attach = () => {
  if (!canAttach.value) return
  props.editor.insertActivationKey(selectedAppId.value)
}

onMounted(refresh)
</script>
