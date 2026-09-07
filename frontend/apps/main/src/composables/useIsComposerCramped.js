import { useMediaQuery } from '@vueuse/core'

// The inline composer needs ~280px of height.
export function useIsComposerCramped () {
  return useMediaQuery('(max-width: 767px), (max-height: 500px) and (pointer: coarse)')
}
