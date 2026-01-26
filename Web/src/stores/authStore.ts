import { create } from 'zustand'
import type { User, UserProfile, CircleInfo } from '../types'

interface AuthState {
  token: string | null
  user: User | null
  isAuthenticated: boolean

  // 邀请相关的临时状态
  inviteContext: {
    inviter: UserProfile | null
    circle: CircleInfo | null
    code: string | null
    isNewUser: boolean
  } | null

  // Actions
  setAuth: (token: string, user: User) => void
  setInviteContext: (context: AuthState['inviteContext']) => void
  clearInviteContext: () => void
  logout: () => void
  loadFromStorage: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  user: null,
  isAuthenticated: false,
  inviteContext: null,

  setAuth: (token, user) => {
    localStorage.setItem('token', token)
    localStorage.setItem('user', JSON.stringify(user))
    set({ token, user, isAuthenticated: true })
  },

  setInviteContext: (context) => {
    if (context) {
      sessionStorage.setItem('inviteContext', JSON.stringify(context))
    }
    set({ inviteContext: context })
  },

  clearInviteContext: () => {
    sessionStorage.removeItem('inviteContext')
    set({ inviteContext: null })
  },

  logout: () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    sessionStorage.removeItem('inviteContext')
    set({ token: null, user: null, isAuthenticated: false, inviteContext: null })
  },

  loadFromStorage: () => {
    const token = localStorage.getItem('token')
    const userStr = localStorage.getItem('user')
    const inviteContextStr = sessionStorage.getItem('inviteContext')

    if (token && userStr) {
      try {
        const user = JSON.parse(userStr)
        set({ token, user, isAuthenticated: true })
      } catch {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
      }
    }

    if (inviteContextStr) {
      try {
        const inviteContext = JSON.parse(inviteContextStr)
        set({ inviteContext })
      } catch {
        sessionStorage.removeItem('inviteContext')
      }
    }
  },
}))

export default useAuthStore
