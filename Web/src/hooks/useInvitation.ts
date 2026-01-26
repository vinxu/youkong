import { useState, useEffect, useCallback } from 'react'
import invitationApi from '../api/invitation'
import type { InvitationPublicInfo } from '../types'

interface UseInvitationResult {
  invitation: InvitationPublicInfo | null
  loading: boolean
  error: string | null
  refetch: () => Promise<void>
}

export function useInvitation(code: string | undefined): UseInvitationResult {
  const [invitation, setInvitation] = useState<InvitationPublicInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchInvitation = useCallback(async () => {
    if (!code) {
      setLoading(false)
      setError('无效的邀请链接')
      return
    }

    setLoading(true)
    setError(null)

    try {
      const response = await invitationApi.getByCode(code)
      if (response.code === 0 && response.data) {
        setInvitation(response.data)
      } else {
        setError(response.message || '获取邀请信息失败')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '网络错误')
    } finally {
      setLoading(false)
    }
  }, [code])

  useEffect(() => {
    fetchInvitation()
  }, [fetchInvitation])

  return {
    invitation,
    loading,
    error,
    refetch: fetchInvitation,
  }
}

export default useInvitation
