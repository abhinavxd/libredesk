<template>
  <div class="space-y-1 min-w-0">
    <p v-if="label" class="text-xs text-muted-foreground">{{ label }}</p>
    <select
      class="w-full rounded-md border bg-background px-2 py-1.5 text-sm"
      :value="modelValue || ''"
      :disabled="disabled || saving"
      @change="onChange"
    >
      <option value="">{{ t('organization.none') }}</option>
      <option v-for="org in orgs" :key="org.id" :value="org.id">{{ org.name }}</option>
    </select>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '../../api/index.js'

const props = defineProps({
  modelValue: { type: [Number, String, null], default: null },
  label: { type: String, default: '' },
  disabled: { type: Boolean, default: false }
})
const emit = defineEmits(['update:modelValue', 'change'])

const { t } = useI18n()
const orgs = ref([])
const saving = ref(false)

onMounted(async () => {
  try {
    const resp = await api.getOrganizations()
    orgs.value = resp.data.data || []
  } catch {
    orgs.value = []
  }
})

function onChange(e) {
  const raw = e.target.value
  const value = raw === '' ? null : Number(raw)
  emit('update:modelValue', value)
  emit('change', value)
}
</script>
