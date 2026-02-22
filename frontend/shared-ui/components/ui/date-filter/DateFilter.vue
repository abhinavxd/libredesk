<template>
  <div class="flex items-center gap-2">
    <Select v-model="selectedDays" @update:model-value="handleFilterChange">
      <SelectTrigger class="w-[140px] h-8 text-xs">
        <SelectValue
          :placeholder="
            t('dateFilter.selectDays')
          "
        />
      </SelectTrigger>
      <SelectContent class="text-xs">
        <SelectItem value="0">{{ $t('globals.terms.today') }}</SelectItem>
        <SelectItem value="1">
          {{
            $t('dateFilter.lastNDays', { n: 1 })
          }}
        </SelectItem>
        <SelectItem value="2">
          {{
            $t('dateFilter.lastNDays', { n: 2 })
          }}
        </SelectItem>
        <SelectItem value="7">
          {{
            $t('dateFilter.lastNDays', { n: 7 })
          }}
        </SelectItem>
        <SelectItem value="30">
          {{
            $t('dateFilter.lastNDays', { n: 30 })
          }}
        </SelectItem>
        <SelectItem value="90">
          {{
            $t('dateFilter.lastNDays', { n: 90 })
          }}
        </SelectItem>
        <SelectItem value="custom">
          {{
            $t('dateFilter.custom')
          }}
        </SelectItem>
      </SelectContent>
    </Select>
    <div v-if="selectedDays === 'custom'" class="flex items-center gap-2">
      <Input
        v-model="customDaysInput"
        type="number"
        min="1"
        max="365"
        class="w-20 h-8"
        @blur="handleCustomDaysChange"
        @keyup.enter="handleCustomDaysChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '../select'
import { Input } from '../input'

const { t } = useI18n()

const emit = defineEmits(['filterChange'])
const selectedDays = ref('30')
const customDaysInput = ref('')

const handleFilterChange = (value) => {
  if (value === 'custom') {
    customDaysInput.value = '30'
    emit('filterChange', 30)
  } else {
    emit('filterChange', parseInt(value))
  }
}

const handleCustomDaysChange = () => {
  const days = parseInt(customDaysInput.value)
  if (days && days > 0 && days <= 365) {
    emit('filterChange', days)
  } else {
    customDaysInput.value = '30'
    emit('filterChange', 30)
  }
}

handleFilterChange(selectedDays.value)
</script>
