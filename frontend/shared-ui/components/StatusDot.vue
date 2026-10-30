<template>
  <div
    class="inline-flex shrink-0 items-center justify-center rounded-full bg-background"
    :class="sizeClass"
    role="img"
    :aria-label="statusLabel"
  >
    <svg
      class="block shrink-0 fill-current"
      :class="iconClass"
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <path
        v-if="isOnline"
        d="M12 1.992Q9.288 1.992 6.96 3.36Q4.68 4.68 3.36 6.96Q1.992 9.288 1.992 12Q1.992 14.712 3.36 17.04Q4.68 19.32 6.96 20.64Q9.288 22.008 12 22.008Q14.712 22.008 17.04 20.64Q19.32 19.32 20.64 17.04Q22.008 14.712 22.008 12Q22.008 9.288 20.64 6.96Q19.32 4.68 17.04 3.36Q14.712 1.992 12 1.992ZM10.248 16.68 8.832 15.288 6 12.456 7.416 11.04 10.248 13.872 16.608 7.488 18.024 8.904Z"
        class="text-success"
      />
      <path
        v-else-if="isAway"
        d="M12 1.992Q9.288 1.992 6.96 3.36Q4.704 4.704 3.36 6.96Q1.992 9.288 1.992 12Q1.992 14.712 3.36 17.04Q4.704 19.296 6.96 20.64Q9.288 22.008 12 22.008Q14.712 22.008 17.04 20.64Q19.296 19.296 20.64 17.04Q22.008 14.712 22.008 12Q22.008 9.288 20.64 6.96Q19.296 4.704 17.04 3.36Q14.712 1.992 12 1.992ZM15.528 16.944 10.992 12.408V7.008H13.008V11.592L16.944 15.528Z"
        class="text-warning"
      />
      <path
        v-else
        d="M12 19.992Q9.84 19.992 7.968 18.912Q6.168 17.832 5.088 16.032Q4.008 14.16 4.008 12Q4.008 9.84 5.088 7.968Q6.168 6.168 7.968 5.088Q9.84 4.008 12 4.008Q14.16 4.008 16.032 5.088Q17.832 6.168 18.912 7.968Q19.992 9.84 19.992 12Q19.992 14.16 18.912 16.032Q17.832 17.832 16.032 18.912Q14.16 19.992 12 19.992ZM12 1.992Q9.288 1.992 6.96 3.36Q4.68 4.68 3.36 6.96Q1.992 9.288 1.992 12Q1.992 14.712 3.36 17.04Q4.68 19.32 6.96 20.64Q9.288 22.008 12 22.008Q14.712 22.008 17.04 20.64Q19.32 19.32 20.64 17.04Q22.008 14.712 22.008 12Q22.008 9.288 20.64 6.96Q19.32 4.68 17.04 3.36Q14.712 1.992 12 1.992Z"
        class="text-muted-foreground"
      />
    </svg>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  status: {
    type: String,
    default: 'offline'
  },
  size: {
    type: String,
    default: 'md',
    validator: (value) => ['sm', 'md', 'lg'].includes(value)
  }
})

const { t } = useI18n()

const isOnline = computed(() => ['online', 'away'].includes(props.status))

const isAway = computed(() => ['away_manual', 'away_and_reassigning'].includes(props.status))

const statusLabel = computed(() =>
  t(
    isOnline.value
      ? 'globals.terms.online'
      : isAway.value
        ? 'globals.terms.away'
        : 'globals.terms.offline',
    1
  )
)

const iconClass = computed(() => (props.size === 'lg' ? 'size-[13px]' : 'size-3'))

const sizeClass = computed(() => {
  const sizes = {
    sm: 'size-[13px]',
    md: 'size-[15px]',
    lg: 'size-[19px]'
  }
  return sizes[props.size]
})
</script>
