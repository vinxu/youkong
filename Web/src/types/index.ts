// API 响应格式
export interface ApiResponse<T> {
  code: number
  message: string
  data?: T
}

// 用户
export interface User {
  id: string
  phone?: string
  nickname: string
  avatar?: string
  wechatBound?: boolean
  createdAt?: string
  updatedAt?: string
}

export interface UserProfile {
  id: string
  nickname: string
  avatar?: string
}

// 圈子
export interface CircleInfo {
  id: string
  name: string
  emoji: string
  color?: string
  memberCount?: number
}

// 邀请
export type InvitationStatus = 'ACTIVE' | 'DISABLED' | 'EXPIRED'

export interface InvitationPublicInfo {
  inviter: UserProfile
  circle?: CircleInfo
  isValid: boolean
}

export interface AcceptInvitationResult {
  joinedCircle?: CircleInfo
}

// 登录
export interface LoginResult {
  token: string
  user: User
  isNewUser: boolean
}

// 有空状态
export type LocationType = 'PRESET' | 'FLEXIBLE' | 'CUSTOM'
export type AvailabilityStatus = 'ACTIVE' | 'EXPIRED' | 'CANCELLED' | 'FULFILLED'

export interface Location {
  type: LocationType
  name?: string
  latitude?: number
  longitude?: number
}

export interface Availability {
  id: string
  user: UserProfile
  startTime: string
  endTime: string
  location: Location
  status: AvailabilityStatus
  circleIds: string[]
  createdAt: string
}

// 页面状态
export interface InvitePageState {
  inviter: UserProfile
  circle?: CircleInfo
  isNewUser: boolean
}

// 消息
export type MessageType = 'TEXT' | 'AVAILABILITY_CARD' | 'CONFIRM_REQUEST' | 'CONFIRM_RESPONSE'

export interface Message {
  id: string
  sender: UserProfile
  type: MessageType
  content?: string
  metadata?: Record<string, unknown>
  createdAt: string
  isRead: boolean
}

export interface Conversation {
  id: string
  partner: UserProfile
  lastMessage?: Message
  unreadCount: number
  createdAt: string
}

// WebSocket 消息
export type WSMessageType = 'NEW_MESSAGE' | 'PING' | 'PONG'

export interface WSMessage {
  type: WSMessageType
  data?: {
    conversation_id: string
    message: Message
  }
}
