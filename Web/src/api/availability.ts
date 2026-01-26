import apiClient from './client'
import type { ApiResponse, Availability } from '../types'

export const availabilityApi = {
  // 获取好友的有空状态
  getFriendsAvailabilities: (): Promise<ApiResponse<Availability[]>> =>
    apiClient.get('/availabilities/friends'),

  // 获取指定用户的有空状态（通过好友关系过滤）
  getUserAvailabilities: (userId: string): Promise<ApiResponse<Availability[]>> =>
    apiClient.get(`/availabilities/friends`, { params: { userId } }),
}

export default availabilityApi
