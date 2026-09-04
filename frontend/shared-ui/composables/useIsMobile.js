import { useMediaQuery } from '@vueuse/core'

// 767px is the exact complement of Tailwind's `md:` (min-width: 768px).
export function useIsMobile () {
  return useMediaQuery('(max-width: 767px)')
}
