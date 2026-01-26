# CLAUDE.md - 微信小程序

> ⚠️ **先阅读** `../CLAUDE.md` **了解共享定义**

---

## 我的职责

- 实现微信小程序界面
- 微信登录（wx.login）
- 微信分享（群/好友）
- 群来源追踪（场景值）
- 订阅消息推送
- 调用后端 API

---

## 技术栈

```
Framework:   微信原生小程序
Language:    TypeScript
UI Library:  Vant Weapp
State:       MobX-miniprogram
```

---

## 项目结构

```
miniprogram/
├── app.ts
├── app.json
├── app.wxss
├── pages/
│   ├── home/
│   │   ├── index.ts
│   │   ├── index.wxml
│   │   ├── index.wxss
│   │   └── index.json
│   ├── publish/
│   │   ├── index.ts
│   │   ├── index.wxml
│   │   └── ...
│   ├── circles/
│   │   └── ...
│   ├── chat/
│   │   └── ...
│   └── profile/
│       └── ...
├── components/
│   ├── availability-card/
│   │   ├── index.ts
│   │   ├── index.wxml
│   │   ├── index.wxss
│   │   └── index.json
│   ├── circle-chip/
│   │   └── ...
│   └── time-picker/
│       └── ...
├── services/
│   ├── api.ts           # API 客户端
│   ├── auth.ts          # 认证服务
│   ├── ai-circle.ts     # AI建圈服务
│   └── share.ts         # 分享服务
├── stores/
│   ├── user.ts
│   ├── availability.ts
│   └── circle.ts
├── utils/
│   ├── request.ts       # 网络请求封装
│   ├── storage.ts       # 本地存储
│   └── formatters.ts    # 格式化工具
└── types/
    └── index.d.ts       # 类型定义
```

---

## 微信特有能力

### 场景值追踪

```typescript
// app.ts
App({
  onLaunch(options) {
    this.trackLaunchSource(options)
  },
  
  trackLaunchSource(options: WechatMiniprogram.App.LaunchShowOption) {
    const { scene, query } = options
    
    // 场景值判断
    // 1007: 单人聊天会话中的小程序消息卡片
    // 1008: 群聊会话中的小程序消息卡片  
    // 1044: 带 shareTicket 的小程序消息卡片（可获取群信息）
    
    const isGroupShare = [1007, 1008, 1044].includes(scene)
    
    if (isGroupShare && query.from === 'share') {
      // 记录群来源，用于建圈推荐
      this.recordGroupSource(query, options.shareTicket)
    }
    
    wx.setStorageSync('launch_source', {
      scene,
      query,
      isGroupShare,
      timestamp: Date.now()
    })
  },
  
  async recordGroupSource(query: Record<string, string>, shareTicket?: string) {
    if (shareTicket) {
      try {
        // 获取群信息需要用户授权
        const groupInfo = await this.getGroupInfo(shareTicket)
        const sources = wx.getStorageSync('group_sources') || []
        sources.push({
          ...groupInfo,
          availabilityId: query.id,
          timestamp: Date.now()
        })
        wx.setStorageSync('group_sources', sources)
      } catch (e) {
        console.log('获取群信息失败', e)
      }
    }
  }
})
```

### 分享卡片

```typescript
// pages/availability/detail.ts
Page({
  data: {
    availability: null as Availability | null
  },
  
  onShareAppMessage(): WechatMiniprogram.Page.ICustomShareContent {
    const { availability } = this.data
    if (!availability) return {}
    
    return {
      title: `🟢 ${availability.userName} ${this.formatTime(availability.startTime)}有空`,
      path: `/pages/availability/view?id=${availability.id}&from=share`,
      imageUrl: '/images/share-card.png'
    }
  },
  
  onShareTimeline(): WechatMiniprogram.Page.ICustomTimelineContent {
    const { availability } = this.data
    if (!availability) return {}
    
    return {
      title: `我${this.formatTime(availability.startTime)}有空，约吗？`,
      query: `id=${availability.id}&from=timeline`
    }
  },
  
  formatTime(timestamp: number): string {
    const date = new Date(timestamp)
    const now = new Date()
    
    if (this.isToday(date)) {
      return `今晚 ${date.getHours()}:00`
    } else if (this.isTomorrow(date)) {
      return `明天 ${date.getHours()}:00`
    } else {
      return `${date.getMonth() + 1}/${date.getDate()} ${date.getHours()}:00`
    }
  }
})
```

### 微信登录

```typescript
// services/auth.ts
export class AuthService {
  async login(): Promise<UserInfo> {
    // 1. 获取微信 code
    const { code } = await wx.login()
    
    // 2. 发送到后端换取 token
    const response = await request.post('/auth/wechat', { code })
    
    // 3. 存储 token
    wx.setStorageSync('access_token', response.data.accessToken)
    wx.setStorageSync('user_info', response.data.user)
    
    return response.data.user
  }
  
  async getUserProfile(): Promise<WechatMiniprogram.UserInfo> {
    // 需要用户点击按钮触发
    const { userInfo } = await wx.getUserProfile({
      desc: '用于完善您的个人资料'
    })
    return userInfo
  }
}
```

### 订阅消息

