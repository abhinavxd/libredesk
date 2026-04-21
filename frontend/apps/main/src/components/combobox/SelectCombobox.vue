<template>
  <ComboBox
    :model-value="normalizedValue"
    @update:model-value="$emit('update:modelValue', $event)"
    :items="items"
    :placeholder="placeholder"
  >
    <!-- Items -->
    <template #item="{ item }">
      <div class="flex items-center gap-2">
        <!--USER -->
        <Avatar v-if="type === 'user'" class="w-7 h-7">
          <AvatarImage :src="item.avatar_url || ''" :alt="item.label.slice(0, 2)" />
          <AvatarFallback>{{ item.label.slice(0, 2).toUpperCase() }}</AvatarFallback>
        </Avatar>

        <!-- Others -->
        <span v-else-if="item.emoji">{{ item.emoji }}</span>
        <span>{{ item.label }}</span>
        <span v-if="isCurrentUser(item)" class="text-muted-foreground text-xs"
          >({{ t('globals.terms.you') }})</span
        >
      </div>
    </template>

    <!-- Selected -->
    <template #selected="{ selected }">
      <div class="flex items-center gap-2 min-w-0">
        <div v-if="selected" class="flex items-center gap-2 min-w-0">
          <!--USER -->
          <Avatar v-if="type === 'user'" class="w-7 h-7 shrink-0">
            <AvatarImage :src="selected.avatar_url || ''" :alt="selected.label.slice(0, 2)" />
            <AvatarFallback>{{ selected.label.slice(0, 2).toUpperCase() }}</AvatarFallback>
          </Avatar>

          <!-- Others -->
          <span v-else-if="selected.emoji" class="shrink-0">{{ selected.emoji }}</span>
          <span class="truncate">{{ selected.label }}</span>
          <span
            v-if="isCurrentUser(selected)"
            class="text-muted-foreground text-xs shrink-0"
            >({{ t('globals.terms.you') }})</span
          >
        </div>
        <span v-else class="truncate">{{ placeholder }}</span>
      </div>
    </template>
  </ComboBox>
</template>

<script setup>
import { computed } from 'vue'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'
import { useUserStore } from '@/stores/user'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
const userStore = useUserStore()

const props = defineProps({
  modelValue: [String, Number, Object],
  placeholder: String,
  items: Array,
  type: {
    type: String
  }
})

// Convert to str.
const normalizedValue = computed(() => String(props.modelValue || ''))

const isCurrentUser = (item) => {
  return (
    props.type === 'user' &&
    item.value !== 'none' &&
    String(item.value) === String(userStore.userID)
  )
}

defineEmits(['update:modelValue'])
</script>
