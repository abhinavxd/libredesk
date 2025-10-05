import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useUserStore } from '../stores/user'
import { debounce } from '../utils/debounce'
import { useStorage } from '@vueuse/core'

export function useIdleDetection () {
    const userStore = useUserStore()
    const AWAY_THRESHOLD = 4 * 60 * 1000
    const CHECK_INTERVAL = 30 * 1000

    const lastActivity = useStorage('last_active', Date.now())
    const timer = ref(null)

    // Debounce the goOnline to prevent it from being called too frequently
    const goOnline = debounce(() => {
        if (userStore.user.availability_status === 'away' || userStore.user.availability_status === 'offline') {
            userStore.updateUserAvailability('online', false)
        }
    }, 200)

    function resetTimer () {
        lastActivity.value = Date.now()
    }

    function checkIdle () {
        if (
            Date.now() - lastActivity.value > AWAY_THRESHOLD &&
            userStore.user.availability_status === 'online'
        ) {
            userStore.updateUserAvailability('away', false)
        }
    }

    onMounted(() => {
        ['mousemove', 'keypress', 'click'].forEach(evt =>
            window.addEventListener(evt, resetTimer)
        )
        timer.value = setInterval(checkIdle, CHECK_INTERVAL)
    })

    onBeforeUnmount(() => {
        ['mousemove', 'keypress', 'click'].forEach(evt =>
            window.removeEventListener(evt, resetTimer)
        )
        clearInterval(timer.value)
    })

    watch(lastActivity, (newVal, oldVal) => {
        if (
            newVal > oldVal &&
            document.visibilityState === 'visible'
        ) {
            goOnline()
        }
    })
}
