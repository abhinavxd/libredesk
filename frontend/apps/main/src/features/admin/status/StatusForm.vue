<template>
  <form>
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Spam" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="category">
      <FormItem class="mt-4">
        <FormLabel>{{ $t('globals.terms.category') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger class="w-full">
              <SelectValue :placeholder="$t('admin.conversationStatus.category.placeholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="open">{{ $t('globals.terms.open') }}</SelectItem>
                <SelectItem value="waiting">{{ $t('globals.terms.waiting') }}</SelectItem>
                <SelectItem value="resolved">{{ $t('globals.terms.resolved') }}</SelectItem>
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
