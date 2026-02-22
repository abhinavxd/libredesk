<template>
  <Breadcrumb>
    <BreadcrumbList>
      <BreadcrumbItem v-for="(item, index) in links" :key="index">
        <router-link :to="{ name: item.path }" v-if="item.path">
          {{ item.label }}
        </router-link>
        <span v-else>{{ item.label }}</span>
        <BreadcrumbSeparator v-if="index < links.length - 1">
          <ChevronRight />
        </BreadcrumbSeparator>
      </BreadcrumbItem>
    </BreadcrumbList>
  </Breadcrumb>
</template>

<script setup>
import { ChevronRight } from 'lucide-vue-next'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbSeparator
} from '.'

defineProps({
  links: {
    type: Array,
    required: true,
    default: () => []
  }
})
</script>
