<template>
  <form @submit="onSubmit" class="space-y-8">
    <FormGrid>
      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>
            {{ t('globals.terms.name') }}
          </FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="description">
        <FormItem>
          <FormLabel>
            {{ t('globals.terms.description') }}
          </FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <FormField v-slot="{ componentField }" name="is_always_open">
      <FormItem>
        <FormLabel>
          {{ t('admin.businessHours.setBusinessHours') }}
        </FormLabel>
        <FormControl>
          <RadioGroup v-bind="componentField">
            <div class="flex flex-col space-y-2">
              <div class="flex items-center space-x-3">
                <RadioGroupItem id="r1" :value="true" />
                <Label for="r1">{{ t('admin.businessHours.alwaysOpen24x7') }}</Label>
              </div>
              <div class="flex items-center space-x-3">
                <RadioGroupItem id="r2" :value="false" />
                <Label for="r2">{{ t('admin.businessHours.customBusinessHours') }}</Label>
              </div>
            </div>
          </RadioGroup>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="hours">
      <div v-if="form.values.is_always_open === false">
        <FormItem>
          <div class="space-y-4">
            <div
              v-for="day in WEEKDAYS"
              :key="day"
              class="business-hours-row grid items-center gap-2"
            >
              <div class="flex items-center space-x-3">
                <Checkbox
                  :id="day"
                  :checked="!!selectedDays[day]"
                  @update:checked="handleDayToggle(day, $event)"
                />
                <Label :id="`${day}-label`" :for="day" class="font-medium">{{ day }}</Label>
              </div>
              <div
                class="grid min-w-0 max-w-56 grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2"
              >
                <div class="min-w-0">
                  <span :id="`${day}-from`" class="sr-only">{{ t('globals.terms.from') }}</span>
                  <TimeField
                    class="min-w-0"
                    :aria-labelledby="`${day}-label ${day}-from`"
                    :modelValue="hours[day]?.open ?? '09:00'"
                    @update:modelValue="(val) => updateHours(day, 'open', val)"
                    :disabled="!selectedDays[day]"
                  />
                </div>
                <span :id="`${day}-to`" class="text-muted-foreground">{{
                  t('globals.terms.to')
                }}</span>
                <div class="min-w-0">
                  <TimeField
                    class="min-w-0"
                    :aria-labelledby="`${day}-label ${day}-to`"
                    :modelValue="hours[day]?.close ?? '17:00'"
                    @update:modelValue="(val) => updateHours(day, 'close', val)"
                    :disabled="!selectedDays[day]"
                  />
                </div>
              </div>
            </div>
          </div>
          <FormMessage />
        </FormItem>
      </div>
    </FormField>

    <Dialog :open="openHolidayForm" @update:open="openHolidayForm = false">
      <div>
        <div class="flex justify-between items-center mb-4">
          <div></div>
          <DialogTrigger as-child>
            <Button type="button" @click="openHolidayForm = true">
              {{ t('businessHour.newHoliday') }}
            </Button>
          </DialogTrigger>
        </div>
      </div>
      <SimpleTable
        :headers="[t('globals.terms.name'), t('globals.terms.date')]"
        :keys="['name', 'date']"
        :data="holidays"
        @deleteItem="deleteHoliday"
      />
      <DialogContent class="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {{ t('businessHour.newHoliday') }}
          </DialogTitle>
          <DialogDescription />
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid items-center gap-2 sm:grid-cols-4 sm:gap-4">
            <Label for="holiday_name" class="sm:text-right"> {{ t('globals.terms.name') }} </Label>
            <Input
              id="holiday_name"
              ref="holidayNameRef"
              v-model="holidayName"
              class="min-w-0 sm:col-span-3"
            />
          </div>
          <div class="grid items-center gap-2 sm:grid-cols-4 sm:gap-4">
            <Label for="date" class="sm:text-right"> {{ t('globals.terms.date') }} </Label>
            <Popover :open="datePickerOpen" @update:open="datePickerOpen = $event">
              <PopoverTrigger as-child>
                <Button
                  type="button"
                  id="date"
                  variant="outline"
                  :class="
                    cn(
                      'w-full min-w-0 justify-start text-left font-normal sm:col-span-3',
                      !holidayDate && 'text-muted-foreground'
                    )
                  "
                >
                  <CalendarIcon class="mr-2 h-4 w-4" />
                  {{
                    holidayDate && !isNaN(new Date(holidayDate).getTime())
                      ? format(new Date(holidayDate), 'MMMM dd, yyyy')
                      : t('globals.terms.pickDate')
                  }}
                </Button>
              </PopoverTrigger>
              <PopoverContent class="w-auto p-0">
                <Calendar v-model="holidayDate" @update:model-value="datePickerOpen = false" />
              </PopoverContent>
            </Popover>
          </div>
        </div>
        <DialogFooter>
          <Button type="button" :disabled="!holidayName || !holidayDate" @click="saveHoliday">
            {{ t('globals.messages.add') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
    <Button type="submit" :disabled="isLoading" :isLoading="isLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { ref, watch, reactive, computed, nextTick } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { RadioGroup, RadioGroupItem } from '@shared-ui/components/ui/radio-group/index.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { Calendar } from '@shared-ui/components/ui/calendar/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import TimeField from '@shared-ui/components/TimeField.vue'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover/index.js'
import { cn } from '@shared-ui/lib/utils.js'
import { format } from 'date-fns'
import { WEEKDAYS } from '@main/constants/date.js'
import { Calendar as CalendarIcon } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import SimpleTable from '@main/components/table/SimpleTable.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog/index.js'

const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: () => ''
  },
  isNewForm: {
    type: Boolean
  },
  isLoading: {
    type: Boolean,
    required: false
  }
})

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.isNewForm ? t('globals.messages.create') : t('globals.messages.save'))
  )
})

