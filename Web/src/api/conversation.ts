import apiClient from './client'
import type { ApiResponse, Conversation, Message } from '../types'

const conversationApi = {
  // 获取会话列表
  getConversations: (): Promise<ApiResponse<Conversation[]>> =>
    apiClient.get('/conversations'),

  // 创建会话
  createConversation: (partnerId: string): Promise<ApiResponse<Conversation>> =>
    apiClient.post('/conversations', { partnerId }),

  // 获取消息列表
  getMessages: (conversationId: string, limit = 20, offset = 0): Promise<ApiResponse<Message[]>> =>
    apiClient.get(
      `/conversations/${conversationId}/messages`,
      { params: { limit, offset } }
    ),

  // 发送消息
  sendMessage: (conversationId: string, content: string): Promise<ApiResponse<Message>> =>
    apiClient.post(
      `/conversations/${conversationId}/messages`,
      { type: 'TEXT', content }
    ),

  // Agent 回复
  agentReply: (conversationId: string): Promise<ApiResponse<Message>> =>
    apiClient.post(
      `/conversations/${conversationId}/agent-reply`
    ),
}

export default conversationApi
