<template>
  <div class="border-t pt-4">
    <div class="flex items-center justify-between">
      <div>
        <div class="text-sm font-medium leading-none">{{ t('admin.guidedForms.testForm') }}</div>
        <p class="text-sm text-muted-foreground mt-1">{{ t('admin.guidedForms.testFormHint') }}</p>
      </div>
      <Button type="button" variant="outline" size="sm" @click="toggleOpen">
        <FlaskConical class="mr-1 h-4 w-4" />
        {{ open ? t('globals.messages.close') : t('admin.guidedForms.testForm') }}
      </Button>
    </div>

    <div v-if="open" class="mt-3 border rounded-lg p-4 space-y-3">
      <div class="max-h-80 overflow-y-auto space-y-2 pr-1">
        <div
          v-for="(entry, i) in transcript"
          :key="i"
          :class="['flex', entry.role === 'visitor' ? 'justify-end' : 'justify-start']"
        >
          <div
            :class="[
              'max-w-[85%] px-3 py-2 rounded-xl text-sm whitespace-pre-wrap',
              entry.role === 'visitor' ? 'bg-primary text-primary-foreground' : 'bg-muted text-foreground'
            ]"
          >
            {{ entry.text }}
          </div>
        </div>
        <div v-if="ended" class="text-center text-xs text-muted-foreground italic pt-1">
          {{ t('admin.guidedForms.testEnded') }}
        </div>
      </div>

      <!-- Quick-reply buttons for the current choice question, mirroring the widget. -->
      <div v-if="currentOptions.length && !ended" class="flex flex-wrap gap-2">
        <Button
          v-for="option in currentOptions"
          :key="option"
          type="button"
          variant="secondary"
          size="sm"
          @click="submitAnswer(option)"
        >
          {{ option }}
        </Button>
      </div>

      <div class="flex items-center gap-2">
        <Input
          v-model="draft"
          :disabled="ended"
          :placeholder="t('admin.guidedForms.testInputPlaceholder')"
          @keydown.enter.prevent="submitAnswer(draft)"
        />
        <Button type="button" size="sm" :disabled="ended || !draft.trim()" @click="submitAnswer(draft)">
          {{ t('globals.messages.send') }}
        </Button>
        <Button type="button" variant="ghost" size="sm" @click="reset">
          {{ t('admin.guidedForms.testReset') }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { FlaskConical } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'

const props = defineProps({
  steps: { type: Array, required: true }
})

const { t } = useI18n()
const open = ref(false)
const transcript = ref([])
const currentStepId = ref('')
const ended = ref(false)
const draft = ref('')

const toggleOpen = () => {
  open.value = !open.value
  if (open.value && transcript.value.length === 0) reset()
}

const findStep = (id) => props.steps.find((s) => s.id === id)

const currentOptions = computed(() => {
  if (ended.value) return []
  const step = findStep(currentStepId.value)
  return step?.type === 'choice' ? step.options || [] : []
})

const reset = () => {
  transcript.value = []
  ended.value = false
  draft.value = ''
  const first = props.steps[0]
  if (!first) return
  currentStepId.value = first.id
  transcript.value.push({ role: 'bot', text: questionText(first) })
}

// Reset whenever the draft steps change shape meaningfully, so the test never runs against a
// stale copy of a since-edited form.
watch(
  () => props.steps.map((s) => `${s.id}|${s.type}`).join(','),
  () => {
    if (open.value) reset()
  }
)

const questionText = (step) => {
  let text = step.question || ''
  if (step.type === 'choice' && step.options?.length) {
    text += '\n\n' + step.options.join(' / ')
  }
  return text
}

const escapeRegex = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const matchesPattern = (pattern, answer) => {
  try {
    return new RegExp(pattern, 'i').test(answer)
  } catch {
    return false
  }
}

// Mirrors internal/guidedform/handler.go's validAnswerFormat.
const validAnswerFormat = (type, answer) => {
  switch (type) {
    case 'email':
      return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(answer)
    case 'phone':
      return /^[0-9+()\-\s]{6,20}$/.test(answer) && /\d/.test(answer)
    case 'number':
      return answer.trim() !== '' && !Number.isNaN(Number(answer))
    default:
      return true
  }
}

const invalidHint = (type) => {
  switch (type) {
    case 'email':
      return t('admin.guidedForms.invalidEmail')
    case 'phone':
      return t('admin.guidedForms.invalidPhone')
    case 'number':
      return t('admin.guidedForms.invalidNumber')
    default:
      return t('admin.guidedForms.invalidAnswer')
  }
}

// Mirrors internal/guidedform/handler.go's matchBranch + the natural next-in-order fallback.
const nextStepID = (step, answer) => {
  for (const branch of step.branches || []) {
    if (branch.pattern && branch.next_step_id && matchesPattern(branch.pattern, answer)) {
      return branch.next_step_id
    }
  }
  if (step.default_next_step_id) return step.default_next_step_id
  if (!step.ends_form) {
    const index = props.steps.findIndex((s) => s.id === step.id)
    const next = props.steps[index + 1]
    if (next) return next.id
  }
  return ''
}

const submitAnswer = (text) => {
  const answer = (text || '').trim()
  if (!answer || ended.value) return
  const step = findStep(currentStepId.value)
  if (!step) return

  transcript.value.push({ role: 'visitor', text: answer })
  draft.value = ''

  if (step.required && !validAnswerFormat(step.type, answer)) {
    transcript.value.push({ role: 'bot', text: invalidHint(step.type) + '\n\n' + questionText(step) })
    return
  }

  const nextID = nextStepID(step, answer)
  if (!nextID) {
    ended.value = true
    return
  }
  const nextStep = findStep(nextID)
  if (!nextStep) {
    transcript.value.push({ role: 'bot', text: t('globals.messages.somethingWentWrong') })
    ended.value = true
    return
  }
  currentStepId.value = nextStep.id
  transcript.value.push({ role: 'bot', text: questionText(nextStep) })
}
</script>
