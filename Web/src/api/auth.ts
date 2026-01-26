import apiClient from './client'
import type { ApiResponse, LoginResult } from '../types'

export const authApi = {
  // 发送验证码
  sendSms: (phone: string): Promise<ApiResponse<null>> =>
    apiClient.post('/auth/sms/send', { phone }),

  // 验证码登录
  verifySms: (phone: string, code: string): Promise<ApiResponse<LoginResult>> =>
    apiClient.post('/auth/sms/verify', { phone, code }),

  // 刷新 Token
  refreshToken: (token: string): Promise<ApiResponse<{ token: string }>> =>
    apiClient.post('/auth/refresh', { token }),
}

export default authApi
