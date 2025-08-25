<script setup>
import {
  ChartCrosshair,
  ChartLegend,
  defaultColors,
} from '../chart';
import { cn } from '../../../lib/utils';
import { Axis, GroupedBar, StackedBar } from '@unovis/ts';
import {
  VisAxis,
  VisGroupedBar,
  VisStackedBar,
  VisXYContainer,
} from '@unovis/vue';
import { useMounted } from '@vueuse/core';
import { computed, ref } from 'vue';

const props = defineProps({
  data: { type: Array, required: true },
  categories: { type: Array, required: true },
  index: { type: null, required: true },
  colors: { type: Array, required: false },
  margin: {
    type: Object,
    required: false,
    default: () => ({ top: 0, bottom: 0, left: 0, right: 0 }),
  },
  filterOpacity: { type: Number, required: false, default: 0.2 },
  xFormatter: { type: Function, required: false },
  yFormatter: { type: Function, required: false },
  showXAxis: { type: Boolean, required: false, default: true },
  showYAxis: { type: Boolean, required: false, default: true },
  showTooltip: { type: Boolean, required: false, default: true },
  showLegend: { type: Boolean, required: false, default: true },
  showGridLine: { type: Boolean, required: false, default: true },
  customTooltip: { type: null, required: false },
  type: { type: String, required: false, default: 'grouped' },
  roundedCorners: { type: Number, required: false, default: 0 },
});
const emits = defineEmits(['legendItemClick']);

const index = computed(() => props.index);
const colors = computed(() =>
  props.colors?.length ? props.colors : defaultColors(props.categories.length),
);
const legendItems = ref(
  props.categories.map((category, i) => ({
    name: category,
    color: colors.value[i],
    inactive: false,
  })),
);

const isMounted = useMounted();

function handleLegendItemClick(d, i) {
  emits('legendItemClick', d, i);
}

const VisBarComponent = computed(() =>
  props.type === 'grouped' ? VisGroupedBar : VisStackedBar,
);
const selectorsBar = computed(() =>
  props.type === 'grouped'
    ? GroupedBar.selectors.bar
    : StackedBar.selectors.bar,
);
</script>

<template>
  <div
    :class="cn('w-full h-[400px] flex flex-col items-end', $attrs.class ?? '')"
  >
    <ChartLegend
      v-if="showLegend"
      v-model:items="legendItems"
      @legend-item-click="handleLegendItemClick"
    />

    <VisXYContainer
      :data="data"
      :style="{ height: isMounted ? '100%' : 'auto' }"
      :margin="margin"
    >
      <ChartCrosshair
        v-if="showTooltip"
        :colors="colors"
        :items="legendItems"
        :custom-tooltip="customTooltip"
        :index="index"
      />

      <VisBarComponent
        :x="(d, i) => i"
        :y="categories.map((category) => (d) => d[category])"
        :color="colors"
        :rounded-corners="roundedCorners"
        :bar-padding="0.05"
        :attributes="{
          [selectorsBar]: {
            opacity: (d, i) => {
              const pos = i % categories.length;
              return legendItems[pos]?.inactive ? filterOpacity : 1;
            },
          },
        }"
      />

      <VisAxis
        v-if="showXAxis"
        type="x"
        :tick-format="xFormatter ?? ((v) => data[v]?.[index])"
        :grid-line="false"
        :tick-line="false"
        tick-text-color="hsl(var(--vis-text-color))"
      />
      <VisAxis
        v-if="showYAxis"
        type="y"
        :tick-line="false"
        :tick-format="yFormatter"
        :domain-line="false"
        :grid-line="showGridLine"
        :attributes="{
          [Axis.selectors.grid]: {
            class: 'text-muted',
          },
        }"
        tick-text-color="hsl(var(--vis-text-color))"
      />

      <slot />
    </VisXYContainer>
  </div>
</template>
