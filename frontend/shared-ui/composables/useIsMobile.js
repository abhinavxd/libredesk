import { useMediaQuery } from '@vueuse/core'

// 1023px is the exact complement of Tailwind's `lg:` (min-width: 1024px).
export function useIsMobile() {
  return useMediaQuery('(max-width: 1023px)')
}
