import { ref, onMounted, onUnmounted } from 'vue'
import { calculateSla } from '../utils/sla'

export function useSla (dueAt, actualAt) {
    const sla = ref(null)
    function updateSla () {
        if (!dueAt.value) {
            sla.value = null
            return
        }
        sla.value = calculateSla(dueAt.value, actualAt.value)
    }
    onMounted(() => {
        updateSla()
        // Update the SLA every 30 seconds.
        const intervalId = setInterval(updateSla, 30000)
        onUnmounted(() => {
            clearInterval(intervalId)
        })
    })
    return sla
}
