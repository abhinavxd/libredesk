<template>
  <form ref="formEl" @submit="onSubmit" novalidate class="w-full space-y-6">
    <div
      v-if="initialValues?.token_invalid"
      class="box border-destructive/40 bg-destructive/5 p-3 text-sm flex items-start gap-2"
    >
      <TriangleAlert class="size-4 mt-0.5 text-destructive shrink-0" />
      <span>{{ i18n._('Authentication failed. This inbox can\'\'t send or receive messages until you update its credentials.') }}</span>
    </div>
    <div v-show="!showFormFields" class="space-y-4">
      <div class="space-y-2">
        <h3 class="ui-heading">{{ i18n._('Choose setup method') }}</h3>
        <p class="ui-description">
          {{ i18n._('Select how you want to connect your email account') }}
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
            {{ i18n._(provider.label) }}
          </span>
          <span class="text-sm font-normal text-muted-foreground">{{
            i18n._(provider.description)
          }}</span>
        </Button>
      </div>
    </div>
    <div v-show="showFormFields" class="space-y-6">
      <div v-show="isOAuthInbox" class="flex flex-wrap items-start gap-3 rounded-md border p-4">
        <CheckCircle2 class="size-5 shrink-0 text-success" aria-hidden="true" />
        <div class="min-w-0 flex-1">
          <p class="font-semibold text-foreground">
            {{ i18n._('Connected via OAuth - {provider}', { provider: oauthProvider }) }}
          </p>
          <p class="break-all ui-description">{{ oauthEmail }}</p>
          <p v-show="oauthClientId" class="mt-1 break-all font-mono text-xs text-muted-foreground">
            {{ i18n._('Client ID') }}: {{ oauthClientId.substring(0, 20) }}...{{
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
          {{ i18n._('Reconnect') }}
        </Button>
      </div>

      <section class="space-y-4">
        <div class="grid gap-4 sm:grid-cols-2">
          <FormField v-slot="{ componentField, handleChange, meta }" name="name">
            <FormItem>
              <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
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
              <FormLabel>{{ i18n.ngettext('From email address', 'From email addresses', 1) }}</FormLabel>
              <FormControl>
                <Input
                  type="text"
                  :placeholder="i18n._('My inbox \u003csupport@example.com\u003e')"
                  :name="componentField.name"
                  :model-value="componentField.modelValue"
                  @update:model-value="(v) => handleChange(v, meta.validated)"
                />
              </FormControl>
              <FormDescription>
                {{ i18n._('From email address for your inbox. e.g. My inbox \u003csupport@example.com\u003e') }}
              </FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>
        <FormField v-slot="{ componentField, handleChange, meta }" name="enabled">
          <FormItem>
            <SwitchField
              :title="i18n._('Enabled')"
              :description="i18n._('Toggle scanning inbox and sending out messages.')"
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
            <h3 id="inbox-aliases-heading" class="ui-heading">
              {{ i18n.ngettext('Email alias', 'Email aliases', 2) }}
            </h3>
            <Button type="button" variant="outline" size="sm" class="shrink-0" @click="addAlias">
              <Plus class="size-4" aria-hidden="true" />
              {{ i18n._('Add') }}
            </Button>
          </div>
          <p class="ui-description">{{ i18n._('Aliases must forward to this inbox\'\'s mailbox, and your SMTP provider must allow sending as each alias.') }}</p>
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
                  >{{ i18n.ngettext('Email', 'Emails', 1) }} {{ index + 1 }}</FormLabel
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
                      {{ i18n._(aliasStatus(field.value).label) }}
                    </span>
                    <span v-else class="text-xs text-muted-foreground" role="status">
                      {{ i18n._('Save the inbox first to verify this alias.') }}
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
                    {{ i18n._(aliasStatus(field.value).action) }}
                  </Button>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="col-start-2 row-start-1 text-muted-foreground hover:text-destructive sm:col-start-3"
                  :aria-label="i18n._('Remove')"
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
            {{ i18n._('IMAP Configuration') }}
          </AccordionTrigger>
          <AccordionContent force-mount class="space-y-4 pb-6 pt-1">
            <div class="grid grid-cols-2 items-start gap-4 sm:grid-cols-6">
              <FormField v-slot="{ componentField, handleChange, meta }" name="imap.host">
                <FormItem class="col-span-2 sm:col-span-3">
                  <FormLabel>{{ i18n.ngettext('Host', 'Hosts', 1) }}</FormLabel>
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
                  <FormLabel>{{ i18n.ngettext('Port', 'Ports', 1) }}</FormLabel>
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
                  <FormLabel>{{ i18n.ngettext('TLS', 'TLSs', 1) }}</FormLabel>
                  <FormControl>
                    <Select
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    >
                      <SelectTrigger type="button">
                        <SelectValue :placeholder="i18n._('Select TLS type')" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">{{ i18n._('OFF') }}</SelectItem>
                        <SelectItem value="tls">SSL/TLS</SelectItem>
                        <SelectItem value="starttls">STARTTLS</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormDescription>{{ i18n._('Choose the encryption method for IMAP.') }}</FormDescription>
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
                  <FormLabel>{{ i18n.ngettext('Username', 'Usernames', 1) }}</FormLabel>
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
                  <FormLabel>{{ i18n.ngettext('Password', 'Passwords', 1) }}</FormLabel>
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
                  i18n._('Advanced')
                }}</AccordionTrigger>
                <AccordionContent force-mount class="space-y-4 pt-5">
                  <FormGrid compact>
                    <FormField v-slot="{ componentField, handleChange, meta }" name="imap.mailbox">
                      <FormItem>
                        <FormLabel>{{ i18n._('Mailbox') }}</FormLabel>
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
                          {{ i18n._('Mailbox (folder) to scan for incoming emails. Default is INBOX (usually no need to change).') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="imap.read_interval"
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Scan Interval') }}</FormLabel>
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
                          {{ i18n._('Interval to scan the inbox for new emails. Format: 120s, 1m, 1h') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="imap.scan_inbox_since"
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Scan Inbox Since') }}</FormLabel>
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
                          {{ i18n._('To improve performance in large helpdesks with high email volume, this limits scans to emails received since the specified duration (e.g., `2h`, `48h`) by subtracting it from the current time.') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </FormGrid>

                  <FormField
                    v-slot="{ componentField, handleChange, meta }"
                    name="imap.tls_skip_verify"
                  >
                    <FormItem v-show="!isOAuthInbox || form.errors.value['imap.tls_skip_verify']">
                      <SwitchField
                        :title="i18n._('Skip TLS Verification')"
                        :description="i18n._('Skip hostname check on the TLS certificate.')"
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
            {{ i18n._('SMTP Configuration') }}
          </AccordionTrigger>
          <AccordionContent force-mount class="space-y-4 pb-6 pt-1">
            <div class="grid grid-cols-2 items-start gap-4 sm:grid-cols-6">
              <FormField v-slot="{ componentField, handleChange, meta }" name="smtp.host">
                <FormItem class="col-span-2 sm:col-span-3">
                  <FormLabel>{{ i18n.ngettext('Host', 'Hosts', 1) }}</FormLabel>
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
                  <FormLabel>{{ i18n.ngettext('Port', 'Ports', 1) }}</FormLabel>
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
                  <FormLabel>{{ i18n.ngettext('TLS', 'TLSs', 1) }}</FormLabel>
                  <FormControl>
                    <Select
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    >
                      <SelectTrigger type="button">
                        <SelectValue :placeholder="i18n._('Select TLS type')" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">{{ i18n._('OFF') }}</SelectItem>
                        <SelectItem value="tls">SSL/TLS</SelectItem>
                        <SelectItem value="starttls">STARTTLS</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormControl>
                  <FormDescription> {{ i18n._('TLS/SSL encryption, STARTTLS is commonly used.') }} </FormDescription>
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
                  <FormLabel>{{ i18n.ngettext('Username', 'Usernames', 1) }}</FormLabel>
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
                  <FormLabel>{{ i18n.ngettext('Password', 'Passwords', 1) }}</FormLabel>
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
                  i18n._('Advanced')
                }}</AccordionTrigger>
                <AccordionContent force-mount class="space-y-4 pt-5">
                  <FormGrid compact>
                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.max_conns"
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Max Connections') }}</FormLabel>
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
                          {{ i18n._('Maximum number of concurrent connections to the server.') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.max_msg_retries"
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Max Retries') }}</FormLabel>
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
                          >{{ i18n._('Number of times to retry when a message fails.') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.idle_timeout"
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Idle Timeout') }}</FormLabel>
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
                          {{ i18n._('Maximum time an inactive connection must be kept alive before closing it and removing it from the pool.') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.pool_wait_timeout"
                    >
                      <FormItem>
                        <FormLabel>{{ i18n._('Wait Timeout') }}</FormLabel>
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
                          {{ i18n._('Maximum time to wait to obtain a connection before timing out. Timeouts may occur when all open connections are busy sending e-mails and they\'\'re not returning to the pool fast enough. This is also the timeout used when creating new SMTP connections.') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>
                  </FormGrid>

                  <div class="grid gap-4 sm:grid-cols-2">
                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.auth_protocol"
                    >
                      <FormItem v-show="!isOAuthInbox || form.errors.value['smtp.auth_protocol']">
                        <FormLabel>{{ i18n._('Authentication Protocol') }}</FormLabel>
                        <FormControl>
                          <Select
                            :name="componentField.name"
                            :model-value="componentField.modelValue"
                            @update:model-value="(v) => handleChange(v, meta.validated)"
                          >
                            <SelectTrigger type="button">
                              <SelectValue :placeholder="i18n._('Select protocol')" />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="login">{{
                                i18n._('Login')
                              }}</SelectItem>
                              <SelectItem value="cram">CRAM</SelectItem>
                              <SelectItem value="plain">{{
                                i18n._('Plain')
                              }}</SelectItem>
                              <SelectItem value="none">{{ i18n.pgettext('No authentication', 'None') }}</SelectItem>
                            </SelectContent>
                          </Select>
                        </FormControl>
                        <FormDescription>
                          {{ i18n._('Authentication protocol to use.') }}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <FormField
                      v-slot="{ componentField, handleChange, meta }"
                      name="smtp.hello_hostname"
                    >
                      <FormItem v-show="!isOAuthInbox || form.errors.value['smtp.hello_hostname']">
                        <FormLabel>{{ i18n._('HELO Hostname') }}</FormLabel>
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
                          {{ i18n._('The hostname to use in the HELO/EHLO command. If not set, defaults to localhost.') }}
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
                        :title="i18n._('Skip TLS Verification')"
                        :description="i18n._('Skip hostname check on the TLS certificate.')"
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
            {{ i18n._('Options') }}
          </AccordionTrigger>
          <AccordionContent force-mount class="space-y-4 pb-6 pt-1">
            <div class="grid gap-4 sm:grid-cols-2">
              <FormField v-slot="{ componentField, handleChange, meta }" name="from_name_template">
                <FormItem>
                  <div class="flex items-center gap-1">
                    <FormLabel>{{ i18n._('From display name template (optional)') }}</FormLabel>
                    <HelpPopover :text="fromNameVariablesHelp" />
                  </div>
                  <FormControl>
                    <Input
                      type="text"
                      :placeholder="i18n._(literalExampleMessage2)"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormDescription>
                    {{ i18n._('Display name for outgoing email replies. Leave blank to use the name from the From address.') }}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
              <FormField v-slot="{ componentField, handleChange, meta }" name="reply_to">
                <FormItem>
                  <FormLabel>{{ i18n._('Reply-To address (optional)') }}</FormLabel>
                  <FormControl>
                    <Input
                      type="text"
                      :placeholder="i18n._('support@example.com')"
                      :name="componentField.name"
                      :model-value="componentField.modelValue"
                      @update:model-value="(v) => handleChange(v, meta.validated)"
                    />
                  </FormControl>
                  <FormDescription>
                    {{ i18n._('Where customer replies go. Only needed when SMTP From and IMAP mailbox are on different domains.') }}
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
                  :title="i18n._('Enable plus addressing')"
                  :description="i18n._('Improves conversation threading but requires provider support (e.g., Gmail, Microsoft 365).')"
                  :checked="componentField.modelValue"
                  :disabled="isMicrosoftInbox && componentField.modelValue"
                  @update:checked="(v) => handleChange(v, meta.validated)"
                />
                <p
                  v-show="isMicrosoftInbox"
                  :class="AUTH_HINT_CLASS"
                >
                  <Lightbulb class="size-4 shrink-0" aria-hidden="true" />
                  <span>{{ i18n._('Required for Microsoft inboxes to thread replies correctly.') }}</span>
                </p>
              </FormItem>
            </FormField>
            <FormField v-slot="{ componentField, handleChange, meta }" name="csat_enabled">
              <FormItem>
                <SwitchField
                  :title="i18n._('CSAT Surveys')"
                  :description="i18n._('Send customer satisfaction surveys when a conversation is marked as resolved.')"
                  :checked="componentField.modelValue"
                  @update:checked="(v) => handleChange(v, meta.validated)"
                />
              </FormItem>
              <p :class="AUTH_HINT_CLASS">
                <Lightbulb class="size-4 shrink-0" aria-hidden="true" />
                <span
                  >{{ i18n._('For better control on when to send surveys, disable this option and create an automation rule to send surveys.') }}
                  {{ i18n._('CSAT surveys are only sent once per conversation.') }}</span
                >
              </p>
            </FormField>
            <FormField v-slot="{ componentField, handleChange, meta }" name="prompt_tags_on_reply">
              <FormItem>
                <SwitchField
                  :title="i18n._('Prompt to tag before replying')"
                  :description="i18n._('Warn agents before they send a reply on a conversation that has no tags. Agents can still send without tagging by confirming the prompt.')"
                  :checked="componentField.modelValue"
                  @update:checked="(v) => handleChange(v, meta.validated)"
                />
              </FormItem>
            </FormField>
          </AccordionContent>
        </AccordionItem>
      </Accordion>

      <div class="border-t border-border pt-4">
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
              ? i18n._('Reconnect {provider} account', {
                  provider:
                    selectedProvider === PROVIDER_GOOGLE
                      ? i18n._('Google')
                      : i18n._('Microsoft')
                })
              : i18n._('Connect {provider} account', {
                  provider:
                    selectedProvider === PROVIDER_GOOGLE
                      ? i18n._('Google')
                      : i18n._('Microsoft')
                })
          }}
        </DialogTitle>
        <DialogDescription>
          {{
            flowType === 'reconnect'
              ? i18n._('Re-enter your credentials to refresh the OAuth connection')
              : i18n._('Follow the steps below to connect your email account')
          }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <ol v-show="flowType === 'new_inbox'" class="space-y-3 border-l-2 pl-4 text-sm" role="list">
          <li>
            {{ i18n._('1. Create OAuth app at') }}
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
                  ? i18n._('Google Cloud Console')
                  : i18n._('Microsoft Azure Portal')
              }}
            </a>
          </li>

          <li class="space-y-1.5">
            <p>{{ i18n._('2. Add this callback URL:') }}</p>
            <div class="flex items-center gap-2">
              <Input
                :aria-label="i18n._('2. Add this callback URL:')"
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
                {{ i18n._('Copy') }}
              </Button>
            </div>
          </li>

          <li>{{ i18n._('3. Enter your credentials below:') }}</li>
        </ol>

        <div class="space-y-2">
          <Label for="oauth-client_id">{{ i18n._('Client ID') }}</Label>
          <Input
            id="oauth-client_id"
            v-model="oauthCredentials.client_id"
            :placeholder="i18n._('Enter your OAuth client ID')"
            :disabled="isSubmittingOAuth"
          />
        </div>

        <div class="space-y-2">
          <Label for="oauth-client_secret">{{ i18n._('Client secret') }}</Label>
          <Input
            id="oauth-client_secret"
            v-model="oauthCredentials.client_secret"
            type="password"
            :placeholder="i18n._('Enter your OAuth client secret')"
            :disabled="isSubmittingOAuth"
          />
        </div>

        <div v-show="selectedProvider === PROVIDER_MICROSOFT" class="space-y-2">
          <Label for="oauth-tenant_id">{{ i18n._('Tenant ID') }}</Label>
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
          {{ i18n._('Cancel') }}
        </Button>
        <Button type="button" @click="submitOAuthCredentials" :disabled="isSubmittingOAuth">
          {{ isSubmittingOAuth ? i18n._('Connecting...') : i18n._('Continue') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { defineMessage as msg } from '@lingui/core/macro'

const literalExampleMessage2 = msg({ message: '\'{{ .Agent.FirstName }} at {{ .Inbox.Name }}\'' })


const AUTH_HINT_CLASS = '!mt-2 flex items-start gap-1.5 text-xs text-muted-foreground'

import FormGrid from '@shared-ui/components/FormGrid.vue'
import { watch, computed, nextTick, ref } from 'vue'
import { useFieldArray, useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema, FROM_NAME_TEMPLATE_VARS } from './formSchema.js'
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
import HelpPopover from '@main/components/HelpPopover.vue'
import { formatTemplateVariables } from '@shared-ui/utils/string'
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
import { useI18n } from '@shared-ui/i18n'
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
    label: msg({ message: 'Not verified' }),
    action: msg({ message: 'Verify' })
  },
  pending: {
    icon: Clock,
    label: msg({ message: 'Pending' }),
    action: msg({ message: 'Resend' })
  },
  verified: {
    class: 'text-success',
    icon: CheckCircle2,
    label: msg({ message: 'Verified' }),
    action: msg({ message: 'Verify' })
  },
  failed: {
    class: 'text-destructive',
    icon: XCircle,
    label: msg({ message: 'Failed' }),
    action: msg({ message: 'Try again' })
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

const i18n = useI18n()
const fromNameVariablesHelp = computed(
  () => i18n._('Variables: {variables}', { variables: formatTemplateVariables(FROM_NAME_TEMPLATE_VARS) })
)
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
  validationSchema: computed(() => toTypedSchema(createFormSchema(i18n))),
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
    (props.isNewForm ? i18n._('Create') : i18n._('Save'))
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
    label: msg({ message: 'Google' }),
    description: msg({ message: 'Connect with Google Workspace or Gmail' }),
    icon: '/images/google-logo.svg'
  },
  {
    value: PROVIDER_MICROSOFT,
    label: msg({ message: 'Microsoft' }),
    description: msg({ message: 'Connect with Microsoft 365 or Outlook' }),
    icon: '/images/microsoft-logo.svg'
  },
  {
    value: 'manual',
    label: msg({ message: 'Other provider' }),
    description: msg({ message: 'Configure IMAP and SMTP manually' })
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
      description: i18n._('Please provide both client ID and client secret')
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
      description: i18n._('Copied to clipboard')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: i18n._('Something went wrong')
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
