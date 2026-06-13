import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import api from '../api'

export const useCountriesStore = defineStore('countries', () => {
  const countries = ref([])
  const emitter = useEmitter()

  const allCountries = computed(() =>
    countries.value.map((country) => ({
      label: country.name,
      value: country.iso_2,
      emoji: country.emoji,
      calling_code: country.calling_code
    }))
  )

  const countryOptions = computed(() =>
    countries.value.map((country) => ({
      label: country.name,
      value: country.iso_2,
      emoji: country.emoji
    }))
  )

  const fetchCountries = async () => {
    if (countries.value.length) return
    try {
      const response = await api.getCountries()
      countries.value = response?.data?.data || []
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }

  return {
    countries,
    allCountries,
    countryOptions,
    fetchCountries
  }
})
