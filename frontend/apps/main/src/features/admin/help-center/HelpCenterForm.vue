<template>
  <form ref="formEl" @submit="onSubmit" novalidate class="space-y-6 w-full">
    <Tabs v-model="activeTab" class="w-full">
      <AdminToolbar>
        <template #left>
          <TabsList class="w-max justify-start">
            <TabsTrigger value="general">{{ i18n._('General') }}</TabsTrigger>
            <TabsTrigger value="appearance">{{ i18n._('Appearance') }}</TabsTrigger>
          </TabsList>
        </template>
      </AdminToolbar>

      <div class="mt-8">
        <div v-show="activeTab === 'general'" class="space-y-6">
          <FormGrid>
            <FormField v-slot="{ componentField }" name="name">
              <FormItem>
                <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
                <FormControl>
                  <Input type="text" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="slug">
              <FormItem>
                <FormLabel>{{ i18n.ngettext('Slug', 'Slugs', 1) }}</FormLabel>
                <FormControl>
                  <Input type="text" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ i18n._('Used in the help center URL.') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
          </FormGrid>

          <FormGrid>
            <FormField v-slot="{ componentField }" name="custom_domain">
              <FormItem>
                <FormLabel>{{ i18n._('Custom domain (optional)') }}</FormLabel>
                <FormControl>
                  <Input
                    type="text"
                    placeholder="https://help.example.com"
                    v-bind="componentField"
                  />
                </FormControl>
                <FormDescription>{{ i18n._('Serve this help center on its own domain instead of the app URL. Point the domain at Libredesk in your DNS and reverse proxy first.') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="livechat_inbox_id">
              <FormItem>
                <FormLabel>{{ i18n.ngettext('Live chat inbox', 'Live chat inboxes', 1) }}</FormLabel>
                <FormControl>
                  <Select v-bind="componentField">
                    <SelectTrigger>
                      <SelectValue :placeholder="i18n._('Select inbox')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">{{ i18n.pgettext('No linked inbox', 'None') }}</SelectItem>
                      <SelectItem
                        v-for="option in inboxStore.livechatOptions"
                        :key="option.value"
                        :value="option.value"
                      >
                        {{ option.label }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormDescription>{{ i18n._('Shows the chat widget for this inbox on every help center page.') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
          </FormGrid>

          <FormGrid>
            <FormField v-slot="{ componentField }" name="page_title">
              <FormItem>
                <FormLabel>{{ i18n._('Page title') }}</FormLabel>
                <FormControl>
                  <Input type="text" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="template">
              <FormItem>
                <FormLabel>{{ i18n.ngettext('Template', 'Templates', 1, {}, 'Help center layout') }}</FormLabel>
                <FormControl>
                  <Select v-bind="componentField">
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="classic">{{
                        i18n._('Classic')
                      }}</SelectItem>
                      <SelectItem value="docs">{{ i18n._('Docs') }}</SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormDescription>{{
                  form.values.template === 'classic'
                    ? i18n._('Landing page layout with a hero banner, a large search bar and collection cards.')
                    : i18n._('Documentation layout with a top navigation bar, a sidebar listing collections and quick search (Ctrl+K).')
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
          </FormGrid>

          <FormField v-slot="{ componentField }" name="meta_description">
            <FormItem>
              <FormLabel>{{ i18n._('Meta description') }}</FormLabel>
              <FormControl>
                <Textarea :rows="2" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{ i18n._('Shown in search-engine results and social previews for the help center home page. Leave blank to use the header text.') }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>

          <LinkListField name="theme.nav_links" :label="i18n._('Navigation links')" />

          <div class="space-y-2">
            <Label>{{ i18n._('Supported languages') }}</Label>
            <p class="ui-description">
              {{ i18n._('Add a language before authoring articles in it.') }}
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
                      :placeholder="i18n.pgettext('Language selector', 'Select language')"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                :aria-label="`${i18n._('Remove')} ${field.value || ''}`"
                :disabled="localeFields.length <= 1"
                @click="removeLocale(index)"
              >
                <X class="w-4 h-4" />
              </Button>
            </div>
            <Button type="button" variant="outline" size="sm" @click="pushLocale('')">
              {{ i18n._('Add') }}
            </Button>
          </div>

          <FormField v-slot="{ componentField }" name="default_locale">
            <FormItem>
              <FormLabel>{{ i18n._('Default language') }}</FormLabel>
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
              <FormDescription>{{ i18n._('Visitors land on this language unless they pick another.') }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>

        <div v-show="activeTab === 'appearance'">
          <CollapsibleSection
            :title="i18n._('Brand')"
            :open="openSection === 'brand'"
            @toggle="toggleSection('brand')"
          >
            <FormField v-slot="{ componentField }" name="theme.favicon">
              <FormItem>
                <FormLabel>{{ i18n._('Favicon URL') }}</FormLabel>
                <FormControl>
                  <Input
                    type="text"
                    :placeholder="i18n._('https://example.com/favicon.ico')"
                    v-bind="componentField"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="theme.color_scheme">
              <FormItem>
                <FormLabel>{{ i18n._('Theme') }}</FormLabel>
                <FormControl>
                  <RadioGroup v-bind="componentField" class="flex gap-4">
                    <div class="flex items-center space-x-2">
                      <RadioGroupItem id="hc-scheme-system" value="system" />
                      <Label for="hc-scheme-system">{{ i18n._('Match system') }}</Label>
                    </div>
                    <div class="flex items-center space-x-2">
                      <RadioGroupItem id="hc-scheme-light" value="light" />
                      <Label for="hc-scheme-light">{{ i18n._('Light') }}</Label>
                    </div>
                    <div class="flex items-center space-x-2">
                      <RadioGroupItem id="hc-scheme-dark" value="dark" />
                      <Label for="hc-scheme-dark">{{ i18n._('Dark') }}</Label>
                    </div>
                  </RadioGroup>
                </FormControl>
                <FormDescription>{{ i18n._('Match system follows the visitor\'\'s own color scheme.') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <Tabs :model-value="editingScheme" @update:model-value="editingScheme = $event">
              <AdminToolbar>
                <template #left>
                  <TabsList class="w-max justify-start">
                    <TabsTrigger value="light" class="gap-2 h-8">
                      <Sun class="size-4" />
                      {{ i18n._('Light') }}
                    </TabsTrigger>
                    <TabsTrigger value="dark" class="gap-2 h-8">
                      <Moon class="size-4" />
                      {{ i18n._('Dark') }}
                    </TabsTrigger>
                  </TabsList>
                </template>
              </AdminToolbar>
            </Tabs>

            <FormField
              v-slot="{ componentField }"
              :name="editingScheme === 'dark' ? 'theme.logo_url_dark' : 'theme.logo_url'"
              keep-value
            >
              <FormItem>
                <FormLabel>{{ i18n._('Logo URL') }}</FormLabel>
                <FormControl>
                  <Input type="text" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField
              v-slot="{ componentField }"
              :name="editingScheme === 'dark' ? 'theme.color_dark' : 'theme.color'"
              keep-value
            >
              <FormItem>
                <FormLabel>{{ i18n.ngettext('Primary color', 'Primary colors', 1) }}</FormLabel>
                <FormControl>
                  <Input type="color" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
          </CollapsibleSection>

          <CollapsibleSection
            :title="i18n._('Header \u0026 hero')"
            :open="openSection === 'header'"
            @toggle="toggleSection('header')"
          >
            <FormField v-slot="{ componentField }" name="theme.header.heading">
              <FormItem>
                <FormLabel>{{ i18n._('Header text') }}</FormLabel>
                <FormControl>
                  <Input type="text" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="theme.tagline">
              <FormItem>
                <FormLabel>{{ i18n._('Tagline') }}</FormLabel>
                <FormControl>
                  <Textarea :rows="2" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ i18n._('Markdown is supported, e.g. **bold**, *italic* and [links](https://example.com).') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <div v-show="isClassic">
              <FormField v-slot="{ componentField }" name="theme.header.background_type">
                <FormItem>
                  <FormLabel>{{ i18n._('Background') }}</FormLabel>
                  <FormControl>
                    <Select v-bind="componentField">
                      <SelectTrigger>
                        <SelectValue :placeholder="i18n._('Default (accent tint)')" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="default">{{
                          i18n._('Default (accent tint)')
                        }}</SelectItem>
                        <SelectItem value="solid">{{ i18n._('Solid color') }}</SelectItem>
                        <SelectItem value="gradient">{{ i18n._('Gradient') }}</SelectItem>
                        <SelectItem value="image">{{ i18n.ngettext('Image', 'Images', 1) }}</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                </FormItem>
              </FormField>
            </div>

            <div v-show="isClassic && form.values.theme?.header?.background_type === 'solid'">
              <FormField v-slot="{ componentField }" name="theme.header.background_color">
                <FormItem>
                  <FormLabel>{{ i18n._('Background color') }}</FormLabel>
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
                  <FormLabel>{{ i18n._('Gradient start') }}</FormLabel>
                  <FormControl>
                    <Input type="color" v-bind="componentField" />
                  </FormControl>
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField }" name="theme.header.gradient_to">
                <FormItem class="flex-1">
                  <FormLabel>{{ i18n._('Gradient end') }}</FormLabel>
                  <FormControl>
                    <Input type="color" v-bind="componentField" />
                  </FormControl>
                </FormItem>
              </FormField>
            </div>

            <div v-show="isClassic && form.values.theme?.header?.background_type === 'image'">
              <FormField v-slot="{ componentField }" name="theme.header.background_image">
                <FormItem>
                  <FormLabel>{{ i18n._('Background image URL') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      placeholder="https://example.com/header.jpg"
                      v-bind="componentField"
                    />
                  </FormControl>
                  <FormDescription>{{ i18n._('Recommended: at least 1600x500px (roughly 3:1), JPG/PNG/WebP. The image is centered and cropped to fit.') }}</FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <div v-show="isClassic">
              <FormField v-slot="{ componentField }" name="theme.header.text_color">
                <FormItem>
                  <FormLabel>{{ i18n._('Text color') }}</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="#ffffff" v-bind="componentField" />
                  </FormControl>
                  <FormDescription>{{ i18n._('Leave blank to use the default.') }}</FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
          </CollapsibleSection>

          <CollapsibleSection
            :title="i18n.ngettext('Announcement', 'Announcements', 1)"
            :open="openSection === 'announcement'"
            @toggle="toggleSection('announcement')"
          >
            <FormField v-slot="{ componentField }" name="theme.announcement.text">
              <FormItem>
                <FormLabel>{{ i18n.ngettext('Message', 'Messages', 1) }}</FormLabel>
                <FormControl>
                  <Textarea :rows="2" v-bind="componentField" />
                </FormControl>
                <FormDescription
                  >{{ i18n._('A dismissible bar shown above the header on every page. Leave the text empty to hide it.') }}
                  {{ i18n._('Markdown is supported, e.g. **bold**, *italic* and [links](https://example.com).') }}</FormDescription
                >
                <FormMessage />
              </FormItem>
            </FormField>

            <div class="flex gap-4">
              <FormField v-slot="{ componentField }" name="theme.announcement.link_label">
                <FormItem class="flex-1">
                  <FormLabel>{{ i18n.ngettext('Label', 'Labels', 1) }}</FormLabel>
                  <FormControl>
                    <Input type="text" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField }" name="theme.announcement.link_url">
                <FormItem class="flex-1">
                  <FormLabel>{{ i18n.ngettext('URL', 'URLs', 1) }}</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="https://" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
          </CollapsibleSection>

          <CollapsibleSection
            :title="i18n._('Landing page')"
            :open="openSection === 'landing'"
            @toggle="toggleSection('landing')"
          >
            <FormField v-slot="{ componentField }" name="theme.layout.collections">
              <FormItem>
                <FormLabel>{{ i18n._('Collection layout') }}</FormLabel>
                <FormControl>
                  <Select v-bind="componentField">
                    <SelectTrigger>
                      <SelectValue :placeholder="i18n._('Card grid')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="grid">{{ i18n._('Card grid') }}</SelectItem>
                      <SelectItem value="list">{{ i18n._('One per row') }}</SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
              </FormItem>
            </FormField>

            <div v-show="form.values.theme?.layout?.collections !== 'list'">
              <FormField v-slot="{ componentField }" name="theme.layout.columns">
                <FormItem>
                  <FormLabel>{{ i18n._('Cards per row') }}</FormLabel>
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
                  <FormLabel>{{ i18n._('Card icon position') }}</FormLabel>
                  <FormControl>
                    <Select v-bind="componentField">
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="inline">{{
                          i18n._('Beside the title')
                        }}</SelectItem>
                        <SelectItem value="top">{{
                          i18n._('Above the title')
                        }}</SelectItem>
                        <SelectItem value="center">{{
                          i18n._('Centered')
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
                  i18n._('Show popular articles')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <div v-show="form.values.theme?.layout?.show_popular_articles">
              <FormField v-slot="{ componentField }" name="theme.layout.popular_articles_label">
                <FormItem>
                  <FormLabel>{{ i18n._('Popular articles heading') }}</FormLabel>
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
                  i18n._('Show collection descriptions')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <FormField v-slot="{ value, handleChange }" name="theme.cards.hide_count">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  i18n._('Show article counts')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <FormField v-slot="{ value, handleChange }" name="theme.cards.show_icon_tile">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="value" @update:checked="handleChange" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  i18n._('Show icon background tile')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <FormField v-slot="{ value, handleChange }" name="theme.cards.show_authors">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="value" @update:checked="handleChange" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  i18n._('Show author avatars')
                }}</FormLabel>
              </FormItem>
            </FormField>
          </CollapsibleSection>

          <CollapsibleSection
            :title="i18n._('Article page')"
            :open="openSection === 'article'"
            @toggle="toggleSection('article')"
          >
            <FormField v-slot="{ value, handleChange }" name="theme.article.hide_toc">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  i18n._('Show table of contents')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <FormField v-slot="{ value, handleChange }" name="theme.article.hide_related">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="!value" @update:checked="(v) => handleChange(!v)" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  i18n._('Show related articles')
                }}</FormLabel>
              </FormItem>
            </FormField>
            <FormField v-slot="{ value, handleChange }" name="theme.article.show_author">
              <FormItem class="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Checkbox :checked="value" @update:checked="handleChange" />
                </FormControl>
                <FormLabel class="font-normal cursor-pointer">{{
                  i18n._('Show author')
                }}</FormLabel>
              </FormItem>
            </FormField>
          </CollapsibleSection>

          <CollapsibleSection
            :title="i18n._('Footer')"
            :open="openSection === 'footer'"
            @toggle="toggleSection('footer')"
          >
            <Tabs :model-value="editingScheme" @update:model-value="editingScheme = $event">
              <AdminToolbar>
                <template #left>
                  <TabsList class="w-max justify-start">
                    <TabsTrigger value="light" class="gap-2 h-8">
                      <Sun class="size-4" />
                      {{ i18n._('Light') }}
                    </TabsTrigger>
                    <TabsTrigger value="dark" class="gap-2 h-8">
                      <Moon class="size-4" />
                      {{ i18n._('Dark') }}
                    </TabsTrigger>
                  </TabsList>
                </template>
              </AdminToolbar>
            </Tabs>

            <div class="flex gap-4">
              <FormField
                v-slot="{ componentField }"
                :name="
                  editingScheme === 'dark'
                    ? 'theme.footer.background_color_dark'
                    : 'theme.footer.background_color'
                "
                keep-value
              >
                <FormItem class="flex-1">
                  <FormLabel>{{ i18n._('Background color') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      :placeholder="editingScheme === 'dark' ? '#16181b' : '#ffffff'"
                      v-bind="componentField"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField
                v-slot="{ componentField }"
                :name="
                  editingScheme === 'dark'
                    ? 'theme.footer.text_color_dark'
                    : 'theme.footer.text_color'
                "
                keep-value
              >
                <FormItem class="flex-1">
                  <FormLabel>{{ i18n._('Text color') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      :placeholder="editingScheme === 'dark' ? '#82878e' : '#909aa5'"
                      v-bind="componentField"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <FormField v-slot="{ componentField }" name="theme.footer.tagline">
              <FormItem>
                <FormLabel>{{ i18n._('Tagline') }}</FormLabel>
                <FormControl>
                  <Textarea
                    :rows="2"
                    :placeholder="i18n._('A company slogan or contact info')"
                    v-bind="componentField"
                  />
                </FormControl>
                <FormDescription>{{ i18n._('Markdown is supported, e.g. **bold**, *italic* and [links](https://example.com).') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <LinkListField name="theme.footer_links" :label="i18n.ngettext('Link', 'Links', 2)" />

            <LinkListField
              name="theme.social_links"
              :label="i18n._('Social links')"
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
                          <SelectValue :placeholder="i18n._('Platform')" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem v-for="(label, p) in socialPlatforms" :key="p" :value="p">{{
                            label
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
            :title="i18n._('Custom code')"
            :open="openSection === 'code'"
            @toggle="toggleSection('code')"
          >
            <FormField v-slot="{ componentField }" name="custom_css">
              <FormItem>
                <FormLabel>{{ i18n._('Custom CSS') }}</FormLabel>
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
                <FormLabel>{{ i18n._('Custom JS') }}</FormLabel>
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
        {{ i18n._('Cancel') }}
      </Button>
      <Button type="submit" :isLoading="isLoading">
        {{ submitLabel }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { watch, computed, ref, inject, onMounted, nextTick } from 'vue'
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
import { X, Sun, Moon } from 'lucide-vue-next'
import { RadioGroup, RadioGroupItem } from '@shared-ui/components/ui/radio-group'
import { Tabs, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import CollapsibleSection from './CollapsibleSection.vue'
import LinkListField from './LinkListField.vue'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import { useInboxStore } from '@/stores/inbox'
import { createHelpCenterFormSchema } from './helpCenterFormSchema.js'
import api from '@/api'
import { useI18n } from '@shared-ui/i18n'

// Field path prefix -> [tab, section] to reveal when that field errors on submit.
const FIELD_LOCATION = [
  ['theme.header', 'appearance', 'header'],
  ['theme.tagline', 'appearance', 'header'],
  ['theme.announcement', 'appearance', 'announcement'],
  ['theme.logo_url', 'appearance', 'brand'],
  ['theme.logo_url_dark', 'appearance', 'brand'],
  ['theme.color_scheme', 'appearance', 'brand'],
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

const SCHEME_FIELDS = {
  'theme.logo_url': 'light',
  'theme.color': 'light',
  'theme.logo_url_dark': 'dark',
  'theme.color_dark': 'dark',
  'theme.footer.background_color': 'light',
  'theme.footer.text_color': 'light',
  'theme.footer.background_color_dark': 'dark',
  'theme.footer.text_color_dark': 'dark'
}

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

const i18n = useI18n()
const inboxStore = useInboxStore()

const submitLabel = computed(() =>
  props.helpCenter ? i18n._('Update') : i18n._('Create')
)

const socialPlatforms = computed(() => ({
  website: i18n._('Website'),
  twitter: i18n._('X (Twitter)'),
  github: i18n._('GitHub'),
  linkedin: i18n._('LinkedIn'),
  facebook: i18n._('Facebook'),
  instagram: i18n._('Instagram'),
  youtube: i18n._('YouTube')
}))

const toFormValues = (hc) => ({
  name: hc?.name || '',
  slug: hc?.slug || '',
  template: hc?.template === 'docs' ? 'docs' : 'classic',
  custom_domain: hc?.custom_domain || '',
  livechat_inbox_id: hc?.livechat_inbox_id ? String(hc.livechat_inbox_id) : 'none',
  page_title: hc?.page_title || '',
  meta_description: hc?.meta_description || '',
  custom_css: hc?.custom_css || '',
  custom_js: hc?.custom_js || '',
  default_locale: hc?.default_locale || 'en',
  allowed_locales:
    Array.isArray(hc?.allowed_locales) && hc.allowed_locales.length ? hc.allowed_locales : ['en'],
  theme: {
    color_scheme: hc?.theme?.color_scheme || 'light',
    color: hc?.theme?.color || '#1f93ff',
    color_dark: hc?.theme?.color_dark || hc?.theme?.color || '#1f93ff',
    logo_url: hc?.theme?.logo_url || '',
    logo_url_dark: hc?.theme?.logo_url_dark || '',
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
        : i18n._('Popular articles')
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
      background_color_dark: hc?.theme?.footer?.background_color_dark || '',
      text_color_dark: hc?.theme?.footer?.text_color_dark || '',
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
  validationSchema: toTypedSchema(createHelpCenterFormSchema(i18n)),
  initialValues: toFormValues(props.helpCenter)
})

const {
  fields: localeFields,
  push: pushLocale,
  remove: removeLocale
} = useFieldArray('allowed_locales')

const isClassic = computed(() => form.values.template === 'classic')

const editingScheme = inject('helpCenterPreviewScheme', ref('light'))
watch(
  () => form.values.theme?.color_scheme,
  (scheme) => {
    editingScheme.value = scheme === 'dark' ? 'dark' : 'light'
  },
  { immediate: true }
)

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
  payload.livechat_inbox_id =
    payload.livechat_inbox_id && payload.livechat_inbox_id !== 'none'
      ? Number(payload.livechat_inbox_id)
      : null
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
      if (SCHEME_FIELDS[firstKey]) editingScheme.value = SCHEME_FIELDS[firstKey]
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
