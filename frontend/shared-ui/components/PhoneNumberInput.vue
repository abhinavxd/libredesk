<template>
  <FormField v-slot="{ componentField }" :name="countryCodeName">
    <FormItem>
      <FormLabel class="flex items-center">
        {{ label || t('globals.terms.phoneNumber') }}
        <span v-if="required" class="text-destructive">*</span>
      </FormLabel>
      <div class="flex items-start">
        <div class="shrink-0">
          <FormControl>
            <ComboBox
              v-bind="componentField"
              :items="allCountries"
              :placeholder="t('globals.terms.select')"
              :buttonClass="'w-auto rounded-r-none border-r-0'"
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
                  <span
                    v-if="selected && selected.calling_code"
                    class="text-xs text-muted-foreground"
                    >({{ selected.calling_code }})</span
                  >
                </div>
              </template>
            </ComboBox>
          </FormControl>
        </div>

        <div class="flex-1 min-w-0">
          <FormField v-slot="{ componentField: phoneField }" :name="phoneNumberName">
            <FormItem>
              <FormControl>
                <Input
                  type="tel"
                  v-bind="phoneField"
                  :placeholder="placeholder"
                  class="rounded-l-none"
                  inputmode="tel"
                />
              </FormControl>
              <FormMessage class="mt-1 text-sm" />
            </FormItem>
          </FormField>
        </div>
      </div>
      <FormMessage />
    </FormItem>
  </FormField>
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