let holidays = reactive([])
const holidayName = ref('')
const holidayDate = ref(null)
const selectedDays = ref({})
const hours = ref({})
const openHolidayForm = ref(false)
const datePickerOpen = ref(false)
const holidayNameRef = ref(null)
const { t } = useI18n()

watch(openHolidayForm, (isOpen) => {
  if (isOpen) {
    nextTick(() => {
      holidayNameRef.value?.$el?.focus()
    })
  }
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
  initialValues: {
    description: '',
    is_always_open: true
  }
})

const syncHoursToForm = () => {
  form.setFieldValue('hours', { ...hours.value })
}

const saveHoliday = () => {
  holidays.push({
    name: holidayName.value,
    date: new Date(holidayDate.value).toISOString().split('T')[0]
  })
  holidayName.value = ''
  holidayDate.value = null
  openHolidayForm.value = false
}

const deleteHoliday = (item) => {
  holidays.splice(
    holidays.findIndex((h) => h.name === item.name),
    1
  )
}

const handleDayToggle = (day, checked) => {
  selectedDays.value[day] = checked

  if (checked) {
    hours.value[day] = hours.value[day] || { open: '09:00', close: '17:00' }
  } else {
    delete hours.value[day]
  }

  syncHoursToForm()
}

const updateHours = (day, type, value) => {
  if (!hours.value[day]) {
    hours.value[day] = { open: '09:00', close: '17:00' }
  }
  hours.value[day][type] = value
  syncHoursToForm()
}

const onSubmit = form.handleSubmit((values) => {
  const businessHours = values.is_always_open === true ? {} : { ...hours.value }

  const finalValues = {
    ...values,
    hours: businessHours,
    holidays: [...holidays]
  }
  props.submitForm(finalValues)
})

const initializeFromValues = (values) => {
  if (!values) return

  hours.value = {}
  selectedDays.value = {}
  holidays.length = 0

  if (values.hours && typeof values.hours === 'object') {
    // Copy each day: mutating a shared day object retriggers the watcher and resets the form.
    hours.value = Object.fromEntries(
      Object.entries(values.hours).map(([day, hour]) => [day, { ...hour }])
    )
    selectedDays.value = Object.keys(values.hours).reduce((acc, day) => {
      acc[day] = true
      return acc
    }, {})
  }

  if (values.holidays) {
    holidays.push(...values.holidays.map((holiday) => ({ ...holiday })))
  }

  form.setValues(values, false)
  syncHoursToForm()
}

watch(() => props.initialValues, initializeFromValues, { immediate: true, deep: true })
</script>

<style scoped>
@container (min-width: 32rem) {
  .business-hours-row {
    grid-template-columns: minmax(8rem, 1fr) minmax(0, 14rem);
  }
}
</style>
