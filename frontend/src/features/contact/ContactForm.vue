<template>
  <form @submit.prevent="onSubmit" class="space-y-8">
    <div class="flex flex-wrap gap-6">
      <div class="flex-1">
        <FormField v-slot="{ componentField }" name="first_name">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.firstName') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="text" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <div class="flex-1">
        <FormField v-slot="{ componentField }" name="last_name">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.lastName') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="text" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <FormField v-slot="{ componentField }" name="avatar_url">
      <FormItem
        ><FormControl><Input v-bind="componentField" type="hidden" /></FormControl
      ></FormItem>
    </FormField>

    <div class="flex flex-wrap gap-6">
      <div class="flex-1">
        <FormField v-slot="{ componentField }" name="email">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.email') }}</FormLabel>
            <FormControl><Input v-bind="componentField" type="email" /></FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <div class="flex flex-col flex-1">
        <div class="flex items-end">
          <FormField v-slot="{ componentField }" name="phone_number_calling_code">
            <FormItem class="w-20">
              <FormLabel class="flex items-center whitespace-nowrap">
                {{ t('globals.terms.phoneNumber') }}
              </FormLabel>
              <FormControl>
                <ComboBox
                  v-bind="componentField"
                  :items="allCountries"
                  :placeholder="t('globals.messages.select', { name: '' })"
                  :buttonClass="'rounded-r-none border-r-0'"
                >
                  <template #item="{ item }">
                    <div class="flex items-center gap-2">
                      <div class="w-7 h-7 flex items-center justify-center">
                        <span v-if="item.emoji">{{ item.emoji }}</span>
                      </div>
                      <span class="text-sm">{{ item.label }} ({{ item.value }})</span>
                    </div>
                  </template>

                  <template #selected="{ selected }">
                    <div class="flex items-center mb-1">
                      <span v-if="selected" class="text-xl leading-none">{{ selected.emoji }}</span>
                    </div>
                  </template>
                </ComboBox>
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <div class="flex-1">
            <FormField v-slot="{ componentField }" name="phone_number">
              <FormItem class="relative">
                <FormControl>
                  <Input
                    type="tel"
                    v-bind="componentField"
                    class="rounded-l-none"
                    inputmode="numeric"
                  />
                  <FormMessage class="absolute top-full mt-1 text-sm" />
                </FormControl>
              </FormItem>
            </FormField>
          </div>
        </div>
      </div>
    </div>

    <div v-if="userStore.can('contacts:write')">
      <Button type="submit" :isLoading="formLoading" :disabled="formLoading">
        {{ t('globals.messages.update', { name: t('globals.terms.contact').toLowerCase() }) }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'
import countries from '@/constants/countries.js'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'

defineProps(['formLoading', 'onSubmit'])

const { t } = useI18n()
const userStore = useUserStore()

const allCountries = countries.map((country) => ({
  label: country.name,
  value: country.calling_code,
  emoji: country.emoji
}))
</script>
