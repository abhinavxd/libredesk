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
      </div>
    </template>

    <!-- Selected -->
    <template #selected="{ selected }">
      <div class="flex items-center gap-2">
        <div v-if="selected" class="flex items-center gap-2">
          <!--USER -->
          <Avatar v-if="type === 'user'" class="w-7 h-7">
            <AvatarImage :src="selected.avatar_url || ''" :alt="selected.label.slice(0, 2)" />
            <AvatarFallback>{{ selected.label.slice(0, 2).toUpperCase() }}</AvatarFallback>
          </Avatar>

          <!-- Others -->
          <span v-else-if="selected.emoji">{{ selected.emoji }}</span>
          <span>{{ selected.label }}</span>
        </div>
        <span v-else>{{ placeholder }}</span>
      </div>
    </template>
  </ComboBox>
</template>

<script setup>
import { computed } from 'vue'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'

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

defineEmits(['update:modelValue'])
</script>
