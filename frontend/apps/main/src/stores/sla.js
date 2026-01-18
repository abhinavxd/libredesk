import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import api from '../api'

export const useSlaStore = defineStore('sla', () => {
    const slas = ref([])
    const options = computed(() => slas.value.map(sla => ({
        label: sla.name,
        value: String(sla.id)
    })))
    const fetchSlas = async () => {
        if (slas.value.length) return
        try {
            const response = await api.getAllSLAs()
            slas.value = response?.data?.data || []
        } catch (error) {
            console.error(error)
        }
    }
    return {
        slas,
        options,
        fetchSlas
    }
})
