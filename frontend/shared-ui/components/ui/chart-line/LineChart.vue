<script setup>
import {
  ChartCrosshair,
  ChartLegend,
  defaultColors,
} from '../chart';
import { cn } from '../../../lib/utils';
import { CurveType } from '@unovis/ts';
import { Axis, Line } from '@unovis/ts';
import { VisAxis, VisLine, VisXYContainer } from '@unovis/vue';
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
  curveType: { type: String, required: false, default: CurveType.MonotoneX },
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
      :margin="{ left: 20, right: 20 }"
      :data="data"
      :style="{ height: isMounted ? '100%' : 'auto' }"
    >
      <ChartCrosshair
        v-if="showTooltip"
        :colors="colors"
        :items="legendItems"
        :index="index"
        :custom-tooltip="customTooltip"
      />

      <template v-for="(category, i) in categories" :key="category">
        <VisLine
          :x="(d, i) => i"
          :y="(d) => d[category]"
          :curve-type="curveType"
          :color="colors[i]"
          :attributes="{
            [Line.selectors.line]: {
              opacity: legendItems.find((item) => item.name === category)
                ?.inactive
                ? filterOpacity
                : 1,
            },
          }"
        />
      </template>

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
