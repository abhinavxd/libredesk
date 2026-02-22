import { ref, computed, watch } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '../composables/useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents'
import { adminNavItems, reportsNavItems } from '../constants/navigation'
import { filterNavItems } from '../utils/nav-permissions'
import api from '../api'
import { useStorage } from '@vueuse/core'

export const useUserStore = defineStore('user', () => {
  const user = ref({
    id: null,
    first_name: '',
    last_name: '',
    avatar_url: '',
    email: '',
    teams: [],
    permissions: [],
    roles: [],
    availability_status: 'offline',
  })
  const emitter = useEmitter()

  const userID = computed(() => user.value.id)
  const firstName = computed(() => user.value.first_name || '')
  const lastName = computed(() => user.value.last_name || '')
  const avatar = computed(() => user.value.avatar_url || '')
  const permissions = computed(() => user.value.permissions || [])
  const roles = computed(() => user.value.roles || [])
  const email = computed(() => user.value.email)
  const teams = computed(() => user.value.teams || [])

  const getFullName = computed(() => {
    const first = user.value.first_name ?? ''
    const last = user.value.last_name ?? ''
    if (!last) return first
    return `${first} ${last}`.trim()
  })

  const getInitials = computed(() => {
    const firstInitial = user.value.first_name?.charAt(0)?.toUpperCase() || ''
    const lastInitial = user.value.last_name?.charAt(0)?.toUpperCase() || ''
    return `${firstInitial}${lastInitial}`
  })

  const can = (permission) => {
    return user.value.permissions.includes(permission)
  }

  const hasAdminTabPermissions = computed(() => {
    return filterNavItems(adminNavItems, can).length > 0
  })

  const hasReportTabPermissions = computed(() => {
    return filterNavItems(reportsNavItems, can).length > 0
  })

  const getCurrentUser = async () => {
    try {
      const response = await api.getCurrentUser()
      const userData = response?.data?.data
      if (userData) {
        user.value = userData
      } else {
        throw new Error('No user data found')
      }
    } catch (error) {
      if (error.response?.status !== 401) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      }
    }
  }

  const setCurrentUser = (userData) => {
    user.value = userData
  }

  const setAvatar = (avatarURL) => {
    if (typeof avatarURL !== 'string') {
      console.warn('Avatar URL must be a string')
      return
    }
    user.value.avatar_url = avatarURL
  }

  const clearAvatar = () => {
    user.value.avatar_url = ''
  }

  // Set and watch user availability status in localStorage to sync across tabs
  const availabilityStatusStorage = useStorage('user_availability_status', user.value.availability_status)
  watch(availabilityStatusStorage, (newVal) => {
    user.value.availability_status = newVal
  })

  const updateUserAvailability = async (status, isManual = true) => {
    try {
      const apiStatus = status === 'away' && isManual ? 'away_manual' : status
      await api.updateCurrentUserAvailability({ status: apiStatus })
      user.value.availability_status = apiStatus
      availabilityStatusStorage.value = apiStatus
    } catch (error) {
      if (error?.response?.status === 401) window.location.href = '/'
    }
  }

  const hasAdminRole = computed(() => {
    return hasRole('Admin')
  })

  const hasRole = (role) => {
    return roles.value.some(r => r === role)
  }

  return {
    user,
    userID,
    firstName,
    lastName,
    avatar,
    hasAdminRole,
    hasRole,
    email,
    teams,
    permissions,
    roles,
    getFullName,
    getInitials,
    hasAdminTabPermissions,
    hasReportTabPermissions,
    setCurrentUser,
    getCurrentUser,
    clearAvatar,
    setAvatar,
    updateUserAvailability,
    can
  }
})