/**
 * YouKong 共享数据类型定义
 * 
 * 所有端使用相同的类型定义：
 * - 后端: 直接使用 TypeScript
 * - iOS/Android: 根据此文件生成对应语言代码
 * - 小程序: TypeScript 直接使用
 */

// ============ 基础类型 ============

export interface User {
  id: string
  nickname: string
  avatar?: string
  phone?: string
  createdAt: number  // Unix timestamp (ms)
  lastActiveAt: number
}

export interface Circle {
  id: string
  name: string       // 最多10字
  emoji: string
  color: string      // Hex color, e.g. "#EC4899"
  ownerId: string
  memberIds: string[]
  createdAt: number
  updatedAt: number
}

export interface Availability {
  id: string
  userId: string
  startTime: number  // Unix timestamp (ms)
  endTime: number
  location: AvailabilityLocation
  visibleCircleIds: string[]
  status: AvailabilityStatus
  createdAt: number
  updatedAt: number
}

export interface AvailabilityLocation {
  type: LocationType
  name?: string
  latitude?: number
  longitude?: number
  radius?: number  // 范围（米）
}

export type LocationType = 'PRESET' | 'FLEXIBLE' | 'CUSTOM'

export type AvailabilityStatus = 'ACTIVE' | 'EXPIRED' | 'CANCELLED' | 'FULFILLED'

export interface AvailabilityWithUser extends Availability {
  user: User
}

export interface Message {
  id: string
  conversationId: string
  senderId: string
  type: MessageType
  content: string
  metadata?: Record<string, unknown>
  createdAt: number
  readAt?: number
}

export type MessageType = 'TEXT' | 'AVAILABILITY_CARD' | 'CONFIRM_REQUEST' | 'CONFIRM_RESPONSE'

export interface Conversation {
  id: string
  participantIds: string[]
  lastMessage?: Message
  unreadCount: number
  createdAt: number
  updatedAt: number
}

// ============ API 响应类型 ============

export interface ApiResponse<T> {
  code: number  // 0 = 成功
  message: string
  data: T
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  size: number
}

// ============ API 请求类型 ============

export interface CreateAvailabilityRequest {
  startTime: number
  endTime: number
  location: AvailabilityLocation
  visibleCircleIds: string[]  // 至少1个
}

export interface CreateCircleRequest {
  name: string  // 最多10字
  emoji: string
}

export interface UpdateCircleRequest {
  name?: string
  emoji?: string
}

export interface AddMembersRequest {
  userIds: string[]
}

export interface WechatLoginRequest {
  code: string
}

export interface SmsLoginRequest {
  phone: string
  code: string
}

export interface SendSmsRequest {
  phone: string
}

// ============ AI 服务类型 ============

export interface ContactInfo {
  name: string
  phoneHash?: string  // 单向哈希，保护隐私
  tags?: string[]
}

export interface GroupSource {
  groupName?: string
  groupId?: string
  timestamp: number
}

export interface AnalyzeContactsRequest {
  contacts: ContactInfo[]
  groupSources?: GroupSource[]
}

export interface AICircleSuggestion {
  name: string
  emoji: string
  members: string[]
  reason: string
  priority: number  // 1-5, 1最高
}

export interface AICircleSuggestionsResponse {
  circles: AICircleSuggestion[]
}

export interface SuggestCircleNameRequest {
  members: string[]
}

export interface CircleNameSuggestion {
  name: string
  emoji: string
  style: 'formal' | 'casual' | 'fun'
}

export interface SuggestCircleNameResponse {
  suggestions: CircleNameSuggestion[]
  recommended: number  // 推荐第几个 (0-based)
}

// ============ 认证响应类型 ============

export interface AuthResponse {
  accessToken: string
  refreshToken: string
  expiresIn: number  // 秒
  user: User
}

// ============ 错误码 ============

export const ErrorCodes = {
  SUCCESS: 0,
  INVALID_PARAMS: 1001,
  UNAUTHORIZED: 1002,
  TOKEN_EXPIRED: 1003,
  USER_NOT_FOUND: 2001,
  INVALID_SMS_CODE: 2002,
  CIRCLE_NOT_FOUND: 3001,
  CIRCLE_NO_PERMISSION: 3002,
  AVAILABILITY_NOT_FOUND: 4001,
  SERVER_ERROR: 5001,
} as const

export type ErrorCode = typeof ErrorCodes[keyof typeof ErrorCodes]

// ============ 预设数据 ============

export interface PresetLocation {
  name: string
  emoji: string
  latitude: number
  longitude: number
}

export const PRESET_LOCATIONS: PresetLocation[] = [
  { name: '三里屯', emoji: '🛍️', latitude: 39.9334, longitude: 116.4551 },
  { name: '国贸', emoji: '🏢', latitude: 39.9087, longitude: 116.4605 },
  { name: '望京', emoji: '🌆', latitude: 39.9982, longitude: 116.4744 },
  { name: '中关村', emoji: '💻', latitude: 39.9836, longitude: 116.3164 },
  { name: '五道口', emoji: '🎓', latitude: 39.9927, longitude: 116.3377 },
  { name: '朝阳大悦城', emoji: '🎡', latitude: 39.9214, longitude: 116.5088 },
]

export const FLEXIBLE_LOCATIONS = [
  { type: 'FLEXIBLE' as const, name: '都行，你定', emoji: '🤷' },
  { type: 'FLEXIBLE' as const, name: '我可以过去', emoji: '🚗' },
]

export const CIRCLE_COLORS = [
  { name: 'pink', value: '#EC4899' },
  { name: 'orange', value: '#F97316' },
  { name: 'blue', value: '#3B82F6' },
  { name: 'green', value: '#22C55E' },
  { name: 'purple', value: '#8B5CF6' },
  { name: 'yellow', value: '#EAB308' },
]

// ============ UI 常量 ============

export const COLORS = {
  primary: '#10B981',
  primaryDark: '#059669',
  primaryLight: '#34D399',
  secondary: '#14B8A6',
  secondaryDark: '#0D9488',
  
  gray50: '#F9FAFB',
  gray100: '#F3F4F6',
  gray200: '#E5E7EB',
  gray300: '#D1D5DB',
  gray400: '#9CA3AF',
  gray500: '#6B7280',
  gray600: '#4B5563',
  gray700: '#374151',
  gray800: '#1F2937',
  gray900: '#111827',
  
  success: '#22C55E',
  warning: '#F59E0B',
  error: '#EF4444',
  info: '#3B82F6',
} as const

export const SPACING = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 20,
  xxl: 24,
  xxxl: 32,
} as const

export const RADIUS = {
  sm: 8,
  md: 12,
  lg: 16,
  xl: 20,
  xxl: 24,
} as const

export const TYPOGRAPHY = {
  headlineLarge: { size: 28, weight: 'bold' },
  headlineMedium: { size: 24, weight: 'semibold' },
  titleLarge: { size: 20, weight: 'semibold' },
  titleMedium: { size: 16, weight: 'semibold' },
  bodyLarge: { size: 16, weight: 'regular' },
  bodyMedium: { size: 14, weight: 'regular' },
  labelMedium: { size: 12, weight: 'medium' },
} as const
