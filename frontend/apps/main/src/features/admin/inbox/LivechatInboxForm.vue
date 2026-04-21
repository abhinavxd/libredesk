<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <!-- Main Tabs -->
    <Tabs v-model="activeTab" class="w-full">
      <TabsList class="flex flex-wrap gap-1 h-auto p-1 w-fit">
        <TabsTrigger value="general">{{ $t('globals.terms.general') }}</TabsTrigger>
        <TabsTrigger value="appearance">{{
          $t('admin.inbox.livechat.tabs.appearance')
        }}</TabsTrigger>
        <TabsTrigger value="messages">{{ $t('admin.inbox.livechat.tabs.messages') }}</TabsTrigger>
        <TabsTrigger value="features">{{ $t('globals.terms.features') }}</TabsTrigger>
        <TabsTrigger value="prechat">{{ $t('admin.inbox.livechat.tabs.prechat') }}</TabsTrigger>
        <TabsTrigger value="users">{{ $t('globals.terms.users') }}</TabsTrigger>
        <TabsTrigger value="security">{{ $t('globals.terms.security') }}</TabsTrigger>
        <TabsTrigger value="installation">{{
          $t('admin.inbox.livechat.tabs.installation')
        }}</TabsTrigger>
      </TabsList>

      <div class="mt-8">
        <!-- General Tab -->
        <div v-show="activeTab === 'general'" class="space-y-8">
          <FormField v-slot="{ componentField, handleChange }" name="enabled">
            <FormItem>
              <SwitchField
                :checked="componentField.modelValue"
                :title="$t('globals.terms.enabled')"
                @update:checked="handleChange"
              />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField, handleChange }" name="csat_enabled">
            <FormItem>
              <SwitchField
                :title="$t('admin.inbox.csatSurveys')"
                :description="$t('admin.inbox.csatSurveys.description_1')"
                :checked="componentField.modelValue"
                @update:checked="handleChange"
              />
            </FormItem>
            <p class="!mt-2 text-muted-foreground text-xs flex items-start gap-1.5">
              <Lightbulb class="size-4" />
              <span>{{ $t('admin.inbox.csatSurveys.description_3') }}</span>
            </p>
          </FormField>

          <FormField v-slot="{ componentField, handleChange }" name="prompt_tags_on_reply">
            <FormItem>
              <SwitchField
                :title="$t('admin.inbox.promptTagsOnReply')"
                :description="$t('admin.inbox.promptTagsOnReply.description')"
                :checked="componentField.modelValue"
                @update:checked="handleChange"
              />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="name">
            <FormItem>
              <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
              <FormControl>
                <Input type="text" placeholder="" v-bind="componentField" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="config.brand_name">
            <FormItem>
              <FormLabel>{{ $t('globals.terms.brandName') }}</FormLabel>
              <FormControl>
                <Input type="text" placeholder="" v-bind="componentField" />
              </FormControl>
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="config.website_url">
            <FormItem>
              <FormLabel>{{ $t('admin.inbox.livechat.websiteUrl') }}</FormLabel>
              <FormControl>
                <Input type="url" placeholder="https://example.com" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{ $t('admin.inbox.livechat.websiteUrl.description') }}</FormDescription>
            </FormItem>
          </FormField>

          <div class="grid grid-cols-2 gap-4">
            <!-- Language -->
            <FormField v-slot="{ componentField }" name="config.language">
              <FormItem>
                <FormLabel>{{ $t('globals.terms.language') }}</FormLabel>
                <FormControl>
                  <Select v-bind="componentField">
                    <SelectTrigger>
                      <SelectValue :placeholder="$t('admin.general.language.placeholder')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="auto">{{ $t('admin.inbox.livechat.language.auto') }}</SelectItem>
                      <SelectItem v-for="lang in availableLanguages" :key="lang.code" :value="lang.code">
                        {{ lang.name }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
              </FormItem>
            </FormField>

            <!-- Fallback Language (shown only when auto-detect is selected) -->
            <FormField v-if="form.values.config?.language === 'auto'" v-slot="{ componentField }" name="config.fallback_language">
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.fallbackLanguage') }}</FormLabel>
                <FormControl>
                  <Select v-bind="componentField">
                    <SelectTrigger>
                      <SelectValue :placeholder="$t('admin.general.language.placeholder')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="lang in availableLanguages" :key="lang.code" :value="lang.code">
                        {{ lang.name }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormDescription>{{ $t('admin.inbox.livechat.fallbackLanguage.description') }}</FormDescription>
              </FormItem>
            </FormField>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <!-- Email Fallback Inbox -->
            <FormField v-slot="{ componentField }" name="linked_email_inbox_id">
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.conversationContinuity') }}</FormLabel>
                <FormControl>
                  <Select v-bind="componentField">
                    <SelectTrigger>
                      <SelectValue
                        :placeholder="$t('placeholders.selectInbox')"
                      />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem :value="0">{{ $t('globals.terms.none') }}</SelectItem>
                      <SelectItem v-for="inbox in emailInboxes" :key="inbox.id" :value="inbox.id">
                        {{ inbox.name }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormDescription>
                  {{ $t('admin.inbox.livechat.conversationContinuity.description') }}
                </FormDescription>
              </FormItem>
            </FormField>

            <template v-if="form.values.linked_email_inbox_id">
            <FormField v-slot="{ componentField }" name="config.continuity.offline_threshold">
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.continuity.offlineThreshold') }}</FormLabel>
                <FormControl>
                  <Input type="text" placeholder="10m" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ $t('admin.inbox.livechat.continuity.offlineThreshold.description') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="config.continuity.max_messages_per_email">
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.continuity.maxMessagesPerEmail') }}</FormLabel>
                <FormControl>
                  <Input type="number" :min="1" :max="100" placeholder="10" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ $t('admin.inbox.livechat.continuity.maxMessagesPerEmail.description') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="config.continuity.min_email_interval">
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.continuity.minEmailInterval') }}</FormLabel>
                <FormControl>
                  <Input type="text" placeholder="15m" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ $t('admin.inbox.livechat.continuity.minEmailInterval.description') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
            </template>
          </div>

        </div>

        <!-- Appearance Tab -->
        <div v-show="activeTab === 'appearance'" class="space-y-8">
          <!-- Logo URL -->
          <FormField v-slot="{ componentField }" name="config.logo_url">
            <FormItem>
              <FormLabel>{{ $t('globals.terms.logoUrl') }}</FormLabel>
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

          <!-- Dark mode -->
          <FormField v-slot="{ componentField, handleChange }" name="config.dark_mode">
            <FormItem>
              <SwitchField
                :title="$t('admin.inbox.livechat.darkMode')"
                :description="$t('admin.inbox.livechat.darkMode.description')"
                :checked="componentField.modelValue"
                @update:checked="handleChange"
              />
            </FormItem>
          </FormField>

          <!-- Show Powered By -->
          <FormField v-slot="{ componentField, handleChange }" name="config.show_powered_by">
            <FormItem>
              <SwitchField
                :title="$t('admin.inbox.livechat.showPoweredBy')"
                :description="$t('admin.inbox.livechat.showPoweredBy.description')"
                :checked="componentField.modelValue"
                @update:checked="handleChange"
              />
            </FormItem>
          </FormField>

          <!-- Colors -->
          <div class="space-y-4">
            <h4 class="text-base font-semibold text-foreground">{{ $t('admin.inbox.livechat.colors') }}</h4>
            <div class="grid grid-cols-2 gap-4">
              <FormField v-slot="{ componentField }" name="config.colors.primary">
                <FormItem>
                  <FormLabel>{{ $t('globals.terms.primaryColor', 1) }}</FormLabel>
                  <FormControl>
                    <Input type="color" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
          </div>

          <!-- Home Screen -->
          <div class="space-y-4">
            <h4 class="text-base font-semibold text-foreground">{{ $t('globals.terms.homeScreen') }}</h4>

            <!-- Header Text Color -->
            <FormField v-slot="{ componentField }" name="config.home_screen.header_text_color">
              <FormItem>
                <FormLabel>{{ $t('globals.messages.headerTextColor') }}</FormLabel>
                <FormControl>
                  <RadioGroup v-bind="componentField" class="flex gap-4">
                    <div class="flex items-center space-x-2">
                      <RadioGroupItem id="text-black" value="black" />
                      <Label for="text-black">{{ $t('globals.terms.black') }}</Label>
                    </div>
                    <div class="flex items-center space-x-2">
                      <RadioGroupItem id="text-white" value="white" />
                      <Label for="text-white">{{ $t('globals.terms.white') }}</Label>
                    </div>
                  </RadioGroup>
                </FormControl>
                <FormDescription>{{ $t('admin.inbox.livechat.homeScreen.headerTextColor.description') }}</FormDescription>
              </FormItem>
            </FormField>

            <!-- Background Type -->
            <FormField v-slot="{ componentField }" name="config.home_screen.background.type">
              <FormItem>
                <FormLabel>{{ $t('globals.terms.background') }}</FormLabel>
                <FormControl>
                  <RadioGroup v-bind="componentField" class="flex gap-4">
                    <div class="flex items-center space-x-2">
                      <RadioGroupItem id="bg-solid" value="solid" />
                      <Label for="bg-solid">{{ $t('globals.terms.solid') }}</Label>
                    </div>
                    <div class="flex items-center space-x-2">
                      <RadioGroupItem id="bg-gradient" value="gradient" />
                      <Label for="bg-gradient">{{ $t('globals.terms.gradient') }}</Label>
                    </div>
                    <div class="flex items-center space-x-2">
                      <RadioGroupItem id="bg-image" value="image" />
                      <Label for="bg-image">{{ $t('globals.terms.image', 1) }}</Label>
                    </div>
                  </RadioGroup>
                </FormControl>
              </FormItem>
            </FormField>

            <!-- Solid color picker -->
            <div v-if="form.values.config?.home_screen?.background?.type === 'solid'" class="grid grid-cols-2 gap-4">
              <FormField v-slot="{ componentField }" name="config.home_screen.background.color">
                <FormItem>
                  <FormLabel>{{ $t('globals.messages.backgroundColor') }}</FormLabel>
                  <FormControl>
                    <Input type="color" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <!-- Gradient color pickers -->
            <div v-if="form.values.config?.home_screen?.background?.type === 'gradient'" class="grid grid-cols-2 gap-4">
              <FormField v-slot="{ componentField }" name="config.home_screen.background.gradient_start">
                <FormItem>
                  <FormLabel>{{ $t('globals.messages.gradientStart') }}</FormLabel>
                  <FormControl>
                    <Input type="color" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField }" name="config.home_screen.background.gradient_end">
                <FormItem>
                  <FormLabel>{{ $t('globals.messages.gradientEnd') }}</FormLabel>
                  <FormControl>
                    <Input type="color" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <!-- Image URL -->
            <FormField v-if="form.values.config?.home_screen?.background?.type === 'image'" v-slot="{ componentField }" name="config.home_screen.background.image_url">
              <FormItem>
                <FormLabel>{{ $t('globals.messages.backgroundImageUrl') }}</FormLabel>
                <FormControl>
                  <Input type="url" placeholder="https://example.com/background.jpg" v-bind="componentField" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <!-- Fade Background -->
            <FormField v-slot="{ componentField, handleChange }" name="config.home_screen.fade_background">
              <FormItem>
                <SwitchField
                  :title="$t('admin.inbox.livechat.homeScreen.fadeBackground')"
                  :description="$t('admin.inbox.livechat.homeScreen.fadeBackground.description')"
                  :checked="componentField.modelValue"
                  @update:checked="handleChange"
                />
              </FormItem>
            </FormField>
          </div>

          <!-- Home Screen Apps -->
          <div class="space-y-4">
            <h4 class="text-base font-semibold text-foreground">{{ $t('globals.terms.homeScreenApp', 2) }}</h4>

            <FormField name="config.home_apps">
              <FormItem>
                <div class="space-y-3">
                  <Draggable v-model="homeApps" item-key="index" :animation="200" handle=".drag-handle" class="space-y-3" @change="updateHomeApps">
                    <template #item="{ element: item, index }">
                      <div class="flex items-start gap-2 p-3 border rounded">
                        <div class="drag-handle cursor-move text-muted-foreground pt-2">
                          <GripVertical class="w-4 h-4" />
                        </div>
                        <div class="flex-1">
                          <div class="text-xs text-muted-foreground mb-2">
                            {{ item.type === 'announcement' ? $t('globals.terms.announcement') : $t('admin.inbox.livechat.externalLinks') }}
                          </div>
                          <!-- Announcement fields -->
                          <div v-if="item.type === 'announcement'" class="flex flex-col gap-2">
                            <Input v-model="item.title" :placeholder="$t('globals.terms.title')" @change="updateHomeApps" />
                            <Textarea v-model="item.description" :placeholder="$t('globals.terms.description')" rows="6" @change="updateHomeApps" />
                            <div class="grid grid-cols-2 gap-2">
                              <Input v-model="item.image_url" type="url" :placeholder="$t('globals.messages.coverImageUrl')" @change="updateHomeApps" />
                              <Input v-model="item.url" type="url" :placeholder="$t('globals.messages.linkUrl')" @change="updateHomeApps" />
                            </div>
                          </div>
                          <!-- External link fields -->
                          <div v-else class="grid grid-cols-2 gap-2">
                            <Input v-model="item.text" :placeholder="$t('placeholders.linkText')" @change="updateHomeApps" />
                            <Input v-model="item.url" placeholder="https://example.com" @change="updateHomeApps" />
                          </div>
                        </div>
                        <Button type="button" variant="ghost" size="sm" @click="removeHomeApp(index)">
                          <X class="w-4 h-4" />
                        </Button>
                      </div>
                    </template>
                  </Draggable>

                  <div class="flex gap-2">
                    <Button type="button" variant="outline" size="sm" @click="addHomeApp('announcement')">
                      <Plus class="w-4 h-4"/>
                      {{ $t('globals.messages.addAnnouncement') }}
                    </Button>
                    <Button type="button" variant="outline" size="sm" @click="addHomeApp('external_link')">
                      <Plus class="w-4 h-4"/>
                      {{ $t('globals.messages.addExternalLink') }}
                    </Button>
                  </div>
                </div>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>

          <!-- Launcher Configuration -->
          <div class="space-y-4">
            <h4 class="text-base font-semibold text-foreground">{{ $t('admin.inbox.livechat.launcher') }}</h4>

            <div class="grid grid-cols-2 gap-4">
              <!-- Launcher Position -->
              <FormField v-slot="{ componentField }" name="config.launcher.position">
                <FormItem>
                  <FormLabel>{{ $t('admin.inbox.livechat.launcher.position') }}</FormLabel>
                  <FormControl>
                    <Select v-bind="componentField">
                      <SelectTrigger>
                        <SelectValue :placeholder="$t('placeholders.selectPosition')" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="left">{{
                          $t('admin.inbox.livechat.launcher.position.left')
                        }}</SelectItem>
                        <SelectItem value="right">{{
                          $t('admin.inbox.livechat.launcher.position.right')
                        }}</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Launcher Logo -->
              <FormField v-slot="{ componentField }" name="config.launcher.logo_url">
                <FormItem>
                  <FormLabel>{{ $t('admin.inbox.livechat.launcher.logo') }}</FormLabel>
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
            </div>

            <div class="grid grid-cols-2 gap-4">
              <!-- Launcher Color -->
              <FormField v-slot="{ componentField }" name="config.launcher.color">
                <FormItem>
                  <FormLabel>{{ $t('admin.inbox.livechat.launcher.color') }}</FormLabel>
                  <FormControl>
                    <Input type="color" v-bind="componentField" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <!-- Launcher Spacing Side -->
              <FormField v-slot="{ componentField }" name="config.launcher.spacing.side">
                <FormItem>
                  <FormLabel>{{ $t('admin.inbox.livechat.launcher.spacing.side') }}</FormLabel>
                  <FormControl>
                    <Input type="number" placeholder="20" min="0" max="200" v-bind="componentField" />
                  </FormControl>
                  <FormDescription>{{
                    $t('admin.inbox.livechat.launcher.spacing.side.description')
                  }}</FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>

              <!-- Launcher Spacing Bottom -->
              <FormField v-slot="{ componentField }" name="config.launcher.spacing.bottom">
                <FormItem>
                  <FormLabel>{{ $t('admin.inbox.livechat.launcher.spacing.bottom') }}</FormLabel>
                  <FormControl>
                    <Input type="number" placeholder="20" min="0" max="200" v-bind="componentField" />
                  </FormControl>
                  <FormDescription>{{
                    $t('admin.inbox.livechat.launcher.spacing.bottom.description')
                  }}</FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
          </div>
        </div>

        <!-- Messages Tab -->
        <div v-show="activeTab === 'messages'" class="space-y-8">
          <FormField v-slot="{ componentField }" name="config.greeting_message">
            <FormItem>
              <FormLabel>{{ $t('admin.inbox.livechat.greetingMessage') }}</FormLabel>
              <FormControl>
                <Textarea
                  v-bind="componentField"
                  :placeholder="$t('placeholders.greetingMessage')"
                  rows="2"
                />
              </FormControl>
              <FormDescription>{{
                $t('admin.inbox.livechat.greetingMessage.variables')
              }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="config.introduction_message">
            <FormItem>
              <FormLabel>{{ $t('admin.inbox.livechat.introductionMessage') }}</FormLabel>
              <FormControl>
                <Textarea v-bind="componentField" :placeholder="$t('placeholders.introductionMessage')" rows="2" />
              </FormControl>
              <FormDescription>{{
                $t('admin.inbox.livechat.greetingMessage.variables')
              }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="config.chat_introduction">
            <FormItem>
              <FormLabel>{{ $t('admin.inbox.livechat.chatIntroduction') }}</FormLabel>
              <FormControl>
                <Textarea
                  v-bind="componentField"
                  :placeholder="$t('placeholders.chatIntroduction')"
                  rows="2"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <!-- Notice Banner -->
          <div class="space-y-4">
            <h4 class="text-base font-semibold text-foreground">
              {{ $t('admin.inbox.livechat.noticeBanner') }}
            </h4>

            <FormField
              v-slot="{ componentField, handleChange }"
              name="config.notice_banner.enabled"
            >
              <FormItem>
                <SwitchField
                  :title="$t('admin.inbox.livechat.noticeBanner.enabled')"
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
                <FormLabel>{{ $t('admin.inbox.livechat.noticeBanner.text') }}</FormLabel>
                <FormControl>
                  <Textarea
                    v-bind="componentField"
                    :placeholder="$t('placeholders.noticeBannerText')"
                    rows="2"
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>
        </div>

        <!-- Features Tab -->
        <div v-show="activeTab === 'features'" class="space-y-8">
          <!-- Office Hours -->
          <div class="space-y-4">
            <h4 class="text-base font-semibold text-foreground">
              {{ $t('admin.inbox.livechat.officeHours') }}
            </h4>

            <FormField
              v-slot="{ componentField, handleChange }"
              name="config.show_office_hours_in_chat"
            >
              <FormItem>
                <SwitchField
                  :title="$t('admin.inbox.livechat.showOfficeHoursInChat')"
                  :description="$t('admin.inbox.livechat.showOfficeHoursInChat.description')"
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
                  :title="$t('admin.inbox.livechat.showOfficeHoursAfterAssignment')"
                  :description="$t('admin.inbox.livechat.showOfficeHoursAfterAssignment.description')"
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
                <FormLabel>{{ $t('admin.inbox.livechat.chatReplyExpectationMessage') }}</FormLabel>
                <FormControl>
                  <Input type="text" v-bind="componentField" />
                </FormControl>
                <FormDescription>
                  {{ $t('admin.inbox.livechat.chatReplyExpectationMessage.description') }}
                </FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>

          <!-- Chat Features -->
          <div class="space-y-4">
            <h4 class="text-base font-semibold text-foreground">{{ $t('globals.terms.features') }}</h4>

            <div class="space-y-3">
              <FormField
                v-slot="{ componentField, handleChange }"
                name="config.features.file_upload"
              >
                <FormItem>
                  <SwitchField
                    :title="$t('admin.inbox.livechat.features.fileUpload')"
                    :description="$t('admin.inbox.livechat.features.fileUpload.description')"
                    :checked="componentField.modelValue"
                    @update:checked="handleChange"
                  />
                </FormItem>
              </FormField>

              <FormField v-slot="{ componentField, handleChange }" name="config.features.emoji">
                <FormItem>
                  <SwitchField
                    :title="$t('admin.inbox.livechat.features.emoji')"
                    :description="$t('admin.inbox.livechat.features.emoji.description')"
                    :checked="componentField.modelValue"
                    @update:checked="handleChange"
                  />
                </FormItem>
              </FormField>

              <FormField
                v-slot="{ componentField, handleChange }"
                name="config.direct_to_conversation"
              >
                <FormItem>
                  <SwitchField
                    :title="$t('admin.inbox.livechat.directToConversation')"
                    :description="$t('admin.inbox.livechat.directToConversation.description')"
                    :checked="componentField.modelValue"
                    @update:checked="handleChange"
                  />
                </FormItem>
              </FormField>
            </div>
          </div>
        </div>

        <!-- Security Tab -->
        <div v-show="activeTab === 'security'" class="space-y-8">
          <div class="grid grid-cols-2 gap-6">
            <FormField v-slot="{ componentField }" name="secret">
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.secretKey') }}</FormLabel>
                <FormControl>
                  <Input type="password" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{
                  $t('admin.inbox.livechat.secretKey.description')
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="config.session_duration">
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.sessionDuration.label') }}</FormLabel>
                <FormControl>
                  <Input type="text" placeholder="10h" v-bind="componentField" />
                </FormControl>
                <FormDescription>{{ $t('admin.inbox.livechat.sessionDuration.description') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>

          <div class="grid grid-cols-2 gap-6">
            <FormField v-slot="{ componentField }" name="config.trusted_domains">
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.trustedDomains.list') }}</FormLabel>
                <FormControl>
                  <Textarea
                    v-bind="componentField"
                    placeholder="example.com&#10;*.example.com&#10;another-domain.com"
                    rows="4"
                  />
                </FormControl>
                <FormDescription>{{
                  $t('admin.inbox.livechat.trustedDomains.description')
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField }" name="config.blocked_ips">
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.blockedIPs.list') }}</FormLabel>
                <FormControl>
                  <Textarea
                    v-bind="componentField"
                    placeholder="192.168.1.0/24&#10;10.0.0.1&#10;2001:db8::/32"
                    rows="4"
                  />
                </FormControl>
                <FormDescription>{{
                  $t('admin.inbox.livechat.blockedIPs.description')
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>
          </div>
        </div>

        <!-- Pre-Chat Form Tab -->
        <div v-show="activeTab === 'prechat'" class="space-y-8">
          <PreChatFormConfig v-model="prechatConfig" />
        </div>

        <!-- Users Tab -->
        <div v-show="activeTab === 'users'" class="space-y-8">
          <Tabs :model-value="selectedUserTab" @update:model-value="selectedUserTab = $event">
            <TabsList class="grid w-full grid-cols-2">
              <TabsTrigger value="visitors">
                {{ $t('admin.inbox.livechat.userSettings.visitors') }}
              </TabsTrigger>
              <TabsTrigger value="users">
                {{ $t('globals.terms.users') }}
              </TabsTrigger>
            </TabsList>

            <div class="space-y-4 mt-4">
              <!-- Visitors Settings -->
              <div v-show="selectedUserTab === 'visitors'" class="space-y-4">
                <FormField
                  v-slot="{ componentField }"
                  name="config.visitors.start_conversation_button_text"
                >
                  <FormItem>
                    <FormLabel>{{
                      $t('admin.inbox.livechat.startConversationButtonText')
                    }}</FormLabel>
                    <FormControl>
                      <Input v-bind="componentField" :placeholder="$t('placeholders.startConversation')" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <FormField
                  v-slot="{ componentField, handleChange }"
                  name="config.visitors.allow_start_conversation"
                >
                  <FormItem>
                    <SwitchField
                      :title="$t('admin.inbox.livechat.allowStartConversation')"
                      :description="$t('admin.inbox.livechat.allowStartConversation.visitors.description')"
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
                      :title="$t('admin.inbox.livechat.preventMultipleConversations')"
                      :description="$t('admin.inbox.livechat.preventMultipleConversations.visitors.description')"
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
                      :title="$t('admin.inbox.livechat.preventReplyToClosedConversation')"
                      :description="$t('admin.inbox.livechat.preventReplyToClosedConversation.description')"
                      :checked="componentField.modelValue"
                      @update:checked="handleChange"
                    />
                  </FormItem>
                </FormField>
              </div>

              <!-- Users Settings -->
              <div v-show="selectedUserTab === 'users'" class="space-y-4">
                <FormField
                  v-slot="{ componentField }"
                  name="config.users.start_conversation_button_text"
                >
                  <FormItem>
                    <FormLabel>{{
                      $t('admin.inbox.livechat.startConversationButtonText')
                    }}</FormLabel>
                    <FormControl>
                      <Input v-bind="componentField" :placeholder="$t('placeholders.startConversation')" />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>

                <FormField
                  v-slot="{ componentField, handleChange }"
                  name="config.users.allow_start_conversation"
                >
                  <FormItem>
                    <SwitchField
                      :title="$t('admin.inbox.livechat.allowStartConversation')"
                      :description="$t('admin.inbox.livechat.allowStartConversation.users.description')"
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
                      :title="$t('admin.inbox.livechat.preventMultipleConversations')"
                      :description="$t('admin.inbox.livechat.preventMultipleConversations.users.description')"
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
                      :title="$t('admin.inbox.livechat.preventReplyToClosedConversation')"
                      :description="$t('admin.inbox.livechat.preventReplyToClosedConversation.description')"
                      :checked="componentField.modelValue"
                      @update:checked="handleChange"
                    />
                  </FormItem>
                </FormField>
              </div>
            </div>
          </Tabs>
        </div>

        <!-- Installation Tab -->
        <div v-show="activeTab === 'installation'" class="space-y-8">
          <div class="space-y-4">
            <h4 class="text-base font-semibold text-foreground">
              {{ $t('admin.inbox.livechat.installation.instructions.title') }}
            </h4>
            <ol class="text-sm space-y-2 list-decimal list-inside text-muted-foreground">
              <li>{{ $t('admin.inbox.livechat.installation.instructions.step1') }}</li>
              <li>{{ $t('admin.inbox.livechat.installation.instructions.step2') }}</li>
            </ol>
          </div>

          <!-- Basic Installation -->
          <div class="relative">
            <CodeEditor :modelValue="integrationSnippet" language="html" :readOnly="true" />
            <CopyButton :text="integrationSnippet" class="absolute top-3 right-3" />
          </div>

          <!-- Identity Verification Section -->
          <div class="space-y-4 pt-4">
            <h4 class="text-base font-semibold text-foreground">
              {{ $t('admin.inbox.livechat.installation.identity.title') }}
            </h4>

            <div class="space-y-1">
              <p class="text-sm text-muted-foreground">
                {{ $t('admin.inbox.livechat.installation.identity.description') }}
              </p>
              <p class="text-sm text-muted-foreground">
                {{ $t('admin.inbox.livechat.installation.identity.howItWorks') }}
              </p>
            </div>

            <div class="relative">
              <CodeEditor :modelValue="jwtPayloadExample" language="javascript" :readOnly="true" />
              <CopyButton :text="jwtPayloadExample" class="absolute top-3 right-3" />
            </div>

            <p class="text-sm text-muted-foreground">
              {{ $t('admin.inbox.livechat.installation.identity.addJwt') }}
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
              {{ $t('admin.inbox.livechat.installation.identity.secretWarning') }}
            </p>
          </div>

          <!-- JavaScript API Section -->
          <div class="space-y-4 pt-4">
            <h4 class="text-base font-semibold text-foreground">
              {{ $t('admin.inbox.livechat.installation.jsApi.title') }}
            </h4>

            <p class="text-sm text-muted-foreground">
              {{ $t('admin.inbox.livechat.installation.jsApi.description') }}
            </p>

            <div class="relative">
              <CodeEditor :modelValue="jsApiSnippet" language="javascript" :readOnly="true" />
              <CopyButton :text="jsApiSnippet" class="absolute top-3 right-3" />
            </div>
          </div>
        </div>
      </div>
    </Tabs>

    <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
      {{ submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import { watch, computed, ref, onMounted } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './livechatFormSchema.js'
import { useInboxStore } from '@/stores/inbox'
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
import { RadioGroup, RadioGroupItem } from '@shared-ui/components/ui/radio-group'
import { Label } from '@shared-ui/components/ui/label'
import { Plus, X, TriangleAlert, GripVertical, Lightbulb } from 'lucide-vue-next'
import Draggable from 'vuedraggable'
import { useI18n } from 'vue-i18n'
import PreChatFormConfig, { getDefaultPrechatFields } from './PreChatFormConfig.vue'
import { useAppSettingsStore } from '@/stores/appSettings'
import CopyButton from '@/components/button/CopyButton.vue'
import CodeEditor from '@/components/editor/CodeEditor.vue'

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

const { t } = useI18n()
const activeTab = ref('general')
const selectedUserTab = ref('visitors')
const homeApps = ref([])
const prechatConfig = ref({
  enabled: false,
  title: '',
  fields: getDefaultPrechatFields()
})

const inboxStore = useInboxStore()
const appSettingsStore = useAppSettingsStore()

const emailInboxes = computed(() =>
  inboxStore.inboxes.filter((inbox) => inbox.channel === 'email' && inbox.enabled)
)

// Get base URL from app settings
const baseUrl = computed(() => {
  return appSettingsStore.settings?.['app.root_url'] || window.location.origin
})

// Generate integration snippet
const integrationSnippet = computed(() => {
  const inboxUUID = props.initialValues?.uuid || '<INBOX_UUID>'
  return `<script>
  window.LibredeskSettings = {
    baseURL: '${baseUrl.value}',
    inboxID: '${inboxUUID}'
  };
<\/script>
<script async src="${baseUrl.value}/widget.js"><\/script>`
})

// JWT payload example
const jwtPayloadExample = computed(() => {
  return `{
  "external_user_id": "your_app_user_123",    // Required: Your system's unique user ID
  "email": "user@example.com",                // Required: User's email
  "first_name": "John",                       // Required: User's first name
  "last_name": "Doe",                         // Optional: User's last name
  "exp": 1735689600,                          // Required: Expiration time (Unix timestamp in seconds)
  "contact_custom_attributes": {              // Optional: Contact-level attributes
    "plan": "premium",
    "company": "Acme Inc"
  }
}`
})

// Authenticated integration snippet
const authenticatedIntegrationSnippet = computed(() => {
  const inboxUUID = props.initialValues?.uuid || '<INBOX_UUID>'
  return `<script>
  window.LibredeskSettings = {
    baseURL: '${baseUrl.value}',
    inboxID: '${inboxUUID}',
    userJWT: 'YOUR_SIGNED_JWT_TOKEN_HERE' // Generated by your server
  };
<\/script>
<script async src="${baseUrl.value}/widget.js"><\/script>`
})

// JavaScript API example
const jsApiSnippet = computed(() => {
  const inboxUUID = props.initialValues?.uuid || '<INBOX_UUID>'
  return `window.Libredesk.show();
window.Libredesk.hide();
window.Libredesk.toggle();

window.Libredesk.setUser('SIGNED_JWT_TOKEN');
window.Libredesk.logout();

window.Libredesk.onShow(function() {});
window.Libredesk.onHide(function() {});
window.Libredesk.onUnreadCountChange(function(count) {});`
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
  initialValues: {
    name: '',
    enabled: true,
    secret: '',
    csat_enabled: false,
    prompt_tags_on_reply: false,
    linked_email_inbox_id: null,
    config: {
      brand_name: '',
      website_url: '',
      dark_mode: false,
      show_powered_by: true,
      language: 'en',
      fallback_language: 'en',
      logo_url: '',
      launcher: {
        position: 'right',
        logo_url: '',
        color: '#000000',
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
      colors: {
        primary: '#2563eb'
      },
      home_screen: {
        header_text_color: 'white',
        background: {
          type: 'solid',
          color: '',
          gradient_start: '#2563eb',
          gradient_end: '#1e40af',
          image_url: ''
        },
        fade_background: false
      },
      features: {
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
        prevent_reply_to_closed_conversation: false
      },
      users: {
        start_conversation_button_text: 'Start conversation',
        allow_start_conversation: true,
        prevent_multiple_conversations: false,
        prevent_reply_to_closed_conversation: false
      },
      prechat_form: {
        enabled: false,
        title: '',
        fields: getDefaultPrechatFields()
      }
    }
  }
})

const submitLabel = computed(() => {
  return props.submitLabel || (props.isNewForm ? t('globals.messages.create') : t('globals.messages.save'))
})

const addHomeApp = (type) => {
  if (type === 'announcement') {
    homeApps.value.push({ type: 'announcement', title: '', description: '', image_url: '', url: '' })
  } else {
    homeApps.value.push({ type: 'external_link', text: '', url: '' })
  }
  updateHomeApps()
}

const removeHomeApp = (index) => {
  homeApps.value.splice(index, 1)
  updateHomeApps()
}

const updateHomeApps = () => {
  form.setFieldValue('config.home_apps', homeApps.value)
}

// Fetch inboxes and app settings on mount
onMounted(() => {
  inboxStore.fetchInboxes()
  appSettingsStore.fetchPublicConfig()
})

const onSubmit = form.handleSubmit(async (values) => {
  // Transform trusted_domains from textarea to array
  if (values.config.trusted_domains) {
    values.config.trusted_domains = values.config.trusted_domains
      .split('\n')
      .map((domain) => domain.trim())
      .filter((domain) => domain)
  } else {
    values.config.trusted_domains = []
  }

  // Transform blocked_ips from textarea to array
  if (values.config.blocked_ips) {
    values.config.blocked_ips = values.config.blocked_ips
      .split('\n')
      .map((ip) => ip.trim())
      .filter((ip) => ip)
  } else {
    values.config.blocked_ips = []
  }

  // Filter out incomplete home apps before submission
  if (values.config.home_apps) {
    values.config.home_apps = values.config.home_apps.filter((item) => {
      if (item.type === 'announcement') return item.title && item.url && item.image_url
      if (item.type === 'external_link') return item.text && item.url
      return true
    })
  }

  // if no linked email inbox, nullify and clear continuity config
  if (!values.linked_email_inbox_id) {
    values.linked_email_inbox_id = null
    values.config.continuity = {}
  }

  // Sync prechat config to form values before submission.
  // If no fields are enabled, disable the form.
  const pc = { ...prechatConfig.value }
  if (pc.enabled && pc.fields?.length > 0 && !pc.fields.some((f) => f.enabled)) {
    pc.enabled = false
  }
  values.config.prechat_form = pc

  await props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }

    // Transform trusted_domains array back to textarea format
    if (newValues.config?.trusted_domains && Array.isArray(newValues.config.trusted_domains)) {
      newValues.config.trusted_domains = newValues.config.trusted_domains.join('\n')
    }

    // Transform blocked_ips array back to textarea format
    if (newValues.config?.blocked_ips && Array.isArray(newValues.config.blocked_ips)) {
      newValues.config.blocked_ips = newValues.config.blocked_ips.join('\n')
    }

    // Set home apps for the reactive array
    if (newValues.config?.home_apps) {
      homeApps.value = [...newValues.config.home_apps]
    }

    // Set prechat config
    if (newValues.config?.prechat_form) {
      const pc = JSON.parse(JSON.stringify(newValues.config.prechat_form))
      if (!pc.fields || pc.fields.length === 0) {
        pc.fields = getDefaultPrechatFields()
      }
      prechatConfig.value = pc
    }

    form.setValues(newValues)
  },
  { deep: true, immediate: true }
)
</script>
