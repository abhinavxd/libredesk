<template>
  <div class="flex items-start gap-2 px-3 py-3 border-b border-border">
    <div class="relative flex-shrink-0 w-10 h-10">
      <div class="skel-bar h-10 w-10 rounded-full" />
      <span class="absolute -bottom-0.5 -right-0.5 w-4 h-4 rounded-full bg-background border border-border flex items-center justify-center">
        <span class="skel-bar w-2 h-2 rounded-full" />
      </span>
    </div>

    <div class="flex-1 min-w-0 space-y-2 pt-0.5">
      <div class="flex items-baseline justify-between gap-2">
        <div class="flex items-baseline gap-1.5 min-w-0">
          <div class="skel-bar h-3.5 rounded-sm" :style="{ width: nameW }" />
          <div class="skel-bar h-3 rounded-sm" :style="{ width: inboxW }" />
        </div>
        <div class="skel-bar h-3 w-9 rounded-sm flex-shrink-0" />
      </div>

      <div v-if="hasSubject" class="skel-bar h-3 rounded-sm" :style="{ width: subjectW }" />

      <div class="flex items-center justify-between gap-2 pt-0.5">
        <div class="skel-bar h-3 rounded-sm flex-1" :style="{ maxWidth: previewW }" />
        <div v-if="hasUnread" class="skel-bar h-5 w-5 rounded-full flex-shrink-0" />
      </div>

      <div v-if="slaBadges.length" class="flex items-center gap-1 pt-0.5">
        <span
          v-for="(w, n) in slaBadges"
          :key="n"
          class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded border border-border"
        >
          <span class="skel-bar w-2.5 h-2.5 rounded-full" />
          <span class="skel-bar h-2 rounded-sm" :style="{ width: w }" />
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  index: { type: Number, default: 0 },
})

const nameWidths = ['96px', '128px', '108px', '142px', '88px']
const inboxWidths = ['52px', '38px', '60px', '46px', '54px']
const subjectWidths = ['72%', '88%', '58%', '92%', '66%']
const previewWidths = ['100%', '78%', '92%', '64%', '86%']
const subjectFlags = [true, false, true, true, false]
const unreadFlags = [true, false, false, true, false]
const slaPatterns = [['28px'], [], ['24px', '32px'], [], ['30px']]

const i = computed(() => props.index % 5)
const nameW = computed(() => nameWidths[i.value])
const inboxW = computed(() => inboxWidths[i.value])
const subjectW = computed(() => subjectWidths[i.value])
const previewW = computed(() => previewWidths[i.value])
const hasSubject = computed(() => subjectFlags[i.value])
const hasUnread = computed(() => unreadFlags[i.value])
const slaBadges = computed(() => slaPatterns[i.value])
</script>

<style scoped>
.skel-bar {
  background-color: hsl(var(--muted));
  animation: skel-pulse 2.4s ease-in-out infinite;
}
@keyframes skel-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}
</style>
