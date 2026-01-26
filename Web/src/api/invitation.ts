import apiClient from './client'
import type { ApiResponse, InvitationPublicInfo, AcceptInvitationResult } from '../types'

export const invitationApi = {
  // 获取邀请详情（公开接口，无需登录）
  getByCode: (code: string): Promise<ApiResponse<InvitationPublicInfo>> =>
    apiClient.get(`/invite/${code}`),

  // 接受邀请（需要登录）
  accept: (code: string): Promise<ApiResponse<AcceptInvitationResult>> =>
    apiClient.post(`/invite/${code}/accept`),
}

export default invitationApi
