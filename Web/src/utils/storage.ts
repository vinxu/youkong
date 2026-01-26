// Token 存储
export const TOKEN_KEY = 'token'
export const USER_KEY = 'user'
export const INVITE_CONTEXT_KEY = 'inviteContext'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function removeToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

// 用户信息存储
export function getUser<T>(): T | null {
  const userStr = localStorage.getItem(USER_KEY)
  if (!userStr) return null
  try {
    return JSON.parse(userStr)
  } catch {
    return null
  }
}

export function setUser<T>(user: T): void {
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

export function removeUser(): void {
  localStorage.removeItem(USER_KEY)
}

// 邀请上下文存储（使用 sessionStorage）
export function getInviteContext<T>(): T | null {
  const contextStr = sessionStorage.getItem(INVITE_CONTEXT_KEY)
  if (!contextStr) return null
  try {
    return JSON.parse(contextStr)
  } catch {
    return null
  }
}

export function setInviteContext<T>(context: T): void {
  sessionStorage.setItem(INVITE_CONTEXT_KEY, JSON.stringify(context))
}

export function removeInviteContext(): void {
  sessionStorage.removeItem(INVITE_CONTEXT_KEY)
}

// 清除所有存储
export function clearAll(): void {
  removeToken()
  removeUser()
  removeInviteContext()
}
