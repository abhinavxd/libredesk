import { useMediaQuery } from '@vueuse/core'

// 1279px is the exact complement of Tailwind's `xl:` (min-width: 1280px).
export function useIsMobile() {
  return useMediaQuery('(max-width: 1279px)')
}
