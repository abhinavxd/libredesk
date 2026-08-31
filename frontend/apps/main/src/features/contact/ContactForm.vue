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
        <PhoneNumberInput />
      </div>
    </div>

    <div class="flex flex-wrap gap-6">
      <div class="flex-1">
        <FormField v-slot="{ componentField }" name="country">
          <FormItem class="flex flex-col">
            <FormLabel class="flex items-center">{{ t('globals.terms.country') }}</FormLabel>
            <FormControl>
              <ComboBox
                v-bind="componentField"
                :items="countryOptions"
                :placeholder="t('globals.terms.select')"
              >
                <template #item="{ item }">
                  <div class="flex items-center gap-2">
                    <span v-if="item.emoji">{{ item.emoji }}</span>
                    <span class="text-sm">{{ item.label }}</span>
                  </div>
                </template>

                <template #selected="{ selected }">
                  <div class="flex items-center gap-1">
                    <span v-if="selected" class="text-lg">{{ selected.emoji }}</span>
                    <span v-if="selected" class="text-sm">{{ selected.label }}</span>
                  </div>
                </template>
              </ComboBox>
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
      <div class="flex-1"></div>
    </div>

    <div v-if="userStore.can('contacts:write')">
      <Button type="submit" :isLoading="formLoading" :disabled="formLoading">
        {{ t('contact.updateContact') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import {
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'
import PhoneNumberInput from '@shared-ui/components/PhoneNumberInput.vue'
import { countryOptions } from '@shared-ui/constants/countries.js'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '../../stores/user'

defineProps(['formLoading', 'onSubmit'])

const { t } = useI18n()
const userStore = useUserStore()
</script>
