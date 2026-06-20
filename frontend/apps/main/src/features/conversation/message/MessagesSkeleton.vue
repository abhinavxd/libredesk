<template>
  <div class="py-2 space-y-5">
    <div v-for="(row, idx) in rows" :key="idx">
      <div
        v-if="row.showName"
        class="mb-1 flex items-center gap-2"
        :class="row.outgoing ? 'justify-end pr-[47px]' : 'pl-[47px]'"
      >
        <div class="skel-bar h-3 w-24 rounded-sm" />
        <div class="skel-bar h-3 w-12 rounded-sm" />
      </div>

      <div class="flex flex-row gap-2 w-full" :class="{ 'justify-end': row.outgoing }">
        <template v-if="!row.outgoing">
          <div v-if="row.showAvatar" class="skel-bar w-8 h-8 rounded-full flex-shrink-0" />
          <div v-else class="w-8 flex-shrink-0" />
        </template>

        <div class="w-4/5" :class="{ 'flex justify-end': row.outgoing }">
          <div
            class="skel-bubble flex flex-col gap-2 px-4 pt-3 pb-3 rounded border border-border shadow-sm"
            :style="{ width: row.width }"
          >
            <div
              v-for="(lineW, n) in row.lines"
              :key="n"
              class="skel-bar h-3 rounded-sm"
              :style="{ width: lineW }"
            />
          </div>
        </div>

        <template v-if="row.outgoing">
          <div v-if="row.showAvatar" class="skel-bar w-8 h-8 rounded-full flex-shrink-0" />
          <div v-else class="w-8 flex-shrink-0" />
        </template>
      </div>

      <div v-if="row.showTime" class="mt-1" :class="row.outgoing ? 'pr-[47px]' : 'pl-[47px]'">
        <div class="skel-bar h-2.5 w-16 rounded-sm" :class="{ 'ml-auto': row.outgoing }" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  count: { type: Number, default: 8 },
})

const pattern = [
  { outgoing: false, showAvatar: true, showName: true, width: '58%', lines: ['100%', '78%'], showTime: true },
  { outgoing: false, showAvatar: false, showName: false, width: '42%', lines: ['100%'], showTime: false },
  { outgoing: true, showAvatar: true, showName: true, width: '66%', lines: ['100%', '92%', '60%'], showTime: true },
  { outgoing: false, showAvatar: true, showName: true, width: '72%', lines: ['100%', '85%'], showTime: true },
  { outgoing: true, showAvatar: true, showName: true, width: '50%', lines: ['100%'], showTime: true },
  { outgoing: true, showAvatar: false, showName: false, width: '38%', lines: ['100%'], showTime: false },
  { outgoing: false, showAvatar: true, showName: true, width: '60%', lines: ['100%', '70%'], showTime: true },
  { outgoing: true, showAvatar: true, showName: true, width: '54%', lines: ['100%', '88%'], showTime: true },
]

const rows = computed(() => {
  const out = []
  for (let i = 0; i < props.count; i++) {
    out.push(pattern[i % pattern.length])
  }
  return out
})
</script>

<style scoped>
.skel-bar {
  background-color: hsl(var(--muted));
  animation: skel-pulse 2.4s ease-in-out infinite;
}
.skel-bubble {
  background-color: hsl(var(--muted) / 0.35);
}
@keyframes skel-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}
</style>
