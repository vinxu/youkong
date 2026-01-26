import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Avatar, Button, Input, Loading, CircleTag } from '../components'
import { useInvitation } from '../hooks/useInvitation'
import { useCountdown } from '../hooks/useCountdown'
import { useAuthStore } from '../stores/authStore'
import authApi from '../api/auth'
import invitationApi from '../api/invitation'

export function InvitePage() {
  const { code } = useParams<{ code: string }>()
  const navigate = useNavigate()
  const { invitation, loading, error } = useInvitation(code)
  const { setAuth, setInviteContext, isAuthenticated } = useAuthStore()
  const { seconds, start: startCountdown, isActive: isCountdownActive } = useCountdown(60)

  const [phone, setPhone] = useState('')
  const [smsCode, setSmsCode] = useState('')
  const [phoneError, setPhoneError] = useState('')
  const [smsError, setSmsError] = useState('')
  const [sending, setSending] = useState(false)
  const [logging, setLogging] = useState(false)

  // 如果用户已登录，直接接受邀请
  useEffect(() => {
    if (isAuthenticated && invitation?.isValid && code) {
      acceptInvitation()
    }
  }, [isAuthenticated, invitation, code])

  const validatePhone = (value: string): boolean => {
    const phoneRegex = /^1[3-9]\d{9}$/
    if (!value) {
      setPhoneError('请输入手机号')
      return false
    }
    if (!phoneRegex.test(value)) {
      setPhoneError('请输入正确的手机号')
      return false
    }
    setPhoneError('')
    return true
  }

  const handleSendSms = async () => {
    if (!validatePhone(phone)) return

    setSending(true)
    try {
      const response = await authApi.sendSms(phone)
      if (response.code === 0) {
        startCountdown()
        setSmsError('')
      } else {
        setSmsError(response.message || '发送失败')
      }
    } catch (err) {
      setSmsError(err instanceof Error ? err.message : '发送失败')
    } finally {
      setSending(false)
    }
  }

  const acceptInvitation = async () => {
    if (!code) return

    try {
      const response = await invitationApi.accept(code)
      if (response.code === 0) {
        // 设置邀请上下文供欢迎页使用
        setInviteContext({
          inviter: invitation?.inviter || null,
          circle: response.data?.joinedCircle || invitation?.circle || null,
          code,
          isNewUser: false,
        })
        navigate('/welcome')
      }
    } catch (err) {
      console.error('接受邀请失败:', err)
      // 即使接受失败，也跳转到欢迎页
      navigate('/welcome')
    }
  }

  const handleLogin = async () => {
    if (!validatePhone(phone)) return
    if (!smsCode) {
      setSmsError('请输入验证码')
      return
    }
    if (smsCode.length !== 6) {
      setSmsError('验证码为6位数字')
      return
    }

    setLogging(true)
    try {
      const response = await authApi.verifySms(phone, smsCode)
      if (response.code === 0 && response.data) {
        const { token, user, isNewUser } = response.data
        setAuth(token, user)

        // 接受邀请
        if (code && invitation?.isValid) {
          try {
            const acceptRes = await invitationApi.accept(code)
            setInviteContext({
              inviter: invitation.inviter,
              circle: acceptRes.data?.joinedCircle || invitation.circle || null,
              code,
              isNewUser,
            })
          } catch {
            // 接受邀请失败，但登录成功，继续跳转
            setInviteContext({
              inviter: invitation.inviter,
              circle: invitation.circle || null,
              code,
              isNewUser,
            })
          }
        }

        navigate('/welcome')
      } else {
        setSmsError(response.message || '登录失败')
      }
    } catch (err) {
      setSmsError(err instanceof Error ? err.message : '登录失败')
    } finally {
      setLogging(false)
    }
  }

  // 加载中
  if (loading) {
    return <Loading text="正在获取邀请信息..." />
  }

  // 邀请无效或出错
  if (error || !invitation?.isValid) {
    return (
      <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center p-6">
        <div className="text-6xl mb-4">😢</div>
        <h1 className="text-xl font-semibold text-gray-900 mb-2">
          邀请链接已失效
        </h1>
        <p className="text-gray-500 text-center mb-6">
          {error || '该邀请链接可能已过期或已被使用'}
        </p>
        <Button variant="outline" onClick={() => navigate('/')}>
          返回首页
        </Button>
      </div>
    )
  }

  const { inviter, circle } = invitation

  return (
    <div className="min-h-screen bg-gradient-to-b from-primary-50 to-white flex flex-col">
      {/* 顶部装饰 */}
      <div className="h-2 bg-primary-500" />

      {/* 主内容 */}
      <div className="flex-1 flex flex-col items-center px-6 pt-12 pb-8">
        {/* 邀请者头像 */}
        <Avatar src={inviter.avatar} name={inviter.nickname} size="xl" />

        {/* 邀请文案 */}
        <h1 className="mt-4 text-2xl font-bold text-gray-900">
          {inviter.nickname}
        </h1>
        <p className="mt-2 text-lg text-gray-600">
          邀请你加入「有空」
        </p>

        {/* 圈子信息 */}
        {circle && (
          <div className="mt-4">
            <CircleTag
              name={circle.name}
              emoji={circle.emoji}
              color={circle.color}
              size="md"
            />
            {circle.memberCount !== undefined && (
              <p className="mt-2 text-sm text-gray-500 text-center">
                已有 {circle.memberCount} 位成员
              </p>
            )}
          </div>
        )}

        {/* 分割线 */}
        <div className="w-full max-w-sm mt-8 mb-6 border-t border-gray-200" />

        {/* 登录表单 */}
        <div className="w-full max-w-sm space-y-4">
          <Input
            type="tel"
            placeholder="请输入手机号"
            value={phone}
            onChange={(e) => {
              setPhone(e.target.value)
              setPhoneError('')
            }}
            error={phoneError}
            maxLength={11}
          />

          <Input
            type="text"
            inputMode="numeric"
            placeholder="请输入验证码"
            value={smsCode}
            onChange={(e) => {
              setSmsCode(e.target.value.replace(/\D/g, ''))
              setSmsError('')
            }}
            error={smsError}
            maxLength={6}
            suffix={
              <Button
                variant="ghost"
                size="sm"
                onClick={handleSendSms}
                disabled={isCountdownActive || sending}
                className="text-primary-500 whitespace-nowrap"
              >
                {sending
                  ? '发送中...'
                  : isCountdownActive
                  ? `${seconds}s`
                  : '获取验证码'}
              </Button>
            }
          />

          <Button
            className="w-full"
            size="lg"
            onClick={handleLogin}
            loading={logging}
            disabled={!phone || !smsCode}
          >
            登录并加入
          </Button>
        </div>

        {/* 底部说明 */}
        <p className="mt-6 text-xs text-gray-400 text-center">
          登录即表示同意「有空」的服务协议和隐私政策
        </p>
      </div>
    </div>
  )
}

export default InvitePage