```typescript
// services/push.ts
export class PushService {
  // 订阅消息模板 ID
  private templateIds = {
    friendAvailable: 'xxx',  // 朋友有空通知
    inviteReceived: 'xxx',   // 收到邀约通知
    inviteConfirmed: 'xxx'   // 邀约确认通知
  }
  
  async requestSubscription(): Promise<void> {
    try {
      await wx.requestSubscribeMessage({
        tmplIds: Object.values(this.templateIds)
      })
    } catch (e) {
      console.log('订阅消息授权失败', e)
    }
  }
}
```

---

## API 客户端

```typescript
// utils/request.ts
const BASE_URL = 'https://api.youkong.app/v1'

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  data?: any
  header?: Record<string, string>
}

export async function request<T>(url: string, options: RequestOptions = {}): Promise<ApiResponse<T>> {
  const token = wx.getStorageSync('access_token')
  
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${BASE_URL}${url}`,
      method: options.method || 'GET',
      data: options.data,
      header: {
        'Content-Type': 'application/json',
        'Authorization': token ? `Bearer ${token}` : '',
        ...options.header
      },
      success(res) {
        const data = res.data as ApiResponse<T>
        if (data.code === 0) {
          resolve(data)
        } else if (data.code === 1003) {
          // Token 过期，重新登录
          wx.navigateTo({ url: '/pages/login/index' })
          reject(new Error('Token expired'))
        } else {
          reject(new Error(data.message))
        }
      },
      fail(err) {
        reject(err)
      }
    })
  })
}

export const api = {
  get: <T>(url: string) => request<T>(url, { method: 'GET' }),
  post: <T>(url: string, data: any) => request<T>(url, { method: 'POST', data }),
  put: <T>(url: string, data: any) => request<T>(url, { method: 'PUT', data }),
  delete: <T>(url: string) => request<T>(url, { method: 'DELETE' })
}
```

---

## 页面示例

### 首页

```html
<!-- pages/home/index.wxml -->
<view class="container">
  <!-- 我的有空状态 -->
  <view wx:if="{{myAvailability}}" class="my-availability">
    <my-availability-card 
      availability="{{myAvailability}}"
      bind:cancel="onCancelAvailability"
    />
  </view>
  
  <!-- 今天有空的朋友 -->
  <view class="section">
    <view class="section-title">🟢 今天有空的朋友</view>
    <availability-card 
      wx:for="{{todayAvailabilities}}" 
      wx:key="id"
      availability="{{item}}"
      bind:invite="onInvite"
    />
  </view>
  
  <!-- 之后有空 -->
  <view class="section">
    <view class="section-title">🟡 之后有空</view>
    <availability-card 
      wx:for="{{laterAvailabilities}}" 
      wx:key="id"
      availability="{{item}}"
      dimmed="{{true}}"
      bind:invite="onInvite"
    />
  </view>
  
  <!-- 发布按钮 -->
  <view class="fab" bind:tap="onPublish">
    <text class="fab-icon">+</text>
  </view>
</view>
```

```typescript
// pages/home/index.ts
import { api } from '../../utils/request'

Page({
  data: {
    myAvailability: null,
    todayAvailabilities: [],
    laterAvailabilities: [],
    isLoading: false
  },
  
  onLoad() {
    this.loadData()
  },
  
  onPullDownRefresh() {
    this.loadData().then(() => {
      wx.stopPullDownRefresh()
    })
  },
  
  async loadData() {
    this.setData({ isLoading: true })
    
    try {
      const res = await api.get<PaginatedResponse<AvailabilityWithUser>>('/availabilities/friends')
      const all = res.data.items
      
      const today = new Date()
      today.setHours(0, 0, 0, 0)
      const tomorrow = new Date(today)
      tomorrow.setDate(tomorrow.getDate() + 1)
      
      this.setData({
        todayAvailabilities: all.filter(a => a.startTime >= today.getTime() && a.startTime < tomorrow.getTime()),
        laterAvailabilities: all.filter(a => a.startTime >= tomorrow.getTime()),
        isLoading: false
      })
    } catch (e) {
      this.setData({ isLoading: false })
      wx.showToast({ title: '加载失败', icon: 'none' })
    }
  },
  
  onPublish() {
    wx.navigateTo({ url: '/pages/publish/index' })
  },
  
  onInvite(e: WechatMiniprogram.CustomEvent<{ userId: string }>) {
    const { userId } = e.detail
    wx.navigateTo({ url: `/pages/chat/index?userId=${userId}` })
  }
})
```

---

## 样式变量

```css
/* app.wxss */
page {
  --primary: #10B981;
  --primary-dark: #059669;
  --secondary: #14B8A6;
  
  --gray-50: #F9FAFB;
  --gray-100: #F3F4F6;
  --gray-200: #E5E7EB;
  --gray-400: #9CA3AF;
  --gray-500: #6B7280;
  --gray-600: #4B5563;
  --gray-800: #1F2937;
  
  --spacing-xs: 8rpx;
  --spacing-sm: 16rpx;
  --spacing-md: 24rpx;
  --spacing-lg: 32rpx;
  --spacing-xl: 40rpx;
  
  --radius-sm: 16rpx;
  --radius-md: 24rpx;
  --radius-lg: 32rpx;
}
```

---

## 待处理事件

检查 `../shared/events/` 目录获取待实现的功能。

---

*版本: 1.0.0*
