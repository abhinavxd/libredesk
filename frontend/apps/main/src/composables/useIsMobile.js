import { useMediaQuery } from '@vueuse/core'

// Single source of truth for "is this a phone-sized viewport".
// 767px is the exact complement of Tailwind's `md:` (min-width: 768px), so a
// component can branch on `isMobile` in script and on `md:` in classes without
// the two disagreeing at the boundary. The sidebar primitive
// (shared-ui/components/ui/sidebar/SidebarProvider.vue) uses the same value.
export function useIsMobile () {
    return useMediaQuery('(max-width: 767px)')
}
