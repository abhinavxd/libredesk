<template>
  <form class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Accounts" v-bind="componentField" />
        </FormControl>
        <FormDescription></FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="category_id">
      <FormItem>
        <FormLabel class="flex items-center whitespace-nowrap">
          {{ $t('globals.terms.category') }}
        </FormLabel>
        <FormControl>
          <ComboBox
            v-bind="componentField"
            :items="categories"
            :placeholder="$t('globals.messages.select', { name: $t('globals.terms.category') })"
          >
            <template #item="{ item }">
              <div class="flex items-center gap-2">
                <span class="text-sm">{{ item.name }}</span>
              </div>
            </template>

            <template #selected="{ selected }">
              <div class="flex items-center">
                <span v-if="selected">{{ selected.name }}</span>
              </div>
            </template>
          </ComboBox>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Form submit button slot -->
    <slot name="footer"></slot>
  </form>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'
import api from '@/api/index.js'

const categories = ref([])

// Fetch categories for the dropdown
const fetchCategories = async () => {
  try {
    const response = await api.getArticleCategories()
    categories.value =
      response?.data?.data?.map((category) => ({
        value: category.id,
        name: category.name
      })) ?? []
  } catch (error) {
    console.error('Error fetching categories:', error)
  }
}

onMounted(() => {
  fetchCategories()
})
</script>
