<template>
  <div class="space-y-2">
    <div class="flex flex-wrap items-center gap-2">
      <div :class="FILTER_CLASS">
        <SelectComboBox
          :model-value="filters.status"
          :items="conversationStore.statusOptions"
          :placeholder="t('globals.terms.status')"
          align="start"
          @update:model-value="set('status', $event)"
        />
      </div>

      <div :class="FILTER_CLASS">
        <SelectComboBox
          :model-value="filters.address"
          :items="addressOptions"
          :placeholder="t('address.title', 2)"
          align="start"
          @update:model-value="set('address', $event)"
        />
      </div>

      <div class="flex-[1.5] min-w-60">
        <DateFilterValue
          :model-value="filters.created"
          :placeholder="t('globals.terms.conversationDate')"
          range
          @update:model-value="set('created', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'

import DateFilterValue from '@main/components/filter/DateFilterValue.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useAddressStore } from '@main/stores/address'

const FILTER_CLASS = 'flex-1 min-w-28'

const props = defineProps({
  filters: { type: Object, required: true }
})
const emit = defineEmits(['update:filters'])

const { t } = useI18n()
const conversationStore = useConversationStore()
const addressStore = useAddressStore()
const addressOptions = computed(() =>
  addressStore.addresses.map((address) => ({ label: address.address, value: String(address.id) }))
)

const set = (key, value) => {
  emit('update:filters', { ...props.filters, [key]: value ?? '' })
}

onMounted(() => {
  conversationStore.fetchStatuses()
  addressStore.fetchAddresses()
})
</script>
