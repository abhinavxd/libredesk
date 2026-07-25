<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" @input="generateSlug" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="slug">
      <FormItem>
        <FormLabel>{{ t('helpCenter.slug') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ t('helpCenter.slugHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="page_title">
      <FormItem>
        <FormLabel>{{ t('helpCenter.pageTitle') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="header_text">
      <FormItem>
        <FormLabel>{{ t('helpCenter.headerText') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="logo_url">
      <FormItem>
        <FormLabel>{{ t('globals.terms.logoUrl') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="color">
      <FormItem>
        <FormLabel>{{ t('globals.terms.primaryColor') }}</FormLabel>
        <FormControl>
          <Input type="color" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="theme.favicon">
      <FormItem>
        <FormLabel>{{ t('helpCenter.favicon') }}</FormLabel>
        <FormControl>
          <Input type="text" :placeholder="t('helpCenter.faviconHint')" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Header & hero -->
    <div class="border-t pt-6 space-y-4">
      <h3 class="text-sm font-semibold">{{ t('helpCenter.styling.header') }}</h3>

      <FormField v-slot="{ componentField }" name="theme.tagline">
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.tagline') }}</FormLabel>
          <FormControl>
            <Input type="text" :placeholder="t('helpCenter.styling.taglineHint')" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="theme.header.background_type">
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.headerBackground') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="t('helpCenter.styling.bgDefault')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="default">{{ t('helpCenter.styling.bgDefault') }}</SelectItem>
                <SelectItem value="solid">{{ t('helpCenter.styling.bgSolid') }}</SelectItem>
                <SelectItem value="gradient">{{ t('helpCenter.styling.bgGradient') }}</SelectItem>
              </SelectContent>
            </Select>
          </FormControl>
        </FormItem>
      </FormField>

      <FormField
        v-if="form.values.theme?.header?.background_type === 'solid'"
        v-slot="{ componentField }"
        name="theme.header.background_color"
      >
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.backgroundColor') }}</FormLabel>
          <FormControl>
            <Input type="color" v-bind="componentField" />
          </FormControl>
        </FormItem>
      </FormField>

      <div v-if="form.values.theme?.header?.background_type === 'gradient'" class="flex gap-4">
        <FormField v-slot="{ componentField }" name="theme.header.gradient_from">
          <FormItem class="flex-1">
            <FormLabel>{{ t('helpCenter.styling.gradientFrom') }}</FormLabel>
            <FormControl>
              <Input type="color" v-bind="componentField" />
            </FormControl>
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="theme.header.gradient_to">
          <FormItem class="flex-1">
            <FormLabel>{{ t('helpCenter.styling.gradientTo') }}</FormLabel>
            <FormControl>
              <Input type="color" v-bind="componentField" />
            </FormControl>
          </FormItem>
        </FormField>
      </div>

      <FormField v-slot="{ componentField }" name="theme.header.text_color">
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.textColor') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="#ffffff" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ t('helpCenter.styling.textColorHint') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <!-- Footer -->
    <div class="border-t pt-6 space-y-4">
      <h3 class="text-sm font-semibold">{{ t('helpCenter.styling.footer') }}</h3>

      <div class="flex gap-4">
        <FormField v-slot="{ componentField }" name="theme.footer.background_color">
          <FormItem class="flex-1">
            <FormLabel>{{ t('helpCenter.styling.backgroundColor') }}</FormLabel>
            <FormControl>
              <Input type="text" placeholder="#ffffff" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="theme.footer.text_color">
          <FormItem class="flex-1">
            <FormLabel>{{ t('helpCenter.styling.textColor') }}</FormLabel>
            <FormControl>
              <Input type="text" placeholder="#909aa5" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <FormField v-slot="{ componentField }" name="theme.footer.tagline">
        <FormItem>
          <FormLabel>{{ t('helpCenter.styling.footerTagline') }}</FormLabel>
          <FormControl>
            <Input type="text" :placeholder="t('helpCenter.styling.footerTaglineHint')" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <div class="space-y-2">
        <Label class="block">{{ t('helpCenter.styling.footerLinks') }}</Label>
        <div v-for="(field, index) in footerLinkFields" :key="field.key" class="flex items-start gap-2">
          <FormField v-slot="{ componentField }" :name="`theme.footer_links[${index}].label`">
            <FormItem class="flex-1">
              <FormControl>
                <Input type="text" :placeholder="t('globals.terms.label')" v-bind="componentField" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
          <FormField v-slot="{ componentField }" :name="`theme.footer_links[${index}].url`">
            <FormItem class="flex-1">
              <FormControl>
                <Input type="text" :placeholder="t('globals.terms.url')" v-bind="componentField" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
          <Button type="button" variant="ghost" size="icon" @click="removeFooterLink(index)">
            <X class="w-4 h-4" />
          </Button>
        </div>
        <Button type="button" variant="outline" size="sm" @click="pushFooterLink({ label: '', url: '' })">
          {{ t('globals.messages.add') }}
        </Button>
      </div>

      <div class="space-y-2">
        <Label class="block">{{ t('helpCenter.styling.socialLinks') }}</Label>
        <div v-for="(field, index) in socialLinkFields" :key="field.key" class="flex items-start gap-2">
          <FormField v-slot="{ componentField }" :name="`theme.social_links[${index}].platform`">
            <FormItem class="w-40">
              <FormControl>
                <Select v-bind="componentField">
                  <SelectTrigger>
                    <SelectValue :placeholder="t('helpCenter.styling.platform')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="p in socialPlatforms" :key="p" :value="p">{{ t(`helpCenter.social.${p}`) }}</SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
          <FormField v-slot="{ componentField }" :name="`theme.social_links[${index}].url`">
            <FormItem class="flex-1">
              <FormControl>
                <Input type="text" :placeholder="t('globals.terms.url')" v-bind="componentField" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
          <Button type="button" variant="ghost" size="icon" @click="removeSocialLink(index)">
            <X class="w-4 h-4" />
          </Button>
        </div>
        <Button type="button" variant="outline" size="sm" @click="pushSocialLink({ platform: 'website', url: '' })">
          {{ t('globals.messages.add') }}
        </Button>
      </div>
    </div>

    <!-- Article page -->
    <div class="border-t pt-6 space-y-3">
      <h3 class="text-sm font-semibold">{{ t('helpCenter.styling.articlePage') }}</h3>
      <FormField v-slot="{ value, handleChange }" name="theme.article.hide_toc">
        <FormItem class="flex items-center gap-2 space-y-0">
          <FormControl>
            <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
          </FormControl>
          <FormLabel class="font-normal cursor-pointer">{{ t('helpCenter.styling.showToc') }}</FormLabel>
        </FormItem>
      </FormField>
      <FormField v-slot="{ value, handleChange }" name="theme.article.hide_related">
        <FormItem class="flex items-center gap-2 space-y-0">
          <FormControl>
            <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
          </FormControl>
          <FormLabel class="font-normal cursor-pointer">{{ t('helpCenter.styling.showRelated') }}</FormLabel>
        </FormItem>
      </FormField>
    </div>

    <div class="border-t pt-6 space-y-2">
      <Label class="block">{{ t('helpCenter.navLinks') }}</Label>
      <div v-for="(field, index) in navLinkFields" :key="field.key" class="flex items-start gap-2">
        <FormField v-slot="{ componentField }" :name="`nav_links[${index}].label`">
          <FormItem class="flex-1">
            <FormControl>
              <Input type="text" :placeholder="t('globals.terms.label')" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" :name="`nav_links[${index}].url`">
          <FormItem class="flex-1">
            <FormControl>
              <Input type="text" :placeholder="t('globals.terms.url')" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <Button type="button" variant="ghost" size="icon" @click="removeNavLink(index)">
          <X class="w-4 h-4" />
        </Button>
      </div>
      <Button type="button" variant="outline" size="sm" @click="pushNavLink({ label: '', url: '' })">
        {{ t('globals.messages.add') }}
      </Button>
    </div>

    <div class="space-y-2">
      <Label>{{ t('helpCenter.supportedLanguages') }}</Label>
      <p class="text-sm text-muted-foreground">{{ t('helpCenter.supportedLanguagesHint') }}</p>
      <div v-for="(field, index) in localeFields" :key="field.key" class="flex items-start gap-2">
        <FormField v-slot="{ componentField }" :name="`allowed_locales[${index}]`">
          <FormItem class="flex-1">
            <FormControl>
              <Input type="text" placeholder="en" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          :disabled="localeFields.length <= 1"
          @click="removeLocale(index)"
        >
          <X class="w-4 h-4" />
        </Button>
      </div>
      <Button type="button" variant="outline" size="sm" @click="pushLocale('')">
        {{ t('globals.messages.add') }}
      </Button>
    </div>

    <FormField v-slot="{ componentField }" name="default_locale">
      <FormItem>
        <FormLabel>{{ t('helpCenter.defaultLanguage') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="loc in localeOptions" :key="loc" :value="loc">{{ loc }}</SelectItem>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>{{ t('helpCenter.defaultLanguageHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="custom_css">
      <FormItem>
        <FormLabel>{{ t('helpCenter.customCSS') }}</FormLabel>
        <FormControl>
          <Textarea rows="4" class="font-mono" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="custom_js">
      <FormItem>
        <FormLabel>{{ t('helpCenter.customJS') }}</FormLabel>
        <FormControl>
          <Textarea rows="4" class="font-mono" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="flex justify-end space-x-2 pt-4">
      <Button type="button" variant="outline" @click="$emit('cancel')">
        {{ t('globals.messages.cancel') }}
      </Button>
      <Button type="submit" :isLoading="isLoading">
        {{ submitLabel }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { watch, computed } from 'vue'
import { useForm, useFieldArray } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import { X } from 'lucide-vue-next'
import { createHelpCenterFormSchema } from './helpCenterFormSchema.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  helpCenter: {
    type: Object,
    default: null
  },
  submitForm: {
    type: Function,
    required: true
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

defineEmits(['cancel'])

const { t } = useI18n()

const submitLabel = computed(() =>
  props.helpCenter ? t('globals.messages.update') : t('globals.messages.create')
)

const socialPlatforms = ['website', 'twitter', 'github', 'linkedin', 'facebook', 'instagram', 'youtube']

const toFormValues = (hc) => ({
  name: hc?.name || '',
  slug: hc?.slug || '',
  page_title: hc?.page_title || '',
  header_text: hc?.header_text || '',
  logo_url: hc?.logo_url || '',
  color: hc?.color || '#1f93ff',
  nav_links: Array.isArray(hc?.nav_links) ? hc.nav_links : [],
  custom_css: hc?.custom_css || '',
  custom_js: hc?.custom_js || '',
  default_locale: hc?.default_locale || 'en',
  allowed_locales: Array.isArray(hc?.allowed_locales) && hc.allowed_locales.length ? hc.allowed_locales : ['en'],
  theme: {
    favicon: hc?.theme?.favicon || '',
    tagline: hc?.theme?.tagline || '',
    header: {
      background_type: hc?.theme?.header?.background_type || 'default',
      background_color: hc?.theme?.header?.background_color || '#1f93ff',
      gradient_from: hc?.theme?.header?.gradient_from || '#1f93ff',
      gradient_to: hc?.theme?.header?.gradient_to || '#ffffff',
      text_color: hc?.theme?.header?.text_color || ''
    },
    footer: {
      background_color: hc?.theme?.footer?.background_color || '',
      text_color: hc?.theme?.footer?.text_color || '',
      tagline: hc?.theme?.footer?.tagline || ''
    },
    footer_links: Array.isArray(hc?.theme?.footer_links) ? hc.theme.footer_links : [],
    social_links: Array.isArray(hc?.theme?.social_links) ? hc.theme.social_links : [],
    article: {
      hide_toc: !!hc?.theme?.article?.hide_toc,
      hide_related: !!hc?.theme?.article?.hide_related
    }
  }
})

const form = useForm({
  validationSchema: toTypedSchema(createHelpCenterFormSchema(t)),
  initialValues: toFormValues(props.helpCenter)
})

const {
  fields: navLinkFields,
  push: pushNavLink,
  remove: removeNavLink
} = useFieldArray('nav_links')

const {
  fields: localeFields,
  push: pushLocale,
  remove: removeLocale
} = useFieldArray('allowed_locales')

const {
  fields: footerLinkFields,
  push: pushFooterLink,
  remove: removeFooterLink
} = useFieldArray('theme.footer_links')

const {
  fields: socialLinkFields,
  push: pushSocialLink,
  remove: removeSocialLink
} = useFieldArray('theme.social_links')

const localeOptions = computed(() =>
  (form.values.allowed_locales || []).map((l) => (l || '').trim()).filter(Boolean)
)

const generateSlug = () => {
  if (!props.helpCenter && form.values.name) {
    form.setFieldValue(
      'slug',
      form.values.name
        .toLowerCase()
        .replace(/[^a-z0-9]/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '')
    )
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  const allowed = (values.allowed_locales || []).map((l) => (l || '').trim()).filter(Boolean)
  props.submitForm({
    ...values,
    nav_links: values.nav_links || [],
    allowed_locales: allowed.length ? allowed : ['en']
  })
})

watch(
  () => props.helpCenter,
  (newValues) => {
    if (newValues && Object.keys(newValues).length > 0) {
      form.setValues(toFormValues(newValues))
    }
  },
  { immediate: true }
)
</script>
