<script>
import { watch } from 'vue'
import { useRoute } from 'vue-router'
import { useSidebar } from '@shared-ui/components/ui/sidebar'

// Renderless. Tapping a link inside the mobile drawer navigates but leaves the
// Sheet open, hiding the page the tap just opened.
//
// This has to sit directly under SidebarProvider rather than inside one of the
// route-conditional <Sidebar> branches: navigating swaps which branch renders,
// so a watcher living in a branch unmounts before it can close the drawer, and
// the newly mounted branch then renders its Sheet already open.
export default {
  name: 'CloseDrawerOnNavigate',
  setup () {
    const route = useRoute()
    const { isMobile, openMobile, setOpenMobile } = useSidebar()

    watch(
      () => route.fullPath,
      () => {
        if (isMobile.value && openMobile.value) setOpenMobile(false)
      }
    )

    return () => null
  }
}
</script>
