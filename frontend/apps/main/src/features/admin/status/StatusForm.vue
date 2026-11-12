<template>
  <form>
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Spam" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="category">
      <FormItem class="mt-4">
        <FormLabel>{{ i18n._('Category') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger class="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="open">{{ i18n._('Open') }}</SelectItem>
                <SelectItem value="waiting">{{ i18n._('Waiting') }}</SelectItem>
                <SelectItem value="resolved">{{ i18n._('Resolved') }}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <slot name="footer"></slot>
  </form>
</template>

<script setup>
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
</script>
