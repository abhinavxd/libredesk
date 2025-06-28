<script setup>
import { VisTooltip } from '@unovis/vue'
import { omit } from '@unovis/ts'
import { createApp } from 'vue'
import { ChartTooltip } from '.'

const props = defineProps({
  selector: { type: String, required: true },
  index: { type: String, required: true },
  items: { type: Array, required: false },
  valueFormatter: { type: Function, required: false, default: (tick) => `${tick}` },
  customTooltip: { type: null, required: false }
})

// Use weakmap to store reference to each datapoint for Tooltip
const wm = new WeakMap()
function template(d, i, elements) {
  if (props.index in d) {
    if (wm.has(d)) {
      return wm.get(d)
    } else {
      const componentDiv = document.createElement('div')
      const omittedData = Object.entries(omit(d, [props.index])).map(([key, value]) => {
        const legendReference = props.items?.find((i) => i.name === key)
        return { ...legendReference, value: props.valueFormatter(value) }
      })
      const TooltipComponent = props.customTooltip ?? ChartTooltip
      createApp(TooltipComponent, { title: d[props.index], data: omittedData }).mount(componentDiv)
      wm.set(d, componentDiv.innerHTML)
      return componentDiv.innerHTML
    }
  } else {
    const data = d.data

    if (wm.has(data)) {
      return wm.get(data)
    } else {
      const style = getComputedStyle(elements[i])
      const omittedData = [
        { name: data.name, value: props.valueFormatter(data[props.index]), color: style.fill }
      ]
      const componentDiv = document.createElement('div')
      const TooltipComponent = props.customTooltip ?? ChartTooltip
      createApp(TooltipComponent, { title: d[props.index], data: omittedData }).mount(componentDiv)
      wm.set(d, componentDiv.innerHTML)
      return componentDiv.innerHTML
    }
  }
}
</script>

<template>
  <VisTooltip
    :horizontal-shift="20"
    :vertical-shift="20"
    :triggers="{
      [selector]: template
    }"
  />
</template>
