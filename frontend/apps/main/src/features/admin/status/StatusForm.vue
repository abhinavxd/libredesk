<template>
  <form>
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Spam" v-bind="componentField" />
        </FormControl>
        <FormDescription/>
        <FormMessage />
      </FormItem>
    </FormField>
    <!-- Form submit button slot -->
    <slot name="footer"></slot>
  </form>
</template>

<script setup>
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
</script>
