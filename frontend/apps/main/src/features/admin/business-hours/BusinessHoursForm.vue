<template>
  <form @submit="onSubmit" class="space-y-8">
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
          <div>
            <div
              v-for="day in WEEKDAYS"
              :key="day"
              class="flex items-center justify-between space-y-2"
            >
              <div class="flex items-center space-x-3">
                <Checkbox
                  :id="day"
                  :checked="!!selectedDays[day]"
                  @update:checked="handleDayToggle(day, $event)"
                />
                <Label :for="day" class="font-medium text-gray-800">{{ day }}</Label>
              </div>
              <div class="flex space-x-2 items-center">
                <div class="flex flex-col items-start">
                  <Input
                    type="time"
                    :modelValue="hours[day]?.open || '09:00'"
                    @update:modelValue="(val) => updateHours(day, 'open', val)"
                    :disabled="!selectedDays[day]"
                  />
                </div>
                <span class="text-gray-500">to</span>
                <div class="flex flex-col items-start">
                  <Input
                    type="time"
                    :modelValue="hours[day]?.close || '17:00'"
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
            <Button @click="openHolidayForm = true">
              {{
                t('globals.messages.new', {
                  name: t('globals.terms.holiday')
                })
              }}
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
            {{
              t('globals.messages.new', {
                name: t('globals.terms.holiday')
              })
            }}
          </DialogTitle>
          <DialogDescription />
        </DialogHeader>
        <div class="grid gap-4 py-4">
          <div class="grid grid-cols-4 items-center gap-4">
            <Label for="holiday_name" class="text-right"> {{ t('globals.terms.name') }} </Label>
            <Input id="holiday_name" v-model="holidayName" class="col-span-3" />
          </div>
          <div class="grid grid-cols-4 items-center gap-4">
            <Label for="date" class="text-right"> {{ t('globals.terms.date') }} </Label>
            <Popover>
              <PopoverTrigger as-child>
                <Button
                  variant="outline"
                  :class="
                    cn(
                      'w-[280px] justify-start text-left font-normal',
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
                <Calendar v-model="holidayDate" />
              </PopoverContent>
            </Popover>
          </div>
        </div>
        <DialogFooter>
          <Button :disabled="!holidayName || !holidayDate" @click="saveHoliday">
            {{ t('globals.messages.saveChanges') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
    <Button type="submit" :disabled="isLoading" :isLoading="isLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import { ref, watch, reactive, computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { RadioGroup, RadioGroupItem } from '@shared-ui/components/ui/radio-group/index.js'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@shared-ui/components/ui/form/index.js'
import { Calendar } from '@shared-ui/components/ui/calendar/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover/index.js'
import { cn } from '@shared-ui/lib/utils.js'
import { format } from 'date-fns'
import { WEEKDAYS } from '../../../constants/date.js'
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
  return props.submitLabel || t('globals.messages.save')
})

let holidays = reactive([])
const holidayName = ref('')
const holidayDate = ref(null)
const selectedDays = ref({})
const hours = ref({})
const openHolidayForm = ref(false)
const { t } = useI18n()

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
  initialValues: {
    is_always_open: true
  }
})

// Sync form field with local state
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

// Initialize state from props
const initializeFromValues = (values) => {
  if (!values) return

  // Reset state
  hours.value = {}
  selectedDays.value = {}
  holidays.length = 0

  // Set hours and selected days
  if (values.hours && typeof values.hours === 'object') {
    hours.value = { ...values.hours }
    selectedDays.value = Object.keys(values.hours).reduce((acc, day) => {
      acc[day] = true
      return acc
    }, {})
  }

  // Set holidays
  if (values.holidays) {
    holidays.push(...values.holidays)
  }

  // Update form
  form.setValues(values)
  syncHoursToForm()
}

// Watch for initial values
watch(() => props.initialValues, initializeFromValues, { immediate: true, deep: true })
</script>
