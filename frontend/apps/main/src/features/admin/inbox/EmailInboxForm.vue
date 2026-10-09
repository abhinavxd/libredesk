<template>
  <form ref="formEl" @submit="onSubmit" novalidate class="w-full space-y-6">
    <div
      v-if="initialValues?.token_invalid"
      class="box border-destructive/40 bg-destructive/5 p-3 text-sm flex items-start gap-2"
    >
      <TriangleAlert class="size-4 mt-0.5 text-destructive shrink-0" />
      <span>{{ $t('admin.inbox.authenticationFailed') }}</span>
    </div>
    <div v-show="!showFormFields" class="space-y-4">
      <div class="space-y-2">
        <h3 class="text-lg font-semibold">{{ $t('admin.inbox.oauth.chooseSetupMethod') }}</h3>
        <p class="text-sm text-muted-foreground">
          {{ $t('admin.inbox.oauth.selectConnectionMethod') }}
        </p>
      </div>
      <div class="grid gap-3 sm:grid-cols-3">
        <Button
          v-for="provider in setupProviders"
          :key="provider.value"
          type="button"
          variant="outline"
          class="h-auto flex-col items-start gap-2 whitespace-normal p-4 text-left"
          @click="selectSetupMethod(provider.value)"
        >
          <span class="flex items-center gap-2 font-semibold">
            <img v-if="provider.icon" :src="provider.icon" alt="" class="size-5" />
            <Mail v-else class="size-5" aria-hidden="true" />
            {{ $t(provider.label) }}
          </span>
          <span class="text-sm font-normal text-muted-foreground">{{
            $t(provider.description)
          }}</span>
        </Button>
      </div>
    </div>
    <div v-show="showFormFields" class="space-y-6">
      <div v-show="isOAuthInbox" class="flex flex-wrap items-start gap-3 rounded-md border p-4">
        <CheckCircle2 class="size-5 shrink-0 text-success" aria-hidden="true" />
        <div class="min-w-0 flex-1">
          <p class="font-semibold text-foreground">
            {{ $t('admin.inbox.oauth.connectedVia', { provider: oauthProvider }) }}
          </p>
          <p class="break-all text-sm text-muted-foreground">{{ oauthEmail }}</p>
          <p v-show="oauthClientId" class="mt-1 break-all font-mono text-xs text-muted-foreground">
            {{ $t('globals.terms.clientID') }}: {{ oauthClientId.substring(0, 20) }}...{{
              oauthClientId.slice(-8)
            }}
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="shrink-0"
          :disabled="isSubmittingOAuth"
          @click="reconnectOAuth"
        >
          <RefreshCw class="size-4" aria-hidden="true" />
          {{ $t('globals.terms.reconnect') }}
        </Button>
      </div>

      <section class="space-y-4">
        <div class="grid gap-4 sm:grid-cols-2">
          <FormField v-slot="{ componentField, handleChange, meta }" name="name">
            <FormItem>
              <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
              <FormControl>
                <Input
                  type="text"
                  placeholder=""
                  :name="componentField.name"
                  :model-value="componentField.modelValue"
                  @update:model-value="(v) => handleChange(v, meta.validated)"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
          <FormField v-slot="{ componentField, handleChange, meta }" name="from">
            <FormItem>
              <FormLabel>{{ $t('globals.terms.fromEmailAddress') }}</FormLabel>
              <FormControl>
                <Input
                  type="text"
                  :placeholder="t('admin.inbox.fromEmailAddress.placeholder')"
                  :name="componentField.name"
                  :model-value="componentField.modelValue"
                  @update:model-value="(v) => handleChange(v, meta.validated)"
                />
              </FormControl>
              <FormDescription>
                {{ $t('admin.inbox.fromEmailAddress.description') }}
              </FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>
        <FormField v-slot="{ componentField, handleChange, meta }" name="enabled">
          <FormItem>
            <SwitchField
              :title="$t('globals.terms.enabled')"
              :description="$t('admin.inbox.enabled.description')"
              :checked="componentField.modelValue"
              @update:checked="(v) => handleChange(v, meta.validated)"
            />
          </FormItem>
        </FormField>
      </section>

      <section
        class="space-y-4 border-t pt-6"
        data-section="aliases"
        aria-labelledby="inbox-aliases-heading"
      >
        <div class="space-y-1.5">
          <div class="flex items-center justify-between gap-4">
            <h3 id="inbox-aliases-heading" class="text-sm font-semibold">
              {{ $t('globals.terms.emailAlias', 2) }}
            </h3>
            <Button type="button" variant="outline" size="sm" class="shrink-0" @click="addAlias">
              <Plus class="size-4" aria-hidden="true" />
              {{ $t('globals.messages.add') }}
            </Button>
          </div>
          <p class="text-sm text-muted-foreground">{{ $t('admin.inbox.aliases.description') }}</p>
        </div>
        <ul v-show="aliasFields.length" class="divide-y" role="list">
          <li
            v-for="(field, index) in aliasFields"
            :key="field.key"
            class="py-3 first:pt-0 last:pb-0"
            data-alias-row
          >
            <FormField
              v-slot="{ componentField, handleChange, meta }"
              :name="`aliases[${index}].email`"
            >
              <FormItem
                class="grid grid-cols-[minmax(0,1fr)_2.25rem] items-center gap-x-3 gap-y-2 space-y-0 sm:grid-cols-[minmax(0,1fr)_7rem_2.25rem]"
              >
                <FormLabel class="sr-only"
                  >{{ $t('globals.terms.email') }} {{ index + 1 }}</FormLabel
                >
                <FormControl>
                  <Input
                    type="email"
                    placeholder="billing@example.com"
                    :name="componentField.name"
                    :model-value="componentField.modelValue"
                    @update:model-value="(v) => handleChange(v, meta.validated)"
                    class="col-start-1 row-start-1 min-w-0"
                  />
                </FormControl>
                <div
                  class="col-span-full row-start-2 flex items-center justify-between gap-3 sm:contents"
                >
                  <div class="sm:col-start-1 sm:row-start-2">
                    <span
                      v-if="savedAliasStatus(field.value)"
                      class="flex items-center gap-1.5 text-xs"
                      :class="aliasStatus(field.value).class || 'text-muted-foreground'"
                      role="status"
                    >
                      <component
                        :is="aliasStatus(field.value).icon"
                        class="size-3.5 shrink-0"
                        aria-hidden="true"
                      />
                      {{ $t(aliasStatus(field.value).label) }}
                    </span>
                    <span v-else class="text-xs text-muted-foreground" role="status">
                      {{ $t('admin.inbox.aliases.saveBeforeVerify') }}
                    </span>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    class="min-h-9 shrink-0 sm:col-start-2 sm:row-start-1"
                    :disabled="!verifyAlias || !savedAliasStatus(field.value)"
                    @click="handleVerifyAlias(field.value)"
                  >
                    {{ $t(aliasStatus(field.value).action) }}
                  </Button>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="col-start-2 row-start-1 text-muted-foreground hover:text-destructive sm:col-start-3"
                  :aria-label="$t('globals.terms.remove')"
                  @click="removeAlias(index)"
                >
                  <Trash2 class="size-4" aria-hidden="true" />
                </Button>
                <FormMessage class="col-span-full" />
              </FormItem>
            </FormField>
          </li>
        </ul>
      </section>

      <Accordion type="multiple" v-model="openSections" class="border-t">
        <AccordionItem value="imap" data-section="imap" data-connection="imap">
          <AccordionTrigger type="button" class="hover:no-underline">
            {{ $t('admin.inbox.imapConfig') }}
          </AccordionTrigger>
          <AccordionContent force-mount class="space-y-4 pb-6 pt-1">
            <div class="grid grid-cols-2 items-start gap-4 sm:grid-cols-6">
              <FormField v-slot="{ componentField, handleChange, meta }" name="imap.host">
                <FormItem class="col-span-2 sm:col-span-3">
                  <FormLabel>{{ $t('globals.terms.host') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      placeholder="imap.gmail.com"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField, handleChange, meta }" name="imap.port">
                <FormItem>
                  <FormLabel>{{ $t('globals.terms.port') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      placeholder="993"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField, handleChange, meta }" name="imap.tls_type">
                <FormItem class="sm:col-span-2">
                  <FormLabel>{{ $t('globals.terms.tls') }}</FormLabel>
                  <FormControl>
                    <Select
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    >
                      <SelectTrigger type="button">
                        <SelectValue :placeholder="t('globals.messages.selectTLS')" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">{{ $t('globals.terms.off') }}</SelectItem>
                        <SelectItem value="tls">SSL/TLS</SelectItem>
                        <SelectItem value="starttls">STARTTLS</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormDescription>{{ $t('admin.inbox.imap.tls.description') }}</FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <div
              v-show="
                !isOAuthInbox ||
                form.errors.value['imap.username'] ||
                form.errors.value['imap.password']
              "
              class="grid gap-4 sm:grid-cols-2"
            >
              <FormField v-slot="{ componentField, handleChange, meta }" name="imap.username">
                <FormItem>
                  <FormLabel>{{ $t('globals.terms.username') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      placeholder="inbox@example.com"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField, handleChange, meta }" name="imap.password">
                <FormItem>
                  <FormLabel>{{ $t('globals.terms.password') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="password"
                      placeholder="••••••••"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <Accordion type="single" collapsible v-model="imapAdvanced">
              <AccordionItem value="advanced" class="border-b-0">
                <AccordionTrigger type="button" :class="ADVANCED_TRIGGER_CLASS">{{
                  $t('globals.terms.advanced')
                }}</AccordionTrigger>
                <AccordionContent force-mount class="space-y-4 pt-5">
                  <div class="grid gap-4 sm:grid-cols-2">
                    <FormField v-slot="{ componentField, handleChange, meta }" name="imap.mailbox">
                      <FormItem>
                        <FormLabel>{{ $t('admin.inbox.mailbox') }}</FormLabel>
                        <FormControl>
                          <Input
                            type="text"
                            placeholder="INBOX"
                            :name="componentField.name"
                            :model-value="componentField.modelValue"
                            @update:model-value="(v) => handleChange(v, meta.validated)"
                          />
                        </FormControl>
                        <FormDescription>
                          {{ $t('admin.inbox.mailbox.description') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="imap.read_interval"
                    >
                      <FormItem>
                        <FormLabel>{{ $t('admin.inbox.imapScanInterval') }}</FormLabel>
                        <FormControl>
                          <Input
                            type="text"
                            placeholder="5m"
                            :name="componentField.name"
                            :model-value="componentField.modelValue"
                            @update:model-value="(v) => handleChange(v, meta.validated)"
                          />
                        </FormControl>
                        <FormDescription>
                          {{ $t('admin.inbox.imapScanInterval.description') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="imap.scan_inbox_since"
                    >
                      <FormItem>
                        <FormLabel>{{ $t('admin.inbox.imapScanInboxSince') }}</FormLabel>
                        <FormControl>
                          <Input
                            type="text"
                            placeholder="48h"
                            :name="componentField.name"
                            :model-value="componentField.modelValue"
                            @update:model-value="(v) => handleChange(v, meta.validated)"
                          />
                        </FormControl>
                        <FormDescription>
                          {{ $t('admin.inbox.imapScanInboxSince.description') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </div>

                  <FormField
                    v-slot="{ componentField, handleChange, meta }"
                    name="imap.tls_skip_verify"
                  >
                    <FormItem v-show="!isOAuthInbox || form.errors.value['imap.tls_skip_verify']">
                      <SwitchField
                        :title="$t('admin.inbox.skipTLSVerification')"
                        :description="$t('admin.inbox.skipTLSVerification.description')"
                        :checked="componentField.modelValue"
                        @update:checked="(v) => handleChange(v, meta.validated)"
                      />
                    </FormItem>
                  </FormField>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </AccordionContent>
        </AccordionItem>
        <AccordionItem value="smtp" data-section="smtp" data-connection="smtp">
          <AccordionTrigger type="button" class="hover:no-underline">
            {{ $t('admin.inbox.smtpConfig') }}
          </AccordionTrigger>
          <AccordionContent force-mount class="space-y-4 pb-6 pt-1">
            <div class="grid grid-cols-2 items-start gap-4 sm:grid-cols-6">
              <FormField v-slot="{ componentField, handleChange, meta }" name="smtp.host">
                <FormItem class="col-span-2 sm:col-span-3">
                  <FormLabel>{{ $t('globals.terms.host') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      placeholder="smtp.gmail.com"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField, handleChange, meta }" name="smtp.port">
                <FormItem>
                  <FormLabel>{{ $t('globals.terms.port') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      placeholder="587"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField, handleChange, meta }" name="smtp.tls_type">
                <FormItem class="sm:col-span-2">
                  <FormLabel>{{ t('globals.terms.tls') }}</FormLabel>
                  <FormControl>
                    <Select
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    >
                      <SelectTrigger type="button">
                        <SelectValue :placeholder="t('globals.messages.selectTLS')" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">{{ $t('globals.terms.off') }}</SelectItem>
                        <SelectItem value="tls">SSL/TLS</SelectItem>
                        <SelectItem value="starttls">STARTTLS</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormDescription> {{ $t('admin.inbox.tls.description') }} </FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <div
              v-show="
                !isOAuthInbox ||
                form.errors.value['smtp.username'] ||
                form.errors.value['smtp.password']
              "
              class="grid gap-4 sm:grid-cols-2"
            >
              <FormField v-slot="{ componentField, handleChange, meta }" name="smtp.username">
                <FormItem>
                  <FormLabel>{{ $t('globals.terms.username') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      placeholder="user@example.com"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField, handleChange, meta }" name="smtp.password">
                <FormItem>
                  <FormLabel>{{ $t('globals.terms.password') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="password"
                      placeholder="••••••••"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>

            <Accordion type="single" collapsible v-model="smtpAdvanced">
              <AccordionItem value="advanced" class="border-b-0">
                <AccordionTrigger type="button" :class="ADVANCED_TRIGGER_CLASS">{{
                  $t('globals.terms.advanced')
                }}</AccordionTrigger>
                <AccordionContent force-mount class="space-y-4 pt-5">
                  <div class="grid gap-4 sm:grid-cols-2">
                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.max_conns"
                    >
                      <FormItem>
                        <FormLabel>{{ $t('admin.inbox.maxConnections') }}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            placeholder="10"
                            :name="componentField.name"
                            :model-value="componentField.modelValue"
                            @update:model-value="(v) => handleChange(v, meta.validated)"
                          />
                        </FormControl>
                        <FormDescription>
                          {{ $t('admin.inbox.maxConnections.description') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.max_msg_retries"
                    >
                      <FormItem>
                        <FormLabel>{{ $t('admin.inbox.maxRetries') }}</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            placeholder="3"
                            :name="componentField.name"
                            :model-value="componentField.modelValue"
                            @update:model-value="(v) => handleChange(v, meta.validated)"
                          />
                        </FormControl>
                        <FormDescription
                          >{{ $t('admin.inbox.maxRetries.description') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </div>

                  <div class="grid gap-4 sm:grid-cols-2">
                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.idle_timeout"
                    >
                      <FormItem>
                        <FormLabel>{{ $t('admin.inbox.idleTimeout') }}</FormLabel>
                        <FormControl>
                          <Input
                            type="text"
                            placeholder="25s"
                            :name="componentField.name"
                            :model-value="componentField.modelValue"
                            @update:model-value="(v) => handleChange(v, meta.validated)"
                          />
                        </FormControl>
                        <FormDescription>
                          {{ $t('admin.inbox.idleTimeout.description') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.pool_wait_timeout"
                    >
                      <FormItem>
                        <FormLabel>{{ $t('admin.inbox.waitTimeout') }}</FormLabel>
                        <FormControl>
                          <Input
                            type="text"
                            placeholder="60s"
                            :name="componentField.name"
                            :model-value="componentField.modelValue"
                            @update:model-value="(v) => handleChange(v, meta.validated)"
                          />
                        </FormControl>
                        <FormDescription>
                          {{ $t('admin.inbox.waitTimeout.description') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </div>

                  <div class="grid gap-4 sm:grid-cols-2">
                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.auth_protocol"
                    >
                      <FormItem v-show="!isOAuthInbox || form.errors.value['smtp.auth_protocol']">
                        <FormLabel>{{ $t('admin.inbox.authProtocol') }}</FormLabel>
                        <FormControl>
                          <Select
                            :name="componentField.name"
                            :model-value="componentField.modelValue"
                            @update:model-value="(v) => handleChange(v, meta.validated)"
                          >
                            <SelectTrigger type="button">
                              <SelectValue :placeholder="t('placeholders.selectProtocol')" />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="login">{{
                                $t('admin.inbox.authProtocol.login')
                              }}</SelectItem>
                              <SelectItem value="cram">CRAM</SelectItem>
                              <SelectItem value="plain">{{
                                $t('admin.inbox.authProtocol.plain')
                              }}</SelectItem>
                              <SelectItem value="none">{{ $t('globals.terms.none') }}</SelectItem>
                            </SelectContent>
                          </Select>
                        </FormControl>
                        <FormDescription>
                          {{ $t('admin.inbox.authProtocol.description') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.hello_hostname"
                    >
                      <FormItem v-show="!isOAuthInbox || form.errors.value['smtp.hello_hostname']">
                        <FormLabel>{{ $t('admin.inbox.heloHostname') }}</FormLabel>
                        <FormControl>
                          <Input
                            type="text"
                            placeholder=""
                            :name="componentField.name"
                            :model-value="componentField.modelValue"
                            @update:model-value="(v) => handleChange(v, meta.validated)"
                          />
                        </FormControl>
                        <FormDescription>
                          {{ $t('admin.inbox.heloHostname.description') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </div>

                  <FormField
                    v-slot="{ componentField, handleChange, meta }"
                    name="smtp.tls_skip_verify"
                  >
                    <FormItem v-show="!isOAuthInbox || form.errors.value['smtp.tls_skip_verify']">
                      <SwitchField
                        :title="$t('admin.inbox.skipTLSVerification')"
                        :description="$t('admin.inbox.skipTLSVerification.description')"
                        :checked="componentField.modelValue"
                        @update:checked="(v) => handleChange(v, meta.validated)"
                      />
                    </FormItem>
                  </FormField>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </AccordionContent>
        </AccordionItem>
        <AccordionItem value="options" data-section="options" class="border-b-0">
          <AccordionTrigger type="button" class="hover:no-underline">
            {{ $t('globals.messages.options') }}
          </AccordionTrigger>
          <AccordionContent force-mount class="space-y-4 pb-6 pt-1">
            <div class="grid gap-4 sm:grid-cols-2">
              <FormField v-slot="{ componentField, handleChange, meta }" name="from_name_template">
                <FormItem>
                  <FormLabel>{{ $t('admin.inbox.fromNameTemplate') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      :placeholder="t('admin.inbox.fromNameTemplate.placeholder')"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormDescription>
                    {{ $t('admin.inbox.fromNameTemplate.description') }}
                    <br />
                    {{ $t('admin.inbox.fromNameTemplate.variables') }}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField, handleChange, meta }" name="reply_to">
                <FormItem>
                  <FormLabel>{{ $t('admin.inbox.replyToAddress') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      :placeholder="t('admin.inbox.replyToAddress.placeholder')"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormDescription>
                    {{ $t('admin.inbox.replyToAddress.description') }}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
            <FormField
              v-slot="{ componentField, handleChange, meta }"
              name="enable_plus_addressing"
            >
              <FormItem>
                <SwitchField
                  :title="$t('admin.inbox.enablePlusAddressing')"
                  :description="$t('admin.inbox.enablePlusAddressing.description')"
                  :checked="componentField.modelValue"
                  :disabled="isMicrosoftInbox && componentField.modelValue"
                  @update:checked="(v) => handleChange(v, meta.validated)"
                />
                <p
                  v-show="isMicrosoftInbox"
                  class="!mt-2 flex items-start gap-1.5 text-xs text-muted-foreground"
                >
                  <Lightbulb class="size-4 shrink-0" aria-hidden="true" />
                  <span>{{ $t('admin.inbox.enablePlusAddressing.requiredForMicrosoft') }}</span>
                </p>
              </FormItem>
            </FormField>
            <FormField v-slot="{ componentField, handleChange, meta }" name="csat_enabled">
              <FormItem>
                <SwitchField
                  :title="$t('admin.inbox.csatSurveys')"
                  :description="$t('admin.inbox.csatSurveys.description_1')"
                  :checked="componentField.modelValue"
                  @update:checked="(v) => handleChange(v, meta.validated)"
                />
              </FormItem>
              <p class="!mt-2 flex items-start gap-1.5 text-xs text-muted-foreground">
                <Lightbulb class="size-4 shrink-0" aria-hidden="true" />
                <span
                  >{{ $t('admin.inbox.csatSurveys.description_2') }}
                  {{ $t('admin.inbox.csatSurveys.description_3') }}</span
                >
              </p>
            </FormField>
            <FormField v-slot="{ componentField, handleChange, meta }" name="prompt_tags_on_reply">
              <FormItem>
                <SwitchField
                  :title="$t('admin.inbox.promptTagsOnReply')"
                  :description="$t('admin.inbox.promptTagsOnReply.description')"
                  :checked="componentField.modelValue"
                  @update:checked="(v) => handleChange(v, meta.validated)"
                />
              </FormItem>
            </FormField>
          </AccordionContent>
        </AccordionItem>
      </Accordion>

      <div class="sticky bottom-0 z-10 !mt-0 border-t bg-background py-4">
        <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
          {{ submitLabel }}
        </Button>
      </div>
    </div>
  </form>

  <Dialog v-model:open="showOAuthModal">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>
          {{
            flowType === 'reconnect'
              ? $t('admin.inbox.oauth.reconnectAccount', {
                  provider:
                    selectedProvider === PROVIDER_GOOGLE
                      ? $t('globals.terms.google')
                      : $t('globals.terms.microsoft')
                })
              : $t('admin.inbox.oauth.connectAccount', {
                  provider:
                    selectedProvider === PROVIDER_GOOGLE
                      ? $t('globals.terms.google')
                      : $t('globals.terms.microsoft')
                })
          }}
        </DialogTitle>
        <DialogDescription>
          {{
            flowType === 'reconnect'
              ? $t('admin.inbox.oauth.reconnectDescription')
              : $t('admin.inbox.oauth.followSteps')
          }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <ol v-show="flowType === 'new_inbox'" class="space-y-3 border-l-2 pl-4 text-sm" role="list">
          <li>
            {{ $t('admin.inbox.oauth.step1CreateApp') }}
            <a
              :href="
                selectedProvider === PROVIDER_GOOGLE
                  ? 'https://console.cloud.google.com/apis/credentials'
                  : 'https://entra.microsoft.com/'
              "
              target="_blank"
              rel="noopener noreferrer"
              class="link-style"
            >
              {{
                selectedProvider === PROVIDER_GOOGLE
                  ? $t('admin.inbox.oauth.googleCloudConsole')
                  : $t('admin.inbox.oauth.microsoftAzurePortal')
              }}
            </a>
          </li>

          <li class="space-y-1.5">
            <p>{{ $t('admin.inbox.oauth.step2AddCallback') }}</p>
            <div class="flex items-center gap-2">
              <Input
                :aria-label="$t('admin.inbox.oauth.step2AddCallback')"
                :model-value="callbackUrl"
                readonly
                class="font-mono text-xs"
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                @click="copyToClipboard(callbackUrl)"
              >
                {{ $t('globals.terms.copy') }}
              </Button>
            </div>
          </li>

          <li>{{ $t('admin.inbox.oauth.step3EnterCredentials') }}</li>
        </ol>

        <div class="space-y-2">
          <Label for="oauth-client_id">{{ $t('globals.terms.clientID') }}</Label>
          <Input
            id="oauth-client_id"
            v-model="oauthCredentials.client_id"
            :placeholder="t('admin.inbox.oauth.enterClientID')"
            :disabled="isSubmittingOAuth"
          />
        </div>

        <div class="space-y-2">
          <Label for="oauth-client_secret">{{ $t('globals.terms.clientSecret') }}</Label>
          <Input
            id="oauth-client_secret"
            v-model="oauthCredentials.client_secret"
            type="password"
            :placeholder="t('admin.inbox.oauth.enterClientSecret')"
            :disabled="isSubmittingOAuth"
          />
        </div>

        <div v-show="selectedProvider === PROVIDER_MICROSOFT" class="space-y-2">
          <Label for="oauth-tenant_id">{{ $t('globals.terms.tenantID') }}</Label>
          <Input
            id="oauth-tenant_id"
            v-model="oauthCredentials.tenant_id"
            :disabled="isSubmittingOAuth"
          />
        </div>
      </div>

      <DialogFooter>
        <Button
          type="button"
          variant="outline"
          @click="showOAuthModal = false"
          :disabled="isSubmittingOAuth"
        >
          {{ $t('globals.messages.cancel') }}
        </Button>
        <Button type="button" @click="submitOAuthCredentials" :disabled="isSubmittingOAuth">
          {{ isSubmittingOAuth ? $t('globals.messages.connecting') : $t('globals.terms.continue') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { watch, computed, nextTick, ref } from 'vue'
import { useFieldArray, useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import { Label } from '@shared-ui/components/ui/label'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger
} from '@shared-ui/components/ui/accordion'
import {
  CheckCircle2,
  Circle,
  Clock,
  RefreshCw,
  XCircle,
  Mail,
  Lightbulb,
  Plus,
  Trash2,
  TriangleAlert
} from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import {
  AUTH_TYPE_PASSWORD,
  AUTH_TYPE_OAUTH2,
  PROVIDER_GOOGLE,
  PROVIDER_MICROSOFT
} from '@/constants/auth.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useAppSettingsStore } from '@/stores/appSettings'

const ALIAS_STATUS = {
  not_verified: {
    icon: Circle,
    label: 'globals.terms.notVerified',
    action: 'globals.messages.verify'
  },
  pending: {
    icon: Clock,
    label: 'globals.terms.pending',
    action: 'globals.messages.resend'
  },
  verified: {
    class: 'text-success',
    icon: CheckCircle2,
    label: 'globals.terms.verified',
    action: 'globals.messages.verify'
  },
  failed: {
    class: 'text-destructive',
    icon: XCircle,
    label: 'globals.terms.failed',
    action: 'globals.terms.tryAgain'
  }
}
const ADVANCED_TRIGGER_CLASS =
  'flex-none justify-start gap-1 py-1 font-normal text-muted-foreground hover:text-foreground hover:no-underline'
const IMAP_ADVANCED_FIELDS = [
  'imap.mailbox',
  'imap.read_interval',
  'imap.scan_inbox_since',
  'imap.tls_skip_verify'
]
const SMTP_ADVANCED_FIELDS = [
  'smtp.max_conns',
  'smtp.max_msg_retries',
  'smtp.idle_timeout',
  'smtp.pool_wait_timeout',
  'smtp.auth_protocol',
  'smtp.hello_hostname',
  'smtp.tls_skip_verify'
]

const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
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
  },
  verifyAlias: {
    type: Function,
    default: null
  },
  aliasVerificationState: {
    type: Object,
    default: () => ({})
  }
})

const { t } = useI18n()
const emitter = useEmitter()
const appSettingsStore = useAppSettingsStore()

const isOAuthInbox = ref(false)
const formEl = ref(null)
const openSections = ref(props.isNewForm ? ['imap', 'smtp'] : [])

const setupMethod = ref(null)

const showOAuthModal = ref(false)
const selectedProvider = ref('')
const flowType = ref('new_inbox')
const oauthCredentials = ref({
  client_id: '',
  client_secret: '',
  tenant_id: ''
})
const isSubmittingOAuth = ref(false)

const callbackUrl = computed(() => {
  const rootUrl = appSettingsStore.settings['app.root_url']
  return `${rootUrl}/api/v1/inboxes/oauth/${selectedProvider.value}/callback`
})

const showFormFields = computed(
  () =>
    isOAuthInbox.value ||
    setupMethod.value === 'manual' ||
    (props.initialValues?.imap && Object.keys(props.initialValues?.imap).length > 0)
)

const form = useForm({
  validationSchema: computed(() => toTypedSchema(createFormSchema(t))),
  initialValues: {
    name: '',
    from: '',
    aliases: [],
    from_name_template: '',
    reply_to: '',
    enabled: true,
    csat_enabled: false,
    prompt_tags_on_reply: false,
    enable_plus_addressing: true,
    auth_type: AUTH_TYPE_PASSWORD,
    imap: {
      host: 'imap.gmail.com',
      port: 993,
      mailbox: 'INBOX',
      username: '',
      password: '',
      tls_type: 'none',
      read_interval: '5m',
      scan_inbox_since: '48h',
      tls_skip_verify: false
    },
    smtp: {
      host: 'smtp.gmail.com',
      port: 587,
      username: '',
      password: '',
      max_conns: 10,
      max_msg_retries: 3,
      idle_timeout: '25s',
      pool_wait_timeout: '120s',
      auth_protocol: 'login',
      tls_type: 'none',
      hello_hostname: '',
      tls_skip_verify: false
    }
  }
})

const { fields: aliasFields, push: pushAlias, remove: removeAlias } = useFieldArray('aliases')

const addAlias = async () => {
  pushAlias({ email: '' })
  await nextTick()
  formEl.value?.querySelector('[data-alias-row]:last-child input')?.focus()
}

const savedAliasStatus = (alias) =>
  props.aliasVerificationState[alias?.email?.trim().toLowerCase()]?.verification_status
const aliasStatus = (alias) => ALIAS_STATUS[savedAliasStatus(alias)] || ALIAS_STATUS.not_verified

const handleVerifyAlias = (alias) => {
  if (!alias?.email || !props.verifyAlias) return
  props.verifyAlias(alias.email)
}

const oauthProvider = computed(() => {
  const provider = form.values.oauth?.provider
  return provider ? provider.charAt(0).toUpperCase() + provider.slice(1) : 'Unknown'
})

const oauthEmail = computed(() => {
  return form.values.imap?.username || form.values.smtp?.username || ''
})

const oauthClientId = computed(() => {
  return form.values.oauth?.client_id || ''
})

const isMicrosoftInbox = computed(() => form.values.oauth?.provider === PROVIDER_MICROSOFT)

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.isNewForm ? t('globals.messages.create') : t('globals.messages.save'))
  )
})

const imapAdvanced = ref('')
const smtpAdvanced = ref('')

const getFieldSection = (key) => {
  if (key.startsWith('imap.')) return 'imap'
  if (key.startsWith('smtp.')) return 'smtp'
  if (
    [
      'from_name_template',
      'reply_to',
      'enable_plus_addressing',
      'csat_enabled',
      'prompt_tags_on_reply'
    ].includes(key)
  )
    return 'options'
  return null
}

const onSubmit = form.handleSubmit(
  async (values) => {
    await props.submitForm(values)
  },
  async ({ errors }) => {
    const keys = Object.keys(errors)
    const invalidSections = keys.map(getFieldSection).filter(Boolean)
    openSections.value = [...new Set([...openSections.value, ...invalidSections])]
    if (keys.some((key) => IMAP_ADVANCED_FIELDS.includes(key))) imapAdvanced.value = 'advanced'
    if (keys.some((key) => SMTP_ADVANCED_FIELDS.includes(key))) smtpAdvanced.value = 'advanced'
    await nextTick()
    const invalidField = [...(formEl.value?.querySelectorAll('[aria-invalid="true"]') || [])].find(
      (field) => field.getClientRects().length
    )
    invalidField?.focus()
    invalidField?.scrollIntoView({ block: 'center' })
  }
)

const setupProviders = [
  {
    value: PROVIDER_GOOGLE,
    label: 'globals.terms.google',
    description: 'admin.inbox.oauth.googleDescription',
    icon: '/images/google-logo.svg'
  },
  {
    value: PROVIDER_MICROSOFT,
    label: 'globals.terms.microsoft',
    description: 'admin.inbox.oauth.microsoftDescription',
    icon: '/images/microsoft-logo.svg'
  },
  {
    value: 'manual',
    label: 'admin.inbox.oauth.otherProvider',
    description: 'admin.inbox.oauth.otherProviderDescription'
  }
]

const selectSetupMethod = (method) => {
  if (method === 'manual') {
    setupMethod.value = method
    return
  }
  flowType.value = 'new_inbox'
  selectedProvider.value = method
  showOAuthModal.value = true
}

const reconnectOAuth = () => {
  const provider = form.values.oauth?.provider
  const clientId = form.values.oauth?.client_id
  const tenantId = form.values.oauth?.tenant_id

  if (!provider) return

  flowType.value = 'reconnect'

  selectedProvider.value = provider
  oauthCredentials.value.client_id = clientId || ''
  oauthCredentials.value.client_secret = ''
  oauthCredentials.value.tenant_id = tenantId || ''

  showOAuthModal.value = true
}

const submitOAuthCredentials = async () => {
  if (!oauthCredentials.value.client_id || !oauthCredentials.value.client_secret) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('admin.inbox.oauth.clientIDSecretRequired')
    })
    return
  }

  try {
    isSubmittingOAuth.value = true
    const payload = {
      ...oauthCredentials.value,
      flow_type: flowType.value
    }

    if (flowType.value === 'reconnect' && props.initialValues?.id) {
      payload.inbox_id = props.initialValues.id
    }

    const response = await api.initiateOAuthFlow(selectedProvider.value, payload)
    window.location.href = response.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.copied')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.somethingWentWrong')
    })
  }
}

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }
    if (newValues.config?.auth_type === AUTH_TYPE_OAUTH2) {
      isOAuthInbox.value = true
      setupMethod.value = 'oauth'
    } else {
      isOAuthInbox.value = false
      setupMethod.value = 'manual'
    }
    form.resetForm({ values: newValues })
  },
  { deep: true, immediate: true }
)
</script>
