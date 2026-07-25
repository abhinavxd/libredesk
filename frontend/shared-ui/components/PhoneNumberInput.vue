<template>
  <div class="flex items-end">
    <FormField v-slot="{ componentField }" :name="countryCodeName">
      <FormItem class="w-max">
        <FormLabel class="flex items-center whitespace-nowrap">
          {{ label || t('globals.terms.phoneNumber') }}
          <span v-if="required" class="text-destructive">*</span>
        </FormLabel>
        <FormControl>
          <ComboBox
            v-bind="componentField"
            :items="allCountries"
            :placeholder="t('globals.terms.select')"
            :buttonClass="'rounded-r-none border-r-0'"
          >
            <template #item="{ item }">
              <div class="flex items-center gap-2">
                <div class="w-7 h-7 flex items-center justify-center">
                  <span v-if="item.emoji">{{ item.emoji }}</span>
                </div>
                <span class="text-sm">{{ item.label }} ({{ item.calling_code }})</span>
              </div>
            </template>

            <template #selected="{ selected }">
              <div class="flex items-center gap-1">
                <span v-if="selected" class="text-lg">{{ selected.emoji }}</span>
                <span v-if="selected && selected.calling_code" class="text-xs text-muted-foreground"
                  >({{ selected.calling_code }})</span
                >
              </div>
            </template>
          </ComboBox>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="flex-1">
      <FormField v-slot="{ componentField }" :name="phoneNumberName">
        <FormItem class="relative">
          <FormControl>
            <Input
            type="tel"
            v-bind="componentField"
            :placeholder="placeholder"
            class="rounded-l-none"
            inputmode="numeric"
          />
            <FormMessage class="absolute top-full mt-1 text-sm" />
          </FormControl>
        </FormItem>
      </FormField>
    </div>
  </div>
</template>

<script setup>
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from './ui/form'
import { Input } from './ui/input'
import ComboBox from './ui/combobox/ComboBox.vue'
import { countryCallingOptions as allCountries } from '../constants/countries.js'
import { useI18n } from 'vue-i18n'

defineProps({
  countryCodeName: { type: String, default: 'phone_number_country_code' },
  phoneNumberName: { type: String, default: 'phone_number' },
  label: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  required: { type: Boolean, default: false }
})

const { t } = useI18n()
</script>
