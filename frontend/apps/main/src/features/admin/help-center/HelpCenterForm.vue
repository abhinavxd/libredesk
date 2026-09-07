<template>
  <form ref="formEl" @submit="onSubmit" novalidate class="space-y-6 w-full">
    <Tabs v-model="activeTab" class="w-full">
      <TabsList class="grid w-full grid-cols-2">
        <TabsTrigger value="general">{{ t('globals.terms.general') }}</TabsTrigger>
        <TabsTrigger value="appearance">{{ t('globals.terms.appearance') }}</TabsTrigger>
      </TabsList>

      <div class="mt-8">
        <div v-show="activeTab === 'general'" class="space-y-6">
          <FormField v-slot="{ componentField }" name="name">
            <FormItem>
              <FormLabel>{{ t('globals.terms.name') }}</FormLabel>
              <FormControl>
                <Input type="text" v-bind="componentField" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="slug">
            <FormItem>
              <FormLabel>{{ t('globals.terms.slug') }}</FormLabel>
              <FormControl>
                <Input type="text" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{ t('helpCenter.slugHint') }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="custom_domain">
            <FormItem>
              <FormLabel>{{ t('helpCenter.customDomain') }}</FormLabel>
              <FormControl>
                <Input type="text" placeholder="https://help.example.com" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{ t('helpCenter.customDomainHint') }}</FormDescription>
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

          <FormField v-slot="{ componentField }" name="template">
            <FormItem>
              <FormLabel>{{ t('globals.terms.template') }}</FormLabel>
              <FormControl>
                <Select v-bind="componentField">
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="classic">{{ t('helpCenter.templates.classic') }}</SelectItem>
                    <SelectItem value="docs">{{ t('helpCenter.templates.docs') }}</SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormDescription>{{
                form.values.template === 'classic'
                  ? t('helpCenter.templates.classicHint')
                  : t('helpCenter.templates.docsHint')
              }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="meta_description">
            <FormItem>
              <FormLabel>{{ t('helpCenter.metaDescription') }}</FormLabel>
              <FormControl>
                <Textarea :rows="2" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{ t('helpCenter.homeMetaDescriptionHint') }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>

          <LinkListField name="theme.nav_links" :label="t('helpCenter.navLinks')" />

          <div class="space-y-2">
            <Label>{{ t('helpCenter.supportedLanguages') }}</Label>
            <p class="text-sm text-muted-foreground">
              {{ t('helpCenter.supportedLanguagesHint') }}
            </p>
            <div
              v-for="(field, index) in localeFields"
              :key="field.key"
              class="flex items-start gap-2"
            >
              <FormField v-slot="{ componentField }" :name="`allowed_locales[${index}]`">
                <FormItem class="flex-1">
                  <FormControl>
                    <SelectComboBox
                      v-bind="componentField"
                      :items="localeItemsFor(index)"
                      :placeholder="t('placeholders.selectLanguage')"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                :aria-label="`${t('globals.terms.remove')} ${field.value || ''}`"
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
                    <SelectItem v-for="loc in localeOptions" :key="loc" :value="loc">{{
                      localeLabel(loc)
                    }}</SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormDescription>{{ t('helpCenter.defaultLanguageHint') }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>

        <div v-show="activeTab === 'appearance'">
          <CollapsibleSection
            :title="t('helpCenter.styling.brand')"
            :open="openSection === 'brand'"
            @toggle="toggleSection('brand')"
          >
            <FormField v-slot="{ componentField }" name="theme.logo_url">
              <FormItem>
                <FormLabel>{{ t('globals.terms.logoUrl') }}</FormLabel>
                <FormControl>
                  <Input type="text" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="theme.color">
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
                <FormLabel>{{ t('admin.general.faviconURL') }}</FormLabel>
                <FormControl>
                  <Input
                    type="text"
                    :placeholder="t('helpCenter.faviconHint')"
                    v-bind="componentField"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
          </CollapsibleSection>

          <CollapsibleSection
            :title="t('helpCenter.styling.header')"
            :open="openSection === 'header'"
            @toggle="toggleSection('header')"
          >
            <FormField v-slot="{ componentField }" name="theme.header.heading">
              <FormItem>
                <FormLabel>{{ t('helpCenter.headerText') }}</FormLabel>
                <FormControl>
                  <Input type="text" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="theme.tagline">
              <FormItem>
                <FormLabel>{{ t('helpCenter.styling.tagline') }}</FormLabel>
                <FormControl>
                  <Textarea :rows="2" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ t('helpCenter.styling.inlineMarkdownHint') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <div v-show="isClassic">
              <FormField v-slot="{ componentField }" name="theme.header.background_type">
                <FormItem>
                  <FormLabel>{{ t('globals.terms.background') }}</FormLabel>
                  <FormControl>
                    <Select v-bind="componentField">
                      <SelectTrigger>
                        <SelectValue :placeholder="t('helpCenter.styling.bgDefault')" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="default">{{
                          t('helpCenter.styling.bgDefault')
                        }}</SelectItem>
                        <SelectItem value="solid">{{ t('helpCenter.styling.bgSolid') }}</SelectItem>
                        <SelectItem value="gradient">{{ t('globals.terms.gradient') }}</SelectItem>
                        <SelectItem value="image">{{ t('globals.terms.image') }}</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                </FormItem>
              </FormField>
            </div>

            <div v-show="isClassic && form.values.theme?.header?.background_type === 'solid'">
              <FormField v-slot="{ componentField }" name="theme.header.background_color">
                <FormItem>
                  <FormLabel>{{ t('globals.messages.backgroundColor') }}</FormLabel>
                  <FormControl>
                    <Input type="color" v-bind="componentField" />
                  </FormControl>
                </FormItem>
              </FormField>
            </div>

            <div
              v-show="isClassic && form.values.theme?.header?.background_type === 'gradient'"
              class="flex gap-4"
            >
              <FormField v-slot="{ componentField }" name="theme.header.gradient_from">
                <FormItem class="flex-1">
                  <FormLabel>{{ t('globals.messages.gradientStart') }}</FormLabel>
                  <FormControl>
                    <Input type="color" v-bind="componentField" />
                  </FormControl>
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField }" name="theme.header.gradient_to">
                <FormItem class="flex-1">
                  <FormLabel>{{ t('globals.messages.gradientEnd') }}</FormLabel>
                  <FormControl>
                    <Input type="color" v-bind="componentField" />
                  </FormControl>
                </FormItem>
              </FormField>
            </div>

            <div v-show="isClassic && form.values.theme?.header?.background_type === 'image'">
              <FormField v-slot="{ componentField }" name="theme.header.background_image">
                <FormItem>
                  <FormLabel>{{ t('globals.messages.backgroundImageUrl') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      placeholder="https://example.com/header.jpg"
                      v-bind="componentField"
                    />
                  </FormControl>
                  <FormDescription>{{ t('helpCenter.styling.headerImageHint') }}</FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <div v-show="isClassic">
              <FormField v-slot="{ componentField }" name="theme.header.text_color">
                <FormItem>
                  <FormLabel>{{ t('globals.terms.textColor') }}</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="#ffffff" v-bind="componentField" />
                  </FormControl>
                  <FormDescription>{{ t('helpCenter.styling.textColorHint') }}</FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
          </CollapsibleSection>

          <CollapsibleSection
            :title="t('globals.terms.announcement')"
            :open="openSection === 'announcement'"
            @toggle="toggleSection('announcement')"
          >
            <FormField v-slot="{ componentField }" name="theme.announcement.text">
              <FormItem>
                <FormLabel>{{ t('globals.terms.message') }}</FormLabel>
                <FormControl>
                  <Textarea :rows="2" v-bind="componentField" />
                </FormControl>
                <FormDescription
                  >{{ t('helpCenter.styling.announcementHint') }}
                  {{ t('helpCenter.styling.inlineMarkdownHint') }}</FormDescription
                >
                <FormMessage />
              </FormItem>
            </FormField>

            <div class="flex gap-4">
              <FormField v-slot="{ componentField }" name="theme.announcement.link_label">
                <FormItem class="flex-1">
                  <FormLabel>{{ t('globals.terms.label') }}</FormLabel>
                  <FormControl>
                    <Input type="text" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField }" name="theme.announcement.link_url">
                <FormItem class="flex-1">
                  <FormLabel>{{ t('globals.terms.url') }}</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="https://" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
          </CollapsibleSection>

          <CollapsibleSection
            :title="t('helpCenter.styling.landingPage')"
            :open="openSection === 'landing'"
            @toggle="toggleSection('landing')"
          >
            <FormField v-slot="{ componentField }" name="theme.layout.collections">
              <FormItem>
                <FormLabel>{{ t('helpCenter.styling.collectionLayout') }}</FormLabel>
                <FormControl>
                  <Select v-bind="componentField">
                    <SelectTrigger>
                      <SelectValue :placeholder="t('helpCenter.styling.layoutGrid')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="grid">{{ t('helpCenter.styling.layoutGrid') }}</SelectItem>
                      <SelectItem value="list">{{ t('helpCenter.styling.layoutList') }}</SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
              </FormItem>
            </FormField>

            <div v-show="form.values.theme?.layout?.collections !== 'list'">
              <FormField v-slot="{ componentField }" name="theme.layout.columns">
                <FormItem>
                  <FormLabel>{{ t('helpCenter.styling.cardsPerRow') }}</FormLabel>
                  <FormControl>
                    <Select v-bind="componentField">
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="2">2</SelectItem>
                        <SelectItem value="3">3</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                </FormItem>
              </FormField>
            </div>

            <div v-show="form.values.theme?.layout?.collections !== 'list'">
              <FormField v-slot="{ componentField }" name="theme.cards.icon_position">
                <FormItem>
                  <FormLabel>{{ t('helpCenter.styling.cardIconPosition') }}</FormLabel>
                  <FormControl>
                    <Select v-bind="componentField">
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="inline">{{
                          t('helpCenter.styling.iconBesideTitle')
                        }}</SelectItem>
                        <SelectItem value="top">{{
                          t('helpCenter.styling.iconAboveTitle')
                        }}</SelectItem>
                        <SelectItem value="center">{{
                          t('helpCenter.styling.iconCentered')
                        }}</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                </FormItem>
              </FormField>
            </div>

            <FormField v-slot="{ value, handleChange }" name="theme.layout.show_popular_articles">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="value" @update:checked="handleChange" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  t('helpCenter.styling.showPopularArticles')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <div v-show="form.values.theme?.layout?.show_popular_articles">
              <FormField v-slot="{ componentField }" name="theme.layout.popular_articles_label">
                <FormItem>
                  <FormLabel>{{ t('helpCenter.styling.popularArticlesLabel') }}</FormLabel>
                  <FormControl>
                    <Input type="text" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
            <FormField v-slot="{ value, handleChange }" name="theme.cards.hide_description">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  t('helpCenter.styling.showCardDescription')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <FormField v-slot="{ value, handleChange }" name="theme.cards.hide_count">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  t('helpCenter.styling.showCardCount')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <FormField v-slot="{ value, handleChange }" name="theme.cards.show_icon_tile">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="value" @update:checked="handleChange" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  t('helpCenter.styling.showIconTile')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <FormField v-slot="{ value, handleChange }" name="theme.cards.show_authors">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="value" @update:checked="handleChange" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  t('helpCenter.styling.showCardAuthors')
                }}</FormLabel>
              </FormItem>
            </FormField>
          </CollapsibleSection>

          <CollapsibleSection
            :title="t('helpCenter.styling.articlePage')"
            :open="openSection === 'article'"
            @toggle="toggleSection('article')"
          >
            <FormField v-slot="{ value, handleChange }" name="theme.article.hide_toc">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  t('helpCenter.styling.showToc')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <FormField v-slot="{ value, handleChange }" name="theme.article.hide_related">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  t('helpCenter.styling.showRelated')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <FormField v-slot="{ value, handleChange }" name="theme.article.show_author">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="value" @update:checked="handleChange" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  t('helpCenter.styling.showAuthor')
                }}</FormLabel>
              </FormItem>
            </FormField>
          </CollapsibleSection>

          <CollapsibleSection
            :title="t('globals.terms.footer')"
            :open="openSection === 'footer'"
            @toggle="toggleSection('footer')"
          >
            <div class="flex gap-4">
              <FormField v-slot="{ componentField }" name="theme.footer.background_color">
                <FormItem class="flex-1">
                  <FormLabel>{{ t('globals.messages.backgroundColor') }}</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="#ffffff" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField }" name="theme.footer.text_color">
                <FormItem class="flex-1">
                  <FormLabel>{{ t('globals.terms.textColor') }}</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="#909aa5" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <FormField v-slot="{ componentField }" name="theme.footer.tagline">
              <FormItem>
                <FormLabel>{{ t('helpCenter.styling.tagline') }}</FormLabel>
                <FormControl>
                  <Textarea
                    :rows="2"
                    :placeholder="t('helpCenter.styling.footerTaglineHint')"
                    v-bind="componentField"
                  />
                </FormControl>
                <FormDescription>{{ t('helpCenter.styling.inlineMarkdownHint') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <LinkListField name="theme.footer_links" :label="t('globals.terms.link', 2)" />

            <LinkListField
              name="theme.social_links"
              :label="t('helpCenter.styling.socialLinks')"
              :new-item="{ platform: 'website', url: '' }"
            >
              <template #leading="{ index }">
                <FormField
                  v-slot="{ componentField }"
                  :name="`theme.social_links[${index}].platform`"
                >
                  <FormItem class="w-40">
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue :placeholder="t('globals.terms.platform')" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem v-for="p in socialPlatforms" :key="p" :value="p">{{
                            t(`helpCenter.social.${p}`)
                          }}</SelectItem>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </template>
            </LinkListField>
          </CollapsibleSection>

          <CollapsibleSection
            :title="t('helpCenter.styling.customCode')"
            :open="openSection === 'code'"
            @toggle="toggleSection('code')"
          >
            <FormField v-slot="{ componentField }" name="custom_css">
              <FormItem>
                <FormLabel>{{ t('helpCenter.customCSS') }}</FormLabel>
                <FormControl>
                  <Textarea
                    rows="4"
                    class="font-mono"
                    placeholder=".hc-brand img { max-height: 2.4rem; }"
                    v-bind="componentField"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="custom_js">
              <FormItem>
                <FormLabel>{{ t('helpCenter.customJS') }}</FormLabel>
                <FormControl>
                  <Textarea
                    rows="4"
                    class="font-mono"
                    placeholder='document.title += " | Support";'
                    v-bind="componentField"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
          </CollapsibleSection>
        </div>
      </div>
    </Tabs>

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
import { watch, computed, ref, onMounted, nextTick } from 'vue'
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
import { Tabs, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import CollapsibleSection from './CollapsibleSection.vue'
import LinkListField from './LinkListField.vue'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import { createHelpCenterFormSchema } from './helpCenterFormSchema.js'
import api from '@/api'
import { useI18n } from 'vue-i18n'

// Field path prefix -> [tab, section] to reveal when that field errors on submit.
const FIELD_LOCATION = [
  ['theme.header', 'appearance', 'header'],
  ['theme.tagline', 'appearance', 'header'],
  ['theme.announcement', 'appearance', 'announcement'],
  ['theme.logo_url', 'appearance', 'brand'],
  ['theme.color', 'appearance', 'brand'],
  ['theme.favicon', 'appearance', 'brand'],
  ['theme.nav_links', 'general', ''],
  ['theme.layout', 'appearance', 'landing'],
  ['theme.cards', 'appearance', 'landing'],
  ['theme.article', 'appearance', 'article'],
  ['theme.footer', 'appearance', 'footer'],
  ['theme.social_links', 'appearance', 'footer'],
  ['custom_css', 'appearance', 'code'],
  ['custom_js', 'appearance', 'code']
]

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

const emit = defineEmits(['cancel', 'change'])

const { t } = useI18n()

const submitLabel = computed(() =>
  props.helpCenter ? t('globals.messages.update') : t('globals.messages.create')
)

const socialPlatforms = [
  'website',
  'twitter',
  'github',
  'linkedin',
  'facebook',
  'instagram',
  'youtube'
]

const toFormValues = (hc) => ({
  name: hc?.name || '',
  slug: hc?.slug || '',
  template: hc?.template === 'docs' ? 'docs' : 'classic',
  custom_domain: hc?.custom_domain || '',
  page_title: hc?.page_title || '',
  meta_description: hc?.meta_description || '',
  custom_css: hc?.custom_css || '',
  custom_js: hc?.custom_js || '',
  default_locale: hc?.default_locale || 'en',
  allowed_locales:
    Array.isArray(hc?.allowed_locales) && hc.allowed_locales.length ? hc.allowed_locales : ['en'],
  theme: {
    color: hc?.theme?.color || '#1f93ff',
    logo_url: hc?.theme?.logo_url || '',
    nav_links: Array.isArray(hc?.theme?.nav_links) ? hc.theme.nav_links : [],
    favicon: hc?.theme?.favicon || '',
    tagline: hc?.theme?.tagline || '',
    header: {
      heading: hc?.theme?.header?.heading || '',
      background_type: hc?.theme?.header?.background_type || 'default',
      background_color: hc?.theme?.header?.background_color || '#1f93ff',
      gradient_from: hc?.theme?.header?.gradient_from || '#1f93ff',
      gradient_to: hc?.theme?.header?.gradient_to || '#ffffff',
      background_image: hc?.theme?.header?.background_image || '',
      text_color: hc?.theme?.header?.text_color || ''
    },
    layout: {
      collections: hc?.theme?.layout?.collections || 'grid',
      columns: String(hc?.theme?.layout?.columns || 2),
      show_popular_articles: hc?.theme?.layout?.show_popular_articles ?? true,
      popular_articles_label: hc
        ? hc.theme?.layout?.popular_articles_label || ''
        : t('helpCenter.popularArticles')
    },
    cards: {
      hide_description: !!hc?.theme?.cards?.hide_description,
      hide_count: !!hc?.theme?.cards?.hide_count,
      show_authors: !!hc?.theme?.cards?.show_authors,
      show_icon_tile: hc?.theme?.cards?.show_icon_tile ?? true,
      icon_position: hc?.theme?.cards?.icon_position || 'inline'
    },
    announcement: {
      text: hc?.theme?.announcement?.text || '',
      link_label: hc?.theme?.announcement?.link_label || '',
      link_url: hc?.theme?.announcement?.link_url || ''
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
      hide_related: !!hc?.theme?.article?.hide_related,
      show_author: hc?.theme?.article?.show_author ?? true
    }
  }
})

const formEl = ref(null)
const activeTab = ref('general')
const openSection = ref('brand')

const toggleSection = (key) => {
  openSection.value = openSection.value === key ? null : key
}

const form = useForm({
  validationSchema: toTypedSchema(createHelpCenterFormSchema(t)),
  initialValues: toFormValues(props.helpCenter)
})

const {
  fields: localeFields,
  push: pushLocale,
  remove: removeLocale
} = useFieldArray('allowed_locales')

const isClassic = computed(() => form.values.template === 'classic')

const supportedLocales = ref([])

onMounted(async () => {
  try {
    const { data } = await api.getHelpCenterLocales()
    supportedLocales.value = data.data || []
  } catch {
    supportedLocales.value = []
  }
})

const localeItems = computed(() =>
  supportedLocales.value.map((l) => ({ label: `${l.name} (${l.code})`, value: l.code }))
)

const localeLabel = (code) => localeItems.value.find((i) => i.value === code)?.label ?? code

const cleanLocales = (locales) => (locales || []).map((l) => (l || '').trim()).filter(Boolean)

const pickedLocales = computed(() => new Set(cleanLocales(form.values.allowed_locales)))

const localeItemsFor = (index) => {
  const own = (form.values.allowed_locales?.[index] || '').trim()
  return localeItems.value.filter(
    (item) => item.value === own || !pickedLocales.value.has(item.value)
  )
}

const localeOptions = computed(() => [...pickedLocales.value])

// The backend forces the default language back into the supported list, so the select is
// moved to a language that is actually still listed instead of showing a stale one.
watch(localeOptions, (locales) => {
  if (locales.length && !locales.includes(form.values.default_locale)) {
    form.setFieldValue('default_locale', locales[0], false)
  }
})

const toPayload = (values) => {
  const payload = JSON.parse(JSON.stringify(values))
  const allowed = cleanLocales(payload.allowed_locales)
  payload.allowed_locales = allowed.length ? allowed : ['en']
  if (payload.theme?.layout) {
    payload.theme.layout.columns = Number(payload.theme.layout.columns) || 2
  }
  return payload
}

const onSubmit = form.handleSubmit(
  async (values) => {
    props.submitForm(toPayload(values))
  },
  async ({ errors }) => {
    const firstKey = Object.keys(errors)[0]
    const match =
      firstKey &&
      FIELD_LOCATION.find(([prefix]) => firstKey === prefix || firstKey.startsWith(prefix))
    if (match) {
      activeTab.value = match[1]
      openSection.value = match[2]
    } else if (firstKey) {
      activeTab.value = 'general'
    }
    await nextTick()
    formEl.value?.querySelector('[role="alert"]')?.scrollIntoView({ block: 'nearest' })
  }
)

watch(
  () => props.helpCenter,
  (newValues) => {
    if (newValues && Object.keys(newValues).length > 0) {
      form.setValues(toFormValues(newValues), false)
    }
  },
  { immediate: true }
)

// The select yields a string; the stored theme needs columns as a number.
watch(
  () => form.values,
  (values) => emit('change', toPayload(values)),
  { deep: true, immediate: true }
)
</script>
