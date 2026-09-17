<template>
  <MultiSelectComboBox
    v-model="modelValue"
    :items="tagStore.tagOptions"
    :placeholder="t('placeholders.selectTags')"
    :selected-label="t('globals.terms.tag', 2)"
    :search="tagStore.searchTagOptions"
  />
</template>

<script setup>
import { onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import MultiSelectComboBox from '@shared-ui/components/ui/combobox/MultiSelectComboBox.vue'
import { useTagStore } from '@main/stores/tag'

const modelValue = defineModel({ type: Array, default: () => [] })

const { t } = useI18n()
const tagStore = useTagStore()

watch(
  modelValue,
  (ids) => {
    tagStore.ensureTagIDs(ids)
  },
  { immediate: true }
)

onMounted(tagStore.fetchTags)
</script>
