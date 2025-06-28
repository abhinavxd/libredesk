import { computed, ref, watch } from 'vue'
import { useFieldValue, useFormValues } from 'vee-validate'
import { createContext } from 'radix-vue'
import { DependencyType } from './interface'
import { getFromPath, getIndexIfArray } from './utils'

export const [injectDependencies, provideDependencies] = createContext('AutoFormDependencies')

export default function useDependencies(fieldName) {
  const form = useFormValues()
  // parsed test[0].age => test.age
  const currentFieldName = fieldName.replace(/\[\d+\]/g, '')
  const currentFieldValue = useFieldValue(fieldName)

  if (!form) throw new Error('useDependencies should be used within <AutoForm>')

  const dependencies = injectDependencies()
  const isDisabled = ref(false)
  const isHidden = ref(false)
  const isRequired = ref(false)
  const overrideOptions = ref()

  const currentFieldDependencies = computed(() =>
    dependencies.value?.filter((dependency) => dependency.targetField === currentFieldName)
  )

  function getSourceValue(dep) {
    const source = dep.sourceField
    const index = getIndexIfArray(fieldName) ?? -1
    const [sourceLast, ...sourceInitial] = source.split('.').toReversed()
    const [_targetLast, ...targetInitial] = dep.targetField.split('.').toReversed()

    if (index >= 0 && sourceInitial.join(',') === targetInitial.join(',')) {
      const [_currentLast, ...currentInitial] = fieldName.split('.').toReversed()
      return getFromPath(form.value, currentInitial.join('.') + sourceLast)
    }

    return getFromPath(form.value, source)
  }

  const sourceFieldValues = computed(() =>
    currentFieldDependencies.value?.map((dep) => getSourceValue(dep))
  )

  const resetConditionState = () => {
    isDisabled.value = false
    isHidden.value = false
    isRequired.value = false
    overrideOptions.value = undefined
  }

  watch(
    [sourceFieldValues, dependencies],
    () => {
      resetConditionState()
      currentFieldDependencies.value?.forEach((dep) => {
        const sourceValue = getSourceValue(dep)
        const conditionMet = dep.when(sourceValue, currentFieldValue.value)

        switch (dep.type) {
          case DependencyType.DISABLES:
            if (conditionMet) isDisabled.value = true

            break
          case DependencyType.REQUIRES:
            if (conditionMet) isRequired.value = true

            break
          case DependencyType.HIDES:
            if (conditionMet) isHidden.value = true

            break
          case DependencyType.SETS_OPTIONS:
            if (conditionMet) overrideOptions.value = dep.options

            break
        }
      })
    },
    { immediate: true, deep: true }
  )

  return {
    isDisabled,
    isHidden,
    isRequired,
    overrideOptions
  }
}
