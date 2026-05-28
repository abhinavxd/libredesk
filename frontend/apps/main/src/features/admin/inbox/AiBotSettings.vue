<template>
  <div class="space-y-6">
    <div class="flex items-center gap-2">
      <span class="text-lg">🤖</span>
      <h3 class="text-base font-semibold">AI Chatbot</h3>
    </div>

    <FormField v-slot="{ componentField, handleChange }" name="config.ai_bot.enabled">
      <FormItem>
        <SwitchField
          title="فعال‌سازی AI"
          description="وقتی فعال باشد، ربات به صورت خودکار به پیام‌های کاربران پاسخ می‌دهد"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <div class="space-y-4">
      <FormField v-slot="{ componentField }" name="config.ai_bot.api_key">
        <FormItem>
          <FormLabel>API Key</FormLabel>
          <FormControl>
            <Input type="password" v-bind="componentField" placeholder="sk-..." />
          </FormControl>
          <FormDescription>کلید API (DeepSeek, OpenAI, یا هر سرویس سازگار)</FormDescription>
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="config.ai_bot.base_url">
        <FormItem>
          <FormLabel>Base URL</FormLabel>
          <FormControl>
            <Input v-bind="componentField" placeholder="https://api.deepseek.com" />
          </FormControl>
          <FormDescription>
            آدرس API سرویس AI. برای DeepSeek: https://api.deepseek.com — برای OpenAI: https://api.openai.com
          </FormDescription>
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="config.ai_bot.model">
        <FormItem>
          <FormLabel>Model</FormLabel>
          <FormControl>
            <Input v-bind="componentField" placeholder="deepseek-chat" />
          </FormControl>
          <FormDescription>
            نام مدل. مثال‌ها: deepseek-chat, gpt-4o, gpt-3.5-turbo, deepseek-v4-flash
          </FormDescription>
        </FormItem>
      </FormField>
    </div>

    <div class="border-t pt-6">
      <h4 class="text-sm font-semibold mb-4">📝 پاسخ‌دهی</h4>
      <div class="space-y-4">
        <FormField v-slot="{ componentField }" name="config.ai_bot.response_length">
          <FormItem>
            <FormLabel>طول پاسخ</FormLabel>
            <FormControl>
              <RadioGroup v-bind="componentField" class="flex gap-4">
                <div class="flex items-center gap-2">
                  <RadioGroupItem value="short" id="rl-short" />
                  <Label for="rl-short" class="cursor-pointer">کوتاه (۲ جمله)</Label>
                </div>
                <div class="flex items-center gap-2">
                  <RadioGroupItem value="medium" id="rl-medium" />
                  <Label for="rl-medium" class="cursor-pointer">متوسط (۴ جمله)</Label>
                </div>
                <div class="flex items-center gap-2">
                  <RadioGroupItem value="long" id="rl-long" />
                  <Label for="rl-long" class="cursor-pointer">بلند (توضیح کامل)</Label>
                </div>
              </RadioGroup>
            </FormControl>
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField, handleChange }" name="config.ai_bot.only_questions">
          <FormItem>
            <SwitchField
              title="فقط سوال بپرسد"
              description="AI فقط سوال می‌پرسد و پاسخ نمی‌دهد"
              :checked="componentField.modelValue"
              @update:checked="handleChange"
            />
          </FormItem>
        </FormField>
      </div>
    </div>

    <div class="border-t pt-6">
      <h4 class="text-sm font-semibold mb-4">🗣️ لحن و سبک</h4>
      <div class="space-y-4">
        <FormField v-slot="{ componentField }" name="config.ai_bot.tone">
          <FormItem>
            <FormLabel>لحن</FormLabel>
            <FormControl>
              <RadioGroup v-bind="componentField" class="grid grid-cols-2 gap-3">
                <div class="flex items-center gap-2">
                  <RadioGroupItem value="friendly" id="tone-friendly" />
                  <Label for="tone-friendly" class="cursor-pointer">😊 صمیمی</Label>
                </div>
                <div class="flex items-center gap-2">
                  <RadioGroupItem value="formal" id="tone-formal" />
                  <Label for="tone-formal" class="cursor-pointer">👔 رسمی</Label>
                </div>
                <div class="flex items-center gap-2">
                  <RadioGroupItem value="casual" id="tone-casual" />
                  <Label for="tone-casual" class="cursor-pointer">🤙 خودمونی</Label>
                </div>
                <div class="flex items-center gap-2">
                  <RadioGroupItem value="professional" id="tone-professional" />
                  <Label for="tone-professional" class="cursor-pointer">💼 حرفه‌ای</Label>
                </div>
              </RadioGroup>
            </FormControl>
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="config.ai_bot.tone_custom">
          <FormItem>
            <FormLabel>توضیحات سفارشی لحن</FormLabel>
            <FormControl>
              <Input v-bind="componentField" placeholder="مثلاً: همیشه با سلام شروع کن، از کلمه عزیز استفاده کن" />
            </FormControl>
          </FormItem>
        </FormField>
      </div>
    </div>

    <div class="border-t pt-6">
      <h4 class="text-sm font-semibold mb-4">📤 خروجی</h4>
      <div class="space-y-3">
        <FormField v-slot="{ componentField, handleChange }" name="config.ai_bot.enable_markdown">
          <FormItem>
            <SwitchField
              title="Markdown"
              description="بولد، بولت، لینک و فرمت‌بندی متن"
              :checked="componentField.modelValue"
              @update:checked="handleChange"
            />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField, handleChange }" name="config.ai_bot.enable_emoji">
          <FormItem>
            <SwitchField
              title="ایموجی"
              description="استفاده از ایموجی در پاسخ‌ها (حداکثر ۳ عدد)"
              :checked="componentField.modelValue"
              @update:checked="handleChange"
            />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField, handleChange }" name="config.ai_bot.enable_links">
          <FormItem>
            <SwitchField
              title="لینک‌های قابل کلیک"
              description="درج لینک با فرمت Markdown [متن](https://...)"
              :checked="componentField.modelValue"
              @update:checked="handleChange"
            />
          </FormItem>
        </FormField>
      </div>
    </div>

    <div class="border-t pt-6">
      <h4 class="text-sm font-semibold mb-4">📚 پایگاه دانش (FAQ)</h4>
      <FormField v-slot="{ componentField }" name="config.ai_bot.faq_data">
        <FormItem>
          <FormControl>
            <Textarea
              v-bind="componentField"
              rows="8"
              placeholder="سوال: قیمت محصول چقدر است؟&#10;پاسخ: قیمت از ۲۵۰ هزار تومان شروع می‌شود.&#10;&#10;سوال: چطور سفارش را پیگیری کنم؟&#10;پاسخ: به بخش پیگیری سفارش در سایت مراجعه کنید."
            />
          </FormControl>
          <FormDescription>
            هر سوال و پاسخ را با «سوال:» و «پاسخ:» شروع کنید. بین هر جفت یک خط خالی بگذارید.
          </FormDescription>
        </FormItem>
      </FormField>
    </div>

    <div class="border-t pt-6">
      <h4 class="text-sm font-semibold mb-4">📋 قوانین سفارشی</h4>
      <FormField v-slot="{ componentField }" name="config.ai_bot.custom_rules">
        <FormItem>
          <FormControl>
            <Textarea
              v-bind="componentField"
              rows="4"
              placeholder="هر خط یک قانون&#10;مثلاً: همیشه شماره پشتیبانی را بده&#10;مثلاً: اشاره به رقبا ممنوع"
            />
          </FormControl>
        </FormItem>
      </FormField>
    </div>

    <div class="border-t pt-6">
      <h4 class="text-sm font-semibold mb-4">🎓 نکات آموزشی</h4>
      <FormField v-slot="{ componentField }" name="config.ai_bot.training_data">
        <FormItem>
          <FormControl>
            <Textarea
              v-bind="componentField"
              rows="4"
              placeholder="اطلاعاتی که AI باید یاد بگیرد و در پاسخ‌هایش اعمال کند.&#10;مثلاً: ما محصولات را با ۲ سال گارانتی می‌فروشیم.&#10;مثلاً: ارسال رایگان برای سفارشات بالای ۵۰۰ هزار تومان."
            />
          </FormControl>
          <FormDescription>این متن به عنوان دانش اضافی به system prompt اضافه می‌شود.</FormDescription>
        </FormItem>
      </FormField>
    </div>
  </div>
</template>

<script setup>
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormDescription,
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import {
  RadioGroup,
  RadioGroupItem,
} from '@shared-ui/components/ui/radio-group'
import { Label } from '@shared-ui/components/ui/label'
</script>
