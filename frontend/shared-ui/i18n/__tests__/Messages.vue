<script setup>
import { computed } from 'vue'
import { useI18n } from '../index'

const props = defineProps({ name: String, count: Number })
const i18n = useI18n()
const greeting = computed(() => i18n._('Hello {name}', { name: props.name }))
</script>

<template>
  <p>{{ greeting }}</p>
  <p>{{ i18n.ngettext('# file', '# files', count) }}</p>
  <button>{{ i18n.pgettext('button', 'Open') }}</button>
</template>
