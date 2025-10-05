import { ref } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'

export function useRelativeTime (timestamp) {
    const relativeTime = ref(getRelativeTime(timestamp))

    useIntervalFn(() => {
        relativeTime.value = getRelativeTime(timestamp)
    }, 60000)

    return relativeTime
}