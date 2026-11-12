<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <Tabs v-model="activeTab" class="w-full">
      <AdminToolbar>
        <template #left>
          <TabsList class="w-max justify-start">
            <TabsTrigger value="general">{{ i18n._('General') }}</TabsTrigger>
            <TabsTrigger value="appearance">{{ i18n._('Appearance') }}</TabsTrigger>
            <TabsTrigger value="content">{{ i18n.ngettext('Content', 'Contents', 1) }}</TabsTrigger>
            <TabsTrigger value="conversations">{{
              i18n.ngettext('Conversation', 'Conversations', 2)
            }}</TabsTrigger>
            <TabsTrigger value="setup">{{ i18n._('Setup') }}</TabsTrigger>
          </TabsList>
        </template>
      </AdminToolbar>

      <div class="mt-8">
        <div v-show="activeTab === 'general'">
          <Accordion type="single" collapsible v-model="activeSection" class="w-full">
            <AccordionItem value="basics" data-section="basics">
              <AccordionTrigger class="text-base">{{
                i18n._('General')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <FormField v-slot="{ componentField, handleChange }" name="enabled">
                  <FormItem>
                    <SwitchField
                      :checked="componentField.modelValue"
                      :title="i18n._('Enabled')"
                      @update:checked="handleChange"
                    />
                  </FormItem>
                </FormField>

                <FormField v-slot="{ componentField, handleChange }" name="csat_enabled">
                  <FormItem>
                    <SwitchField
                      :title="i18n._('CSAT Surveys')"
                      :description="i18n._('Send customer satisfaction surveys when a conversation is marked as resolved.')"
                      :checked="componentField.modelValue"
                      @update:checked="handleChange"
                    />
                  </FormItem>
                  <p :class="HINT_CLASS">
                    <Lightbulb class="size-4" />
                    <span>{{ i18n._('CSAT surveys are only sent once per conversation.') }}</span>
                  </p>
                </FormField>

                <FormField v-slot="{ componentField, handleChange }" name="prompt_tags_on_reply">
                  <FormItem>
                    <SwitchField
                      :title="i18n._('Prompt to tag before replying')"
                      :description="i18n._('Warn agents before they send a reply on a conversation that has no tags. Agents can still send without tagging by confirming the prompt.')"
                      :checked="componentField.modelValue"
                      @update:checked="handleChange"
                    />
                  </FormItem>
                </FormField>

                <FormGrid>
                  <FormField v-slot="{ componentField }" name="name">
                    <FormItem>
                      <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
                      <FormControl>
                        <Input type="text" placeholder="" v-bind="componentField" />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  </FormField>

                  <FormField v-slot="{ componentField }" name="config.brand_name">
                    <FormItem>
                      <FormLabel>{{ i18n._('Brand name') }}</FormLabel>
                      <FormControl>
                        <Input type="text" placeholder="" v-bind="componentField" />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  </FormField>

                  <FormField v-slot="{ componentField }" name="config.website_url">
                    <FormItem>
                      <FormLabel>{{ i18n._('Website URL') }}</FormLabel>
                      <FormControl>
                        <Input
                          type="url"
                          placeholder="https://example.com"
                          v-bind="componentField"
                        />
                      </FormControl>
                      <FormDescription>{{
                        i18n._('URL where the chat widget is installed. Used in continuity emails to link back to chat.')
                      }}</FormDescription>
                      <FormMessage />
                    </FormItem>
                  </FormField>
                </FormGrid>

                <FormGrid compact>
                  <FormField v-slot="{ componentField }" name="config.language">
                    <FormItem>
                      <FormLabel>{{ i18n.ngettext('Language', 'Languages', 1) }}</FormLabel>
                      <FormControl>
                        <Select v-bind="componentField">
                          <SelectTrigger>
                            <SelectValue :placeholder="i18n.pgettext('Application language setting', 'Select language')" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="auto">{{
                              i18n._('Auto-detect (browser language)')
                            }}</SelectItem>
                            <SelectItem
                              v-for="lang in availableLanguages"
                              :key="lang.code"
                              :value="lang.code"
                            >
                              {{ lang.name }}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      </FormControl>
                    </FormItem>
                  </FormField>

                  <FormField
                    v-if="form.values.config?.language === 'auto'"
                    v-slot="{ componentField }"
                    name="config.fallback_language"
                  >
                    <FormItem>
                      <FormLabel>{{ i18n._('Fallback language') }}</FormLabel>
                      <FormControl>
                        <Select v-bind="componentField">
                          <SelectTrigger>
                            <SelectValue :placeholder="i18n.pgettext('Application language setting', 'Select language')" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem
                              v-for="lang in availableLanguages"
                              :key="lang.code"
                              :value="lang.code"
                            >
                              {{ lang.name }}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      </FormControl>
                      <FormDescription>{{
                        i18n._('Language to use when visitor\'\'s browser language is not available.')
                      }}</FormDescription>
                    </FormItem>
                  </FormField>
                </FormGrid>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="continuity" data-section="continuity">
              <AccordionTrigger class="text-base">{{
                i18n._('Conversation continuity email inbox')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <FormGrid compact>
                  <FormField v-slot="{ componentField }" name="linked_email_inbox_id">
                    <FormItem class="col-span-full">
                      <FormLabel>{{ i18n._('Conversation continuity email inbox') }}</FormLabel>
                      <FormControl>
                        <Select v-bind="componentField">
                          <SelectTrigger>
                            <SelectValue :placeholder="i18n._('Select inbox')" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem :value="0">{{ i18n.pgettext('No linked inbox', 'None') }}</SelectItem>
                            <SelectItem
                              v-for="inbox in emailInboxes"
                              :key="inbox.id"
                              :value="inbox.id"
                            >
                              {{ inbox.name }}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      </FormControl>
                      <FormDescription>
                        {{ i18n._('When a contact goes offline, replies are sent via this email inbox. They can continue the conversation by replying to the email or returning to the chat widget.') }}
                      </FormDescription>
                    </FormItem>
                  </FormField>

                  <template v-if="form.values.linked_email_inbox_id">
                    <FormField
                      v-slot="{ componentField }"
                      name="config.continuity.offline_threshold"
                    >
                      <FormItem>
                        <FormLabel>{{
                          i18n._('Offline threshold')
                        }}</FormLabel>
                        <FormControl>
                          <Input type="text" placeholder="10m" v-bind="componentField" />
                        </FormControl>
                        <FormDescription>{{
                          i18n._('How long to wait after a contact goes offline before sending email. Min 1m.')
                        }}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField }"
                      name="config.continuity.max_messages_per_email"
                    >
                      <FormItem>
                        <FormLabel>{{
                          i18n._('Max messages per email')
                        }}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            :min="1"
                            :max="100"
                            placeholder="10"
                            v-bind="componentField"
                          />
                        </FormControl>
                        <FormDescription>{{
                          i18n._('Maximum number of messages to include in a single continuity email. Max 100.')
                        }}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField }"
                      name="config.continuity.min_email_interval"
                    >
                      <FormItem>
                        <FormLabel>{{
                          i18n._('Min email interval')
                        }}</FormLabel>
                        <FormControl>
                          <Input type="text" placeholder="15m" v-bind="componentField" />
                        </FormControl>
                        <FormDescription>{{
                          i18n._('Minimum time between continuity emails to the same contact. Min 1m.')
                        }}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </template>
                </FormGrid>
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>

        <div v-show="activeTab === 'appearance'">
          <Accordion type="single" collapsible v-model="activeSection" class="w-full">
            <AccordionItem value="appearance" data-section="appearance">
              <AccordionTrigger class="text-base">{{ i18n._('Theme') }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <FormField v-slot="{ componentField }" name="config.theme">
                  <FormItem>
                    <FormLabel>{{ i18n._('Theme') }}</FormLabel>
                    <FormControl>
                      <RadioGroup v-bind="componentField" class="flex gap-4">
                        <div class="flex items-center space-x-2">
                          <RadioGroupItem id="theme-system" value="system" />
                          <Label for="theme-system">{{ i18n._('Match system') }}</Label>
                        </div>
                        <div class="flex items-center space-x-2">
                          <RadioGroupItem id="theme-light" value="light" />
                          <Label for="theme-light">{{ i18n._('Light') }}</Label>
                        </div>
                        <div class="flex items-center space-x-2">
                          <RadioGroupItem id="theme-dark" value="dark" />
                          <Label for="theme-dark">{{ i18n._('Dark') }}</Label>
                        </div>
                      </RadioGroup>
                    </FormControl>
                    <FormDescription>{{ i18n._('Match system follows the visitor\'\'s own color scheme.') }}</FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <FormField v-slot="{ componentField, handleChange }" name="config.show_powered_by">
                  <FormItem>
                    <SwitchField
                      :title="i18n._('Show powered by')"
                      :description="i18n._('Show \u0022Powered by Libredesk\u0022 when a conversation is open in the widget')"
                      :checked="componentField.modelValue"
                      @update:checked="handleChange"
                    />
                  </FormItem>
                </FormField>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="branding" data-section="branding">
              <AccordionTrigger class="text-base">{{
                i18n._('Branding')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <div class="space-y-4">
                  <Tabs :model-value="editingTheme" @update:model-value="editingTheme = $event">
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
                    :name="`${brandingPath}.logo_url`"
                    keep-value
                  >
                    <FormItem>
                      <FormLabel>{{ i18n._('Logo URL') }}</FormLabel>
                      <FormControl>
                        <Input
                          type="url"
                          placeholder="https://example.com/logo.png"
                          v-bind="componentField"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  </FormField>

                  <div class="grid grid-cols-2 gap-4">
                    <FormField
                      v-slot="{ componentField }"
                      :name="`${brandingPath}.colors.primary`"
                      keep-value
                    >
                      <FormItem>
                        <FormLabel>{{ i18n.ngettext('Primary color', 'Primary colors', 1) }}</FormLabel>
                        <FormControl>
                          <Input type="color" class="h-9 w-24 p-1" v-bind="componentField" />
                        </FormControl>
                        <FormMessage />
                        <p
                          v-if="lowPrimaryContrast"
                          class="text-sm text-destructive flex items-start gap-1.5"
                        >
                          <TriangleAlert class="size-4 shrink-0 mt-0.5" />
                          <span>{{
                            i18n._('The primary color is nearly the same as the widget background. Buttons and message bubbles may be hard to see. Pick a color with more contrast.')
                          }}</span>
                        </p>
                      </FormItem>
                    </FormField>
                  </div>

                  <h5 class="ui-subheading pt-2">
                    {{ i18n._('Home screen') }}
                  </h5>

                  <FormField
                    v-slot="{ componentField }"
                    :name="`${brandingPath}.home_screen.header_text_color`"
                    keep-value
                  >
                    <FormItem>
                      <FormLabel>{{ i18n._('Header text color') }}</FormLabel>
                      <FormControl>
                        <RadioGroup v-bind="componentField" class="flex gap-4">
                          <div class="flex items-center space-x-2">
                            <RadioGroupItem id="text-black" value="black" />
                            <Label for="text-black">{{ i18n._('Black') }}</Label>
                          </div>
                          <div class="flex items-center space-x-2">
                            <RadioGroupItem id="text-white" value="white" />
                            <Label for="text-white">{{ i18n._('White') }}</Label>
                          </div>
                        </RadioGroup>
                      </FormControl>
                      <FormDescription>{{
                        i18n._('Color of the greeting and introduction text on the home screen')
                      }}</FormDescription>
                      <p
                        v-if="lowHeaderContrast"
                        class="text-sm text-destructive flex items-start gap-1.5"
                      >
                        <TriangleAlert class="size-4 shrink-0 mt-0.5" />
                        <span>{{
                          i18n._('Header text and background colors are too similar. The greeting text may be hard to read. Switch the text color or adjust the background.')
                        }}</span>
                      </p>
                    </FormItem>
                  </FormField>

                  <FormField
                    v-slot="{ componentField }"
                    :name="`${brandingPath}.home_screen.background.type`"
                    keep-value
                  >
                    <FormItem>
                      <FormLabel>{{ i18n._('Background') }}</FormLabel>
                      <FormControl>
                        <RadioGroup
                          v-bind="componentField"
                          @update:model-value="onBackgroundTypeChange"
                          class="flex gap-4"
                        >
                          <div class="flex items-center space-x-2">
                            <RadioGroupItem id="bg-solid" value="solid" />
                            <Label for="bg-solid">{{ i18n._('Solid') }}</Label>
                          </div>
                          <div class="flex items-center space-x-2">
                            <RadioGroupItem id="bg-gradient" value="gradient" />
                            <Label for="bg-gradient">{{ i18n._('Gradient') }}</Label>
                          </div>
                          <div class="flex items-center space-x-2">
                            <RadioGroupItem id="bg-image" value="image" />
                            <Label for="bg-image">{{ i18n.ngettext('Image', 'Images', 1) }}</Label>
                          </div>
                        </RadioGroup>
                      </FormControl>
                    </FormItem>
                  </FormField>

                  <div v-if="activeBackgroundType === 'solid'" class="grid grid-cols-2 gap-4">
                    <FormField
                      v-slot="{ componentField }"
                      :name="`${brandingPath}.home_screen.background.color`"
                      keep-value
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Background color') }}</FormLabel>
                        <FormControl>
                          <Input type="color" class="h-9 w-24 p-1" v-bind="componentField" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </div>

                  <div v-if="activeBackgroundType === 'gradient'" class="flex gap-8">
                    <FormField
                      v-slot="{ componentField }"
                      :name="`${brandingPath}.home_screen.background.gradient_start`"
                      keep-value
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Gradient start') }}</FormLabel>
                        <FormControl>
                          <Input type="color" class="h-9 w-24 p-1" v-bind="componentField" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                    <FormField
                      v-slot="{ componentField }"
                      :name="`${brandingPath}.home_screen.background.gradient_end`"
                      keep-value
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Gradient end') }}</FormLabel>
                        <FormControl>
                          <Input type="color" class="h-9 w-24 p-1" v-bind="componentField" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </div>

                  <FormField
                    v-if="activeBackgroundType === 'image'"
                    v-slot="{ componentField }"
                    :name="`${brandingPath}.home_screen.background.image_url`"
                    keep-value
                  >
                    <FormItem>
                      <FormLabel>{{ i18n._('Background image URL') }}</FormLabel>
                      <FormControl>
                        <Input
                          type="url"
                          placeholder="https://example.com/background.jpg"
                          v-bind="componentField"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  </FormField>

                  <FormField
                    v-slot="{ componentField, handleChange }"
                    :name="`${brandingPath}.home_screen.fade_background`"
                    keep-value
                  >
                    <FormItem>
                      <SwitchField
                        :title="i18n._('Fade background')"
                        :description="
                          i18n._('Blend the header background into the content area below')
                        "
                        :checked="componentField.modelValue"
                        @update:checked="handleChange"
                      />
                    </FormItem>
                  </FormField>
                </div>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="launcher" data-section="launcher">
              <AccordionTrigger class="text-base">{{
                i18n._('Launcher')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <div class="space-y-4">
                  <Tabs :model-value="editingTheme" @update:model-value="editingTheme = $event">
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

                  <div class="grid grid-cols-2 gap-4">
                    <FormField
                      v-slot="{ componentField }"
                      :name="`${brandingPath}.launcher.logo_url`"
                      keep-value
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Launcher logo') }}</FormLabel>
                        <FormControl>
                          <Input
                            type="url"
                            placeholder="https://example.com/launcher-logo.png"
                            v-bind="componentField"
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField }"
                      :name="`${brandingPath}.launcher.color`"
                      keep-value
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Launcher color') }}</FormLabel>
                        <FormControl>
                          <Input type="color" class="h-9 w-24 p-1" v-bind="componentField" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </div>

                  <FormField v-slot="{ componentField }" name="config.launcher.position">
                    <FormItem>
                      <FormLabel>{{ i18n._('Position') }}</FormLabel>
                      <FormControl>
                        <Select v-bind="componentField">
                          <SelectTrigger>
                            <SelectValue :placeholder="i18n._('Select position')" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="left">{{
                              i18n._('Left')
                            }}</SelectItem>
                            <SelectItem value="right">{{
                              i18n._('Right')
                            }}</SelectItem>
                          </SelectContent>
                        </Select>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  </FormField>

                  <FormGrid compact>
                    <FormField v-slot="{ componentField }" name="config.launcher.icon_scale">
                      <FormItem>
                        <FormLabel>{{ i18n._('Icon size') }}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            placeholder="100"
                            min="40"
                            max="100"
                            v-bind="componentField"
                          />
                        </FormControl>
                        <FormDescription>{{
                          i18n._('How much of the launcher the logo fills, as a percentage. Lower it to inset a logo that has no padding of its own.')
                        }}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField v-slot="{ componentField }" name="config.launcher.spacing.side">
                      <FormItem>
                        <FormLabel>{{
                          i18n._('Side spacing')
                        }}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            placeholder="20"
                            min="0"
                            max="200"
                            v-bind="componentField"
                          />
                        </FormControl>
                        <FormDescription>{{
                          i18n._('Distance from the side of the screen in pixels')
                        }}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField v-slot="{ componentField }" name="config.launcher.spacing.bottom">
                      <FormItem>
                        <FormLabel>{{
                          i18n._('Bottom spacing')
                        }}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            placeholder="20"
                            min="0"
                            max="200"
                            v-bind="componentField"
                          />
                        </FormControl>
                        <FormDescription>{{
                          i18n._('Distance from the bottom of the screen in pixels')
                        }}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </FormGrid>
                </div>
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>

        <div v-show="activeTab === 'content'">
          <Accordion type="single" collapsible v-model="activeSection" class="w-full">
            <AccordionItem value="messages" data-section="messages">
              <AccordionTrigger class="text-base">{{
                i18n._('Messages')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <FormField v-slot="{ componentField }" name="config.greeting_message">
                  <FormItem>
                    <FormLabel>{{ i18n._('Greeting message') }}</FormLabel>
                    <FormControl>
                      <Textarea
                        v-bind="componentField"
                        :placeholder="i18n._('Welcome! How can we help you today?')"
                        rows="2"
                      />
                    </FormControl>
                    <FormDescription>{{
                      i18n._(literalExampleMessage1)
                    }}</FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <FormField v-slot="{ componentField }" name="config.introduction_message">
                  <FormItem>
                    <FormLabel>{{ i18n._('Introduction message') }}</FormLabel>
                    <FormControl>
                      <Textarea
                        v-bind="componentField"
                        :placeholder="i18n._('We\'\'re here to help!')"
                        rows="2"
                      />
                    </FormControl>
                    <FormDescription>{{
                      i18n._(literalExampleMessage2)
                    }}</FormDescription>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <FormField v-slot="{ componentField }" name="config.chat_introduction">
                  <FormItem>
                    <FormLabel>{{ i18n._('Chat introduction') }}</FormLabel>
                    <FormControl>
                      <Textarea
                        v-bind="componentField"
                        :placeholder="i18n._('Ask us anything, or share your feedback.')"
                        rows="2"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="noticeBanner" data-section="noticeBanner">
              <AccordionTrigger class="text-base">{{
                i18n._('Notice banner')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <div class="space-y-4">
                  <FormField
                    v-slot="{ componentField, handleChange }"
                    name="config.notice_banner.enabled"
                  >
                    <FormItem>
                      <SwitchField
                        :title="i18n._('Enable notice banner')"
                        :checked="componentField.modelValue"
                        @update:checked="handleChange"
                      />
                    </FormItem>
                  </FormField>

                  <FormField
                    v-slot="{ componentField }"
                    name="config.notice_banner.text"
                    v-if="form.values.config?.notice_banner?.enabled"
                  >
                    <FormItem>
                      <FormLabel>{{ i18n._('Notice banner text') }}</FormLabel>
                      <FormControl>
                        <Textarea
                          v-bind="componentField"
                          :placeholder="i18n._('Our response times are slower than usual. We\'\'re working hard to get to your message.')"
                          rows="2"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  </FormField>
                </div>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="homeApps" data-section="homeApps">
              <AccordionTrigger class="text-base">{{
                i18n.ngettext('Home screen app', 'Home screen apps', 2)
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <div class="space-y-4">
                  <FormField name="config.home_apps">
                    <FormItem>
                      <div class="space-y-3">
                        <Draggable
                          v-model="homeApps"
                          item-key="index"
                          :animation="reducedMotion === 'reduce' ? 0 : 200"
                          handle=".drag-handle"
                          :force-fallback="true"
                          fallback-on-body
                          :fallback-tolerance="3"
                          ghost-class="drag-ghost"
                          class="space-y-3"
                          @change="updateHomeApps"
                        >
                          <template #item="{ element: item, index }">
                            <div class="flex min-w-0 items-start gap-2 p-3 border rounded-md">
                              <div
                                class="drag-handle shrink-0 cursor-move text-muted-foreground pt-2"
                              >
                                <GripVertical class="w-4 h-4" />
                              </div>
                              <div class="min-w-0 flex-1">
                                <div class="text-xs text-muted-foreground mb-2">
                                  {{ homeAppLabel(item.type) }}
                                </div>
                                <p
                                  v-if="item.type === 'help'"
                                  class="ui-description"
                                >
                                  {{ i18n._('Shows a search button and the featured articles set under Help center.') }}
                                </p>
                                <div
                                  v-else-if="item.type === 'announcement'"
                                  class="flex flex-col gap-2"
                                >
                                  <Input
                                    v-model="item.title"
                                    :placeholder="i18n.ngettext('Title', 'Titles', 1)"
                                    @change="updateHomeApps"
                                  />
                                  <Textarea
                                    v-model="item.description"
                                    :placeholder="i18n.ngettext('Description', 'Descriptions', 1)"
                                    rows="6"
                                    @change="updateHomeApps"
                                  />
                                  <FormGrid compact class="gap-2">
                                    <Input
                                      v-model="item.image_url"
                                      type="url"
                                      :placeholder="i18n._('Cover image URL')"
                                      @change="updateHomeApps"
                                    />
                                    <Input
                                      v-model="item.url"
                                      type="url"
                                      :placeholder="i18n._('Link URL')"
                                      @change="updateHomeApps"
                                    />
                                  </FormGrid>
                                </div>
                                <FormGrid
                                  v-else-if="item.type === 'external_link'"
                                  compact
                                  class="gap-2"
                                >
                                  <Input
                                    v-model="item.text"
                                    :placeholder="i18n._('Link Text')"
                                    @change="updateHomeApps"
                                  />
                                  <Input
                                    v-model="item.url"
                                    placeholder="https://example.com"
                                    @change="updateHomeApps"
                                  />
                                </FormGrid>
                              </div>
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                class="shrink-0"
                                :aria-label="i18n._('Remove')"
                                @click="removeHomeApp(index)"
                              >
                                <X class="w-4 h-4" aria-hidden="true" />
                              </Button>
                            </div>
                          </template>
                        </Draggable>

                        <div class="flex flex-wrap gap-2">
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            @click="addHomeApp('announcement')"
                          >
                            <Plus class="w-4 h-4" />
                            {{ i18n._('Add announcement') }}
                          </Button>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            @click="addHomeApp('external_link')"
                          >
                            <Plus class="w-4 h-4" />
                            {{ i18n._('Add external link') }}
                          </Button>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            :disabled="!form.values.config.help?.help_center_id || hasHelpHomeApp"
                            @click="addHomeApp('help')"
                          >
                            <Plus class="w-4 h-4" />
                            {{ i18n._('Add help articles') }}
                          </Button>
                        </div>
                        <p
                          v-if="!form.values.config.help?.help_center_id"
                          class="text-xs text-muted-foreground"
                        >
                          {{ i18n._('Choose a help center under Help center before adding this card.') }}
                        </p>
                        <p
                          v-if="showHomeAppsError && incompleteHomeApps"
                          class="text-sm text-destructive flex items-start gap-1.5"
                        >
                          <TriangleAlert class="size-4 shrink-0 mt-0.5" />
                          <span>{{ i18n._('Fill in or remove the incomplete home screen app.') }}</span>
                        </p>
                      </div>
                      <FormMessage />
                    </FormItem>
                  </FormField>
                </div>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="help" data-section="help">
              <AccordionTrigger class="text-base">{{
                i18n.ngettext('Help center', 'Help centers', 1)
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <WidgetHelpConfig
                  :model-value="form.values.config.help"
                  :centers="helpCenters"
                  :articles="helpArticles"
                  :failed="helpFailed"
                  @update:model-value="form.setFieldValue('config.help', $event, false)"
                />
                <p
                  v-if="Object.keys(form.errors.value).some((key) => key.startsWith('config.help'))"
                  role="alert"
                  class="text-sm text-destructive"
                >
                  {{ i18n._('Invalid value') }}
                </p>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="campaigns" data-section="campaigns">
              <AccordionTrigger class="text-base">
                {{ i18n.ngettext('Proactive message', 'Proactive messages', 2) }}
              </AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <WidgetCampaigns
                  ref="campaignsRef"
                  :model-value="form.values.config.campaigns || []"
                  :inbox-id="initialValues.id || 0"
                  :brand-name="form.values.config.brand_name || ''"
                  :cooldown="form.values.config.campaign_cooldown ?? '24h'"
                  :show-errors="showCampaignErrors"
                  @update:model-value="updateCampaigns"
                  @update:cooldown="updateCampaignCooldown"
                  @update:preview="previewCampaign = $event"
                />
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>

        <div v-show="activeTab === 'conversations'">
          <Accordion type="single" collapsible v-model="activeSection" class="w-full">
            <AccordionItem value="features" data-section="features">
              <AccordionTrigger class="text-base">{{
                i18n._('Features')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <div class="space-y-4">
                  <div class="space-y-3">
                    <FormField
                      v-slot="{ componentField, handleChange }"
                      name="config.features.file_upload"
                    >
                      <FormItem>
                        <SwitchField
                          :title="i18n._('File upload')"
                          :description="i18n._('Allow users to upload files in chat')"
                          :checked="componentField.modelValue"
                          @update:checked="handleChange"
                        />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange }"
                      name="config.features.emoji"
                    >
                      <FormItem>
                        <SwitchField
                          :title="i18n._('Emoji support')"
                          :description="i18n._('Allow users to use emojis in chat')"
                          :checked="componentField.modelValue"
                          @update:checked="handleChange"
                        />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange }"
                      name="config.features.transcript"
                    >
                      <FormItem>
                        <SwitchField
                          :title="i18n._('Download transcript')"
                          :checked="componentField.modelValue"
                          @update:checked="handleChange"
                        />
                      </FormItem>
                    </FormField>
                  </div>
                </div>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="officeHours" data-section="officeHours">
              <AccordionTrigger class="text-base">{{
                i18n._('Office hours')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <div class="space-y-4">
                  <FormField
                    v-slot="{ componentField, handleChange }"
                    name="config.show_office_hours_in_chat"
                  >
                    <FormItem>
                      <SwitchField
                        :title="i18n._('Show office hours in chat')"
                        :description="i18n._('Show when the team will be next available')"
                        :checked="componentField.modelValue"
                        @update:checked="handleChange"
                      />
                    </FormItem>
                  </FormField>

                  <FormField
                    v-slot="{ componentField, handleChange }"
                    name="config.show_office_hours_after_assignment"
                  >
                    <FormItem>
                      <SwitchField
                        :title="i18n._('Show office hours after team assignment')"
                        :description="
                          i18n._('Show office hours after conversation is assigned to a team')
                        "
                        :checked="componentField.modelValue"
                        :disabled="!form.values.config.show_office_hours_in_chat"
                        @update:checked="handleChange"
                      />
                    </FormItem>
                  </FormField>

                  <FormField
                    v-if="form.values.config.show_office_hours_in_chat"
                    v-slot="{ componentField }"
                    name="config.chat_reply_expectation_message"
                  >
                    <FormItem>
                      <FormLabel>{{
                        i18n._('Chat reply expectation message')
                      }}</FormLabel>
                      <FormControl>
                        <Input type="text" v-bind="componentField" />
                      </FormControl>
                      <FormDescription>
                        {{ i18n._('Message shown to during business hours about expected reply times.') }}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  </FormField>
                </div>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="prechat" data-section="prechat">
              <AccordionTrigger class="text-base">{{
                i18n._('Pre-chat form')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <PreChatFormConfig v-model="prechatConfig" />
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="users" data-section="users">
              <AccordionTrigger class="text-base">{{ i18n._('Users') }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <Tabs :model-value="selectedUserTab" @update:model-value="selectedUserTab = $event">
                  <AdminToolbar>
                    <template #left>
                      <TabsList class="w-max justify-start">
                        <TabsTrigger value="visitors">
                          {{ i18n._('Visitors') }}
                        </TabsTrigger>
                        <TabsTrigger value="users">
                          {{ i18n._('Users') }}
                        </TabsTrigger>
                      </TabsList>
                    </template>
                  </AdminToolbar>

                  <div class="space-y-4 mt-4">
                    <div
                      v-show="selectedUserTab === 'visitors'"
                      data-audience="visitors"
                      class="space-y-4"
                    >
                      <FormField
                        v-slot="{ componentField }"
                        name="config.visitors.start_conversation_button_text"
                      >
                        <FormItem>
                          <FormLabel>{{
                            i18n._('Start conversation button text')
                          }}</FormLabel>
                          <FormControl>
                            <Input
                              v-bind="componentField"
                              :placeholder="i18n._('Start conversation')"
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      </FormField>

                      <FormField v-slot="{ componentField }" name="config.visitors.quick_replies">
                        <FormItem>
                          <FormLabel>{{ i18n.ngettext('Quick reply', 'Quick replies', 2) }}</FormLabel>
                          <FormControl>
                            <Textarea
                              v-bind="componentField"
                              :placeholder="i18n._('I have a question about billing\nHow do I reset my password?')"
                              rows="6"
                            />
                          </FormControl>
                          <FormDescription>{{
                            i18n._('One reply per line, up to 6. Shown when someone starts a new conversation.')
                          }}</FormDescription>
                          <FormMessage />
                        </FormItem>
                      </FormField>

                      <FormField
                        v-slot="{ componentField, handleChange }"
                        name="config.visitors.direct_to_conversation"
                      >
                        <FormItem>
                          <SwitchField
                            :title="i18n._('Launch directly into conversation')"
                            :description="
                              i18n._('Skip the home screen and open directly into the user\'\'s latest conversation, or a new conversation if none exists')
                            "
                            :checked="componentField.modelValue"
                            @update:checked="handleChange"
                          />
                        </FormItem>
                      </FormField>

                      <FormField
                        v-slot="{ componentField, handleChange }"
                        name="config.visitors.allow_start_conversation"
                      >
                        <FormItem>
                          <SwitchField
                            :title="i18n._('Allow start conversation')"
                            :description="
                              i18n._('Allow visitors to start new conversations')
                            "
                            :checked="componentField.modelValue"
                            @update:checked="handleChange"
                          />
                        </FormItem>
                      </FormField>

                      <FormField
                        v-slot="{ componentField, handleChange }"
                        name="config.visitors.prevent_multiple_conversations"
                      >
                        <FormItem>
                          <SwitchField
                            :title="i18n._('Prevent multiple conversations')"
                            :description="
                              i18n._('Prevent visitors from starting multiple conversations simultaneously'
                              )
                            "
                            :checked="componentField.modelValue"
                            @update:checked="handleChange"
                          />
                        </FormItem>
                      </FormField>

                      <FormField
                        v-slot="{ componentField, handleChange }"
                        name="config.visitors.prevent_reply_to_closed_conversation"
                      >
                        <FormItem>
                          <SwitchField
                            :title="i18n._('Prevent replying to closed conversations')"
                            :description="
                              i18n._('When enabled, users cannot reply to closed conversations'
                              )
                            "
                            :checked="componentField.modelValue"
                            @update:checked="handleChange"
                          />
                        </FormItem>
                      </FormField>
                    </div>

                    <div
                      v-show="selectedUserTab === 'users'"
                      data-audience="users"
                      class="space-y-4"
                    >
                      <FormField
                        v-slot="{ componentField }"
                        name="config.users.start_conversation_button_text"
                      >
                        <FormItem>
                          <FormLabel>{{
                            i18n._('Start conversation button text')
                          }}</FormLabel>
                          <FormControl>
                            <Input
                              v-bind="componentField"
                              :placeholder="i18n._('Start conversation')"
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      </FormField>

                      <FormField v-slot="{ componentField }" name="config.users.quick_replies">
                        <FormItem>
                          <FormLabel>{{ i18n.ngettext('Quick reply', 'Quick replies', 2) }}</FormLabel>
                          <FormControl>
                            <Textarea
                              v-bind="componentField"
                              :placeholder="i18n._('I have a question about billing\nHow do I reset my password?')"
                              rows="6"
                            />
                          </FormControl>
                          <FormDescription>{{
                            i18n._('One reply per line, up to 6. Shown when someone starts a new conversation.')
                          }}</FormDescription>
                          <FormMessage />
                        </FormItem>
                      </FormField>

                      <FormField
                        v-slot="{ componentField, handleChange }"
                        name="config.users.direct_to_conversation"
                      >
                        <FormItem>
                          <SwitchField
                            :title="i18n._('Launch directly into conversation')"
                            :description="
                              i18n._('Skip the home screen and open directly into the user\'\'s latest conversation, or a new conversation if none exists')
                            "
                            :checked="componentField.modelValue"
                            @update:checked="handleChange"
                          />
                        </FormItem>
                      </FormField>

                      <FormField
                        v-slot="{ componentField, handleChange }"
                        name="config.users.allow_start_conversation"
                      >
                        <FormItem>
                          <SwitchField
                            :title="i18n._('Allow start conversation')"
                            :description="
                              i18n._('Allow users to start new conversations')
                            "
                            :checked="componentField.modelValue"
                            @update:checked="handleChange"
                          />
                        </FormItem>
                      </FormField>

                      <FormField
                        v-slot="{ componentField, handleChange }"
                        name="config.users.prevent_multiple_conversations"
                      >
                        <FormItem>
                          <SwitchField
                            :title="i18n._('Prevent multiple conversations')"
                            :description="
                              i18n._('Prevent users from starting multiple conversations simultaneously'
                              )
                            "
                            :checked="componentField.modelValue"
                            @update:checked="handleChange"
                          />
                        </FormItem>
                      </FormField>

                      <FormField
                        v-slot="{ componentField, handleChange }"
                        name="config.users.prevent_reply_to_closed_conversation"
                      >
                        <FormItem>
                          <SwitchField
                            :title="i18n._('Prevent replying to closed conversations')"
                            :description="
                              i18n._('When enabled, users cannot reply to closed conversations'
                              )
                            "
                            :checked="componentField.modelValue"
                            @update:checked="handleChange"
                          />
                        </FormItem>
                      </FormField>
                    </div>
                  </div>
                </Tabs>
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>

        <div v-show="activeTab === 'setup'">
          <Accordion type="single" collapsible v-model="activeSection" class="w-full">
            <AccordionItem value="installation" data-section="installation">
              <AccordionTrigger class="text-base">{{
                i18n._('Installation')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <div class="space-y-4">
                  <ol class="text-sm space-y-2 list-decimal list-inside text-muted-foreground">
                    <li>{{ i18n._('Copy the code snippet below') }}</li>
                    <li>{{ i18n._('Paste it just before the closing `body` tag on every page where you want the live chat widget to appear') }}</li>
                  </ol>
                </div>

                <div class="relative">
                  <CodeEditor :modelValue="integrationSnippet" language="html" :readOnly="true" />
                  <CopyButton :text="integrationSnippet" class="absolute top-3 right-3" />
                </div>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="identity" data-section="identity">
              <AccordionTrigger class="text-base">{{
                i18n._('Identity verification')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <div class="space-y-4">
                  <div class="space-y-1">
                    <p class="ui-description">
                      {{ i18n._('Verify logged-in users so conversations are linked to their real accounts and users cannot impersonate each other.') }}
                    </p>
                    <p class="ui-description">
                      {{ i18n._('Your server generates a signed JWT containing the user\'\'s details. Use the secret key from the Security tab to sign it with the HS256 algorithm.') }}
                    </p>
                  </div>

                  <div class="relative">
                    <CodeEditor
                      :modelValue="jwtPayloadExample"
                      language="javascript"
                      :readOnly="true"
                    />
                    <CopyButton :text="jwtPayloadExample" class="absolute top-3 right-3" />
                  </div>

                  <p class="ui-description">
                    {{ i18n._('Add the signed JWT to LibredeskSettings:') }}
                  </p>

                  <div class="relative">
                    <CodeEditor
                      :modelValue="authenticatedIntegrationSnippet"
                      language="html"
                      :readOnly="true"
                    />
                    <CopyButton
                      :text="authenticatedIntegrationSnippet"
                      class="absolute top-3 right-3"
                    />
                  </div>

                  <p class="text-sm text-destructive flex items-center gap-1.5">
                    <TriangleAlert class="size-4 shrink-0" />
                    {{ i18n._('Never expose the secret key in client-side code. Always generate the JWT on your server.') }}
                  </p>
                </div>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="jsApi" data-section="jsApi">
              <AccordionTrigger class="text-base">{{
                i18n._('JavaScript API')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <div class="space-y-4">
                  <p class="ui-description">
                    {{ i18n._('Control the widget programmatically from your application.') }}
                  </p>

                  <div class="relative">
                    <CodeEditor :modelValue="jsApiSnippet" language="javascript" :readOnly="true" />
                    <CopyButton :text="jsApiSnippet" class="absolute top-3 right-3" />
                  </div>
                </div>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="security" data-section="security">
              <AccordionTrigger class="text-base">{{
                i18n._('Security')
              }}</AccordionTrigger>
              <AccordionContent force-mount class="space-y-8 pt-2">
                <FormGrid>
                  <FormField v-slot="{ componentField }" name="secret">
                    <FormItem>
                      <FormLabel>{{ i18n._('Secret key') }}</FormLabel>
                      <FormControl>
                        <Input type="password" v-bind="componentField" />
                      </FormControl>
                      <FormDescription>{{
                        i18n._('Set a secret key to secure the chat widget.')
                      }}</FormDescription>
                      <FormMessage />
                      <p
                        v-if="weakSecret"
                        :class="HINT_CLASS"
                      >
                        <TriangleAlert class="size-4 shrink-0 mt-0.5" />
                        <span>{{ i18n._('This secret is short, which weakens JWT signing. Use at least 32 random characters.') }}</span>
                      </p>
                    </FormItem>
                  </FormField>

                  <FormField v-slot="{ componentField }" name="config.session_duration">
                    <FormItem>
                      <FormLabel>{{ i18n._('Authenticated user session duration') }}</FormLabel>
                      <FormControl>
                        <Input type="text" placeholder="10h" v-bind="componentField" />
                      </FormControl>
                      <FormDescription>{{
                        i18n._('How long authenticated user sessions stay active. Sessions auto-extend on activity. Minimum 1 hour. Example: 24h, 168h, 720h, 2160h.')
                      }}</FormDescription>
                      <FormMessage />
                    </FormItem>
                  </FormField>
                </FormGrid>

                <FormGrid>
                  <FormField v-slot="{ componentField }" name="config.trusted_domains">
                    <FormItem>
                      <FormLabel>{{ i18n._('Domain list') }}</FormLabel>
                      <FormControl>
                        <Textarea
                          v-bind="componentField"
                          placeholder="example.com&#10;*.example.com&#10;another-domain.com"
                          rows="4"
                        />
                      </FormControl>
                      <FormDescription>{{
                        i18n._('One domain per line. *.example.com matches subdomains only, add the bare domain example.com separately. Leave empty to allow any domain.')
                      }}</FormDescription>
                      <FormMessage />
                    </FormItem>
                  </FormField>

                  <FormField v-slot="{ componentField }" name="config.blocked_ips">
                    <FormItem>
                      <FormLabel>{{ i18n._('IP block list') }}</FormLabel>
                      <FormControl>
                        <Textarea
                          v-bind="componentField"
                          placeholder="192.168.1.0/24&#10;10.0.0.1&#10;2001:db8::/32"
                          rows="4"
                        />
                      </FormControl>
                      <FormDescription>{{
                        i18n._('Enter IP addresses or CIDR ranges to block, one per line. Blocked IPs will be denied access to the widget.')
                      }}</FormDescription>
                      <FormMessage />
                    </FormItem>
                  </FormField>
                </FormGrid>
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>
      </div>
    </Tabs>

    <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
      {{ submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import { defineMessage as msg } from '@lingui/core/macro'

const literalExampleMessage1 = msg({ message: 'Available variables: \'{{.FirstName}}\', \'{{.LastName}}\'. Use | for fallback, e.g. Hi \'{{.FirstName | there}}\'' })
const literalExampleMessage2 = msg({ message: 'Available variables: \'{{.FirstName}}\', \'{{.LastName}}\'. Use | for fallback, e.g. Hi \'{{.FirstName | there}}\'' })

const HINT_CLASS = '!mt-2 text-muted-foreground text-xs flex items-start gap-1.5'

import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { usePreferredReducedMotion } from '@vueuse/core'
import { watch, computed, ref, inject, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import {
  createFormSchema,
  defaultWidgetHelp,
  normalizeAudienceConfig,
  normalizePrechatConfig
} from './livechatFormSchema.js'
import WidgetHelpConfig from './WidgetHelpConfig.vue'
import { useHelpCenterArticles } from './useHelpCenterArticles.js'
import WidgetCampaigns from './WidgetCampaigns.vue'
import { useInboxStore } from '@/stores/inbox'
import api from '@/api'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Button } from '@shared-ui/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@shared-ui/components/ui/accordion'
import { RadioGroup, RadioGroupItem } from '@shared-ui/components/ui/radio-group'
import { Label } from '@shared-ui/components/ui/label'
import { Plus, X, TriangleAlert, GripVertical, Lightbulb, Sun, Moon } from 'lucide-vue-next'
import Draggable from 'vuedraggable'
import { useI18n } from '@shared-ui/i18n'
import { useRoute, useRouter } from 'vue-router'
import PreChatFormConfig, { getDefaultPrechatFields } from './PreChatFormConfig.vue'
import { useAppSettingsStore } from '@/stores/appSettings'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import CopyButton from '@/components/button/CopyButton.vue'
import CodeEditor from '@/components/editor/CodeEditor.vue'
import { contrastRatio } from '@shared-ui/utils/color'

// Warn only when a color is nearly indistinguishable from what it sits on; merely-low
// (but perceptible) contrast is left to the user's judgment, so this sits below WCAG's 3.
const MIN_CONTRAST = 2
const HEX_COLOR = /^#([0-9a-f]{6}|[0-9a-f]{3})$/i
// Widget page background shown when no explicit header color is set (mirrors --background in main.scss).
const WIDGET_BG = { light: '#ffffff', dark: '#1a1a1e' }
const DEFAULT_GRADIENT_START = '#2563eb'
const DEFAULT_GRADIENT_END = '#1e40af'
const DEFAULT_LAUNCHER_ICON_SCALE = 100
const MASKED_SECRET = '••••••••••'

const TABS = ['general', 'appearance', 'content', 'conversations', 'setup']

const SECTION_TAB = {
  basics: 'general',
  continuity: 'general',
  appearance: 'appearance',
  branding: 'appearance',
  launcher: 'appearance',
  messages: 'content',
  noticeBanner: 'content',
  homeApps: 'content',
  help: 'content',
  campaigns: 'content',
  features: 'conversations',
  officeHours: 'conversations',
  prechat: 'conversations',
  users: 'conversations',
  installation: 'setup',
  identity: 'setup',
  jsApi: 'setup',
  security: 'setup'
}

const LEGACY_TAB_SECTION = {
  appearance: 'branding',
  messages: 'messages',
  features: 'features',
  campaigns: 'campaigns',
  help: 'help',
  prechat: 'prechat',
  users: 'users',
  security: 'security',
  installation: 'installation'
}

// Ordered: specific prefixes before the general-tab fallbacks.
const FIELD_SECTION = [
  ['config.help', 'help'],
  ['config.campaign', 'campaigns'],
  ['config.branding', 'branding'],
  ['config.theme', 'appearance'],
  ['config.launcher', 'launcher'],
  ['config.home_apps', 'homeApps'],
  ['config.notice_banner', 'noticeBanner'],
  ['config.greeting_message', 'messages'],
  ['config.introduction_message', 'messages'],
  ['config.chat_introduction', 'messages'],
  ['config.chat_reply_expectation_message', 'officeHours'],
  ['config.features', 'features'],
  ['config.prechat_form', 'prechat'],
  ['config.visitors', 'users'],
  ['config.users', 'users'],
  ['config.session_duration', 'security'],
  ['config.trusted_domains', 'security'],
  ['config.blocked_ips', 'security'],
  ['secret', 'security'],
  ['config.continuity', 'continuity'],
  ['config.brand_name', 'basics'],
  ['config.website_url', 'basics'],
  ['config.language', 'basics'],
  ['name', 'basics']
]

const reducedMotion = usePreferredReducedMotion()

const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  availableLanguages: {
    type: Array,
    default: () => []
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    default: ''
  },
  isNewForm: {
    type: Boolean,
    default: false
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const i18n = useI18n()
const route = useRoute()
const router = useRouter()
const legacySection = LEGACY_TAB_SECTION[route.query.tab]
const activeTab = ref(
  TABS.includes(route.query.tab) ? route.query.tab : SECTION_TAB[legacySection] || 'general'
)
const activeSection = ref(legacySection || '')
watch(activeTab, (tab) => router.replace({ query: { ...route.query, tab } }))

const openSection = (section) => {
  activeTab.value = SECTION_TAB[section]
  activeSection.value = section
}
const selectedUserTab = ref('visitors')
const homeApps = ref([])
const prechatConfig = ref({
  enabled: false,
  handoff_only: false,
  title: '',
  fields: getDefaultPrechatFields(),
  visitors: {
    enabled: true,
    title: '',
    fields: getDefaultPrechatFields()
  },
  users: {
    enabled: true,
    title: '',
    fields: getDefaultPrechatFields()
  }
})

const inboxStore = useInboxStore()
const appSettingsStore = useAppSettingsStore()
const emitter = useEmitter()

const emailInboxes = computed(() =>
  inboxStore.inboxes.filter((inbox) => inbox.channel === 'email' && inbox.enabled)
)

const baseUrl = computed(() => {
  return appSettingsStore.settings?.['app.root_url'] || window.location.origin
})

const inboxUUID = computed(() => props.initialValues?.uuid || '<INBOX_UUID>')

const integrationSnippet = computed(() => {
  return `<script>
  window.LibredeskSettings = {
    baseURL: '${baseUrl.value}',
    inboxID: '${inboxUUID.value}'
  };
<\/script>
<script async src="${baseUrl.value}/widget.js"><\/script>`
})

const jwtPayloadExample = computed(() => {
  return `{
  "external_user_id": "your_app_user_123",    // Required: Your system's unique user ID
  "email": "user@example.com",                // Required: User's email
  "first_name": "John",                       // Required: User's first name
  "last_name": "Doe",                         // Optional: User's last name
  "phone_number": "9876543210",                // Optional: User's phone number (e.g. "199999999999")
  "phone_number_country_code": "IN",          // Optional: ISO 3166-1 alpha-2 country code (e.g. "IN", "US")
  "exp": 1735689600,                          // Required: Expiration time (Unix timestamp in seconds)
  "contact_custom_attributes": {              // Optional: Contact-level attributes
    "plan": "premium",
    "company": "Acme Inc"
  }
}`
})

const authenticatedIntegrationSnippet = computed(() => {
  return `<script>
  window.LibredeskSettings = {
    baseURL: '${baseUrl.value}',
    inboxID: '${inboxUUID.value}',
    userJWT: 'YOUR_SIGNED_JWT_TOKEN_HERE' // Generated by your server
  };
<\/script>
<script async src="${baseUrl.value}/widget.js"><\/script>`
})

const jsApiSnippet = computed(() => {
  return `window.Libredesk.show();
window.Libredesk.hide();
window.Libredesk.toggle();
window.Libredesk.isVisible();

window.Libredesk.setLanguage('en-US');
window.Libredesk.trackEvent('checkout_started');

window.Libredesk.setUser('SIGNED_JWT_TOKEN');
window.Libredesk.logout();

window.Libredesk.onShow(function() {});
window.Libredesk.onHide(function() {});
window.Libredesk.onUnreadCountChange(function(count) {});

window.Libredesk.destroy();`
})

function defaultBranding(theme) {
  const isDark = theme === 'dark'
  return {
    colors: { primary: '#2563eb' },
    logo_url: '',
    launcher: { logo_url: '', color: '#000000' },
    home_screen: {
      header_text_color: isDark ? 'white' : 'black',
      background: {
        type: 'solid',
        color: isDark ? WIDGET_BG.dark : WIDGET_BG.light,
        gradient_start: DEFAULT_GRADIENT_START,
        gradient_end: DEFAULT_GRADIENT_END,
        image_url: ''
      },
      fade_background: false
    }
  }
}

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n)),
  initialValues: {
    name: '',
    enabled: true,
    secret: '',
    csat_enabled: false,
    prompt_tags_on_reply: false,
    linked_email_inbox_id: null,
    config: {
      help: defaultWidgetHelp(),
      campaigns: [],
      campaign_cooldown: '24h',
      brand_name: '',
      website_url: '',
      theme: 'light',
      show_powered_by: true,
      language: 'en-US',
      fallback_language: 'en-US',
      branding: {
        light: defaultBranding('light'),
        dark: defaultBranding('dark')
      },
      launcher: {
        position: 'right',
        icon_scale: DEFAULT_LAUNCHER_ICON_SCALE,
        spacing: {
          side: 20,
          bottom: 20
        }
      },
      greeting_message: 'Hello {{.FirstName | there}}',
      introduction_message: 'How can we help?',
      chat_introduction: 'Ask us anything, or share your feedback.',
      show_office_hours_in_chat: false,
      show_office_hours_after_assignment: false,
      chat_reply_expectation_message: 'We typically reply in 5 minutes.',
      notice_banner: {
        enabled: false,
        text: 'Our response times are slower than usual. We regret the inconvenience caused.'
      },
      features: {
        transcript: props.isNewForm,
        file_upload: true,
        emoji: true
      },
      continuity: {
        offline_threshold: '10m',
        max_messages_per_email: 10,
        min_email_interval: '15m'
      },
      session_duration: '10h',
      direct_to_conversation: false,
      trusted_domains: '',
      blocked_ips: '',
      home_apps: [],
      visitors: {
        start_conversation_button_text: 'Start conversation',
        allow_start_conversation: true,
        prevent_multiple_conversations: false,
        prevent_reply_to_closed_conversation: false,
        quick_replies: '',
        direct_to_conversation: false
      },
      users: {
        start_conversation_button_text: 'Start conversation',
        allow_start_conversation: true,
        prevent_multiple_conversations: false,
        prevent_reply_to_closed_conversation: false,
        quick_replies: '',
        direct_to_conversation: false
      },
      prechat_form: {
        enabled: false,
        handoff_only: false,
        title: '',
        fields: getDefaultPrechatFields(),
        visitors: {
          enabled: true,
          title: '',
          fields: getDefaultPrechatFields()
        },
        users: {
          enabled: true,
          title: '',
          fields: getDefaultPrechatFields()
        }
      }
    }
  }
})

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.isNewForm ? i18n._('Create') : i18n._('Save'))
  )
})

const theme = computed(() => form.values.config?.theme || 'light')
const editingTheme = inject('livechatPreviewTheme', ref('light'))
const brandingPath = computed(() => `config.branding.${editingTheme.value}`)
const activeBranding = computed(() => form.values.config?.branding?.[editingTheme.value])
const activeBackgroundType = computed(() => activeBranding.value?.home_screen?.background?.type)

// A forced theme renders one branding half, so that is the half to edit.
watch(
  theme,
  (value) => {
    editingTheme.value = value === 'dark' ? 'dark' : 'light'
    if (value !== 'light' && !form.values.config?.branding?.dark) {
      form.setFieldValue('config.branding.dark', defaultBranding('dark'), false)
    }
  },
  { immediate: true }
)

const lowHeaderContrast = computed(() => {
  const hs = activeBranding.value?.home_screen
  if (!hs?.background) return false

  const textColor = hs.header_text_color === 'black' ? '#000000' : '#ffffff'
  const pageBg = WIDGET_BG[editingTheme.value]
  // An empty/unset color renders the widget's page background, so measure against that.
  const isLow = (bg) => HEX_COLOR.test(bg) && contrastRatio(textColor, bg) < MIN_CONTRAST

  switch (hs.background.type) {
    case 'solid':
      return isLow(hs.background.color || pageBg)
    case 'gradient':
      return isLow(hs.background.gradient_start) || isLow(hs.background.gradient_end)
    default:
      return false
  }
})

// Primary is used as a fill (buttons, message bubbles, badges) over the widget background,
// so warn if it blends into the background of the configured light/dark mode.
const lowPrimaryContrast = computed(() => {
  const primary = activeBranding.value?.colors?.primary
  if (!HEX_COLOR.test(primary)) return false
  return contrastRatio(primary, WIDGET_BG[editingTheme.value]) < MIN_CONTRAST
})

// Advisory only: a short secret weakens HS256 JWT signing. Not enforced, since a hard
// minimum would force rotating existing secrets and break live integrations.
const weakSecret = computed(() => {
  const s = form.values.secret
  return typeof s === 'string' && s !== MASKED_SECRET && s.length > 0 && s.length < 32
})

const helpCenters = ref([])
const helpCenterID = computed(() => form.values.config?.help?.help_center_id || 0)
const {
  tree: helpTree,
  articles: helpArticles,
  failed: helpFailed
} = useHelpCenterArticles(helpCenterID)

// The campaign the admin is editing, mirrored in the preview as an invitation card.
const previewCampaign = ref(null)
const campaignsRef = ref(null)
const showCampaignErrors = ref(false)
const hasCampaignErrors = computed(() =>
  Object.keys(form.errors.value).some((field) => field.startsWith('config.campaign'))
)
const updateCampaigns = (campaigns) =>
  form.setFieldValue('config.campaigns', campaigns, hasCampaignErrors.value)
const updateCampaignCooldown = (duration) =>
  form.setFieldValue('config.campaign_cooldown', duration, hasCampaignErrors.value)

// home_apps in form.values only syncs on change events, so pull the live ref for the preview.
const previewConfig = computed(() => {
  const branding = form.values.config?.branding?.[editingTheme.value] || {}
  return {
    ...form.values.config,
    ...branding,
    dark: editingTheme.value === 'dark',
    launcher: { ...form.values.config?.launcher, ...branding.launcher },
    home_apps: homeApps.value,
    help_tree: helpTree.value,
    help_articles: helpArticles.value,
    preview_campaign: activeSection.value === 'campaigns' ? previewCampaign.value : null
  }
})

// InboxView renders the preview in the help rail; feed it this form's live config while mounted.
const livechatPreview = inject('livechatPreview', null)
watch(
  previewConfig,
  (cfg) => {
    if (livechatPreview) livechatPreview.value = cfg
  },
  { immediate: true, deep: true }
)
onBeforeUnmount(() => {
  if (livechatPreview) livechatPreview.value = null
})

// Switching to gradient with no colors set would render a blank picker (black),
// so seed sensible defaults while retaining any colors already chosen.
const onBackgroundTypeChange = (type) => {
  if (type !== 'gradient') return
  const bg = activeBranding.value?.home_screen?.background || {}
  if (!bg.gradient_start) {
    form.setFieldValue(
      `${brandingPath.value}.home_screen.background.gradient_start`,
      DEFAULT_GRADIENT_START
    )
  }
  if (!bg.gradient_end) {
    form.setFieldValue(
      `${brandingPath.value}.home_screen.background.gradient_end`,
      DEFAULT_GRADIENT_END
    )
  }
}

const addHomeApp = (type) => {
  if (type === 'announcement') {
    homeApps.value.push({
      type: 'announcement',
      title: '',
      description: '',
      image_url: '',
      url: ''
    })
  } else if (type === 'help') {
    homeApps.value.push({ type: 'help' })
  } else {
    homeApps.value.push({ type: 'external_link', text: '', url: '' })
  }
  updateHomeApps()
}

const HOME_APP_LABELS = {
  announcement: msg({ message: '{count, plural, one {Announcement} other {Announcements}}' }),
  external_link: msg({ message: 'External links' }),
  help: msg({ message: '{count, plural, one {Help center} other {Help centers}}' })
}
const homeAppLabel = (type) => i18n._(HOME_APP_LABELS[type], 1)

const hasHelpHomeApp = computed(() => homeApps.value.some((item) => item.type === 'help'))

const removeHomeApp = (index) => {
  homeApps.value.splice(index, 1)
  updateHomeApps()
}

const updateHomeApps = () => {
  showHomeAppsError.value = false
  form.setFieldValue('config.home_apps', homeApps.value)
}

const isHomeAppEmpty = (item) => {
  if (item.type === 'help') return false
  return item.type === 'announcement'
    ? !item.title && !item.description && !item.image_url && !item.url
    : !item.text && !item.url
}

const isHomeAppComplete = (item) => {
  if (item.type === 'help') return true
  return item.type === 'announcement'
    ? Boolean(item.title && item.image_url && item.url)
    : Boolean(item.text && item.url)
}

const incompleteHomeApps = computed(() =>
  homeApps.value.some((item) => !isHomeAppEmpty(item) && !isHomeAppComplete(item))
)

const showHomeAppsError = ref(false)

const textareaToLines = (value) =>
  typeof value === 'string'
    ? value
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean)
    : []

onMounted(async () => {
  inboxStore.fetchInboxes()
  appSettingsStore.fetchPublicConfig()
  try {
    helpCenters.value = (await api.getHelpCenters()).data.data || []
  } catch {
    helpFailed.value = true
  }
})

const onSubmit = form.handleSubmit(
  async (values) => {
    showCampaignErrors.value = false
    values.config.trusted_domains = textareaToLines(values.config.trusted_domains)
    values.config.visitors.quick_replies = textareaToLines(values.config.visitors.quick_replies)
    values.config.users.quick_replies = textareaToLines(values.config.users.quick_replies)
    values.config.blocked_ips = textareaToLines(values.config.blocked_ips)

    // Reject partially filled rows to avoid silently discarding typed values.
    if (incompleteHomeApps.value) {
      showHomeAppsError.value = true
      openSection('homeApps')
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: i18n._('Fill in or remove the incomplete home screen app.')
      })
      return
    }
    values.config.home_apps = homeApps.value.filter((item) => !isHomeAppEmpty(item))

    if (!values.linked_email_inbox_id) {
      values.linked_email_inbox_id = null
      values.config.continuity = {}
    }

    const pc = JSON.parse(JSON.stringify(prechatConfig.value))
    for (const audience of ['visitors', 'users']) {
      const audienceConfig = pc[audience]
      if (
        audienceConfig.enabled &&
        audienceConfig.fields?.length > 0 &&
        !audienceConfig.fields.some((field) => field.enabled)
      ) {
        audienceConfig.enabled = false
      }
    }
    values.config.prechat_form = pc

    await props.submitForm(values)
  },
  ({ errors }) => {
    const firstKey = Object.keys(errors)[0]
    if (!firstKey) return
    if (firstKey.startsWith('config.branding.')) {
      editingTheme.value = firstKey.split('.')[2] === 'dark' ? 'dark' : 'light'
    }
    const match = FIELD_SECTION.find(
      ([prefix]) => firstKey === prefix || firstKey.startsWith(prefix)
    )
    if (match) openSection(match[1])
    if (firstKey.startsWith('config.campaign')) {
      showCampaignErrors.value = true
      nextTick(() => campaignsRef.value?.showInvalidField(firstKey))
    }
  }
)

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }

    if (Array.isArray(newValues.config?.trusted_domains)) {
      newValues.config.trusted_domains = newValues.config.trusted_domains.join('\n')
    }

    const audienceConfigs = Object.fromEntries(
      ['visitors', 'users'].map((audience) => [
        audience,
        normalizeAudienceConfig(newValues.config, audience)
      ])
    )

    if (Array.isArray(newValues.config?.blocked_ips)) {
      newValues.config.blocked_ips = newValues.config.blocked_ips.join('\n')
    }

    if (newValues.config?.home_apps) {
      // Copy each app: mutating a shared app object retriggers the watcher and resets the form.
      homeApps.value = newValues.config.home_apps.map((app) => ({ ...app }))
    }

    const pc = normalizePrechatConfig(newValues.config?.prechat_form)
    for (const audience of ['visitors', 'users']) {
      const existingFields = pc[audience].fields
      if (existingFields.length === 0) {
        pc[audience].fields = getDefaultPrechatFields()
        continue
      }
      const existingKeys = new Set(existingFields.map((field) => field.key))
      let nextOrder = existingFields.reduce((max, field) => Math.max(max, field.order || 0), 0)
      const missing = getDefaultPrechatFields()
        .filter((field) => !existingKeys.has(field.key))
        .map((field) => ({ ...field, order: ++nextOrder }))
      pc[audience].fields = [...existingFields, ...missing]
    }
    prechatConfig.value = pc

    form.setValues(
      {
        ...newValues,
        config: {
          ...newValues.config,
          ...audienceConfigs,
          prechat_form: pc,
          campaigns: newValues.config?.campaigns || [],
          campaign_cooldown: newValues.config?.campaign_cooldown || '24h',
          help: newValues.config?.help || defaultWidgetHelp(),
          features: {
            ...newValues.config?.features,
            transcript: newValues.config?.features?.transcript ?? false
          }
        }
      },
      false
    )
  },
  { deep: true, immediate: true }
)
</script>
