# 邀请系统 API 文档

## Base URL
```
http://49.232.13.41:8080/api/v1
```

## 认证方式
```
Authorization: Bearer <token>
```

---

## 1. 认证接口

### 1.1 微信授权登录

**POST** `/auth/wechat/login`

微信OAuth授权登录，支持携带邀请码。

**请求参数：**
```json
{
  "code": "微信授权码",
  "inviteCode": "ABC123XY"  // 可选，邀请码
}
```

**响应示例：**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "uuid",
      "nickname": "微信昵称",
      "avatar": "https://...",
      "wechatBound": true,
      "createdAt": "2024-01-20T10:00:00Z",
      "updatedAt": "2024-01-20T10:00:00Z"
    },
    "isNewUser": true,
    "joinedCircle": {
      "id": "circle_uuid",
      "name": "闺蜜",
      "emoji": "💕",
      "memberCount": 5
    }
  }
}
```

**错误码：**
| 错误码 | 说明 |
|--------|------|
| 1001 | 微信授权码无效 |
| 5000 | 微信服务异常 |

---

## 2. 邀请接口

### 2.1 创建邀请链接

**POST** `/invitations`

创建一个新的邀请链接。

**请求头：**
```
Authorization: Bearer <token>
```

**请求参数：**
```json
{
  "circleId": "circle_uuid",  // 可选，关联圈子
  "maxUses": 100,             // 可选，默认100
  "expiresDays": 7            // 可选，默认7天，最长30天
}
```

**响应示例：**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "id": "invitation_uuid",
    "code": "ABC123XY",
    "inviteUrl": "https://youkong.app/i/ABC123XY",
    "inviter": {
      "id": "user_uuid",
      "nickname": "小明",
      "avatar": "https://..."
    },
    "circle": {
      "id": "circle_uuid",
      "name": "闺蜜",
      "emoji": "💕",
      "color": "#EC4899",
      "memberCount": 5
    },
    "maxUses": 100,
    "useCount": 0,
    "expiresAt": "2024-01-27T10:00:00Z",
    "status": "ACTIVE",
    "isValid": true,
    "createdAt": "2024-01-20T10:00:00Z"
  }
}
```

**错误码：**
| 错误码 | 说明 |
|--------|------|
| 1001 | 参数错误 |
| 1005 | 无权限创建此圈子的邀请 |
| 5000 | 今日邀请次数已达上限 |

---

### 2.2 获取我的邀请列表

**GET** `/invitations`

获取当前用户创建的所有邀请链接。

**请求头：**
```
Authorization: Bearer <token>
```

**响应示例：**
```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "id": "invitation_uuid",
      "code": "ABC123XY",
      "inviteUrl": "https://youkong.app/i/ABC123XY",
      "circle": {
        "id": "circle_uuid",
        "name": "闺蜜",
        "emoji": "💕"
      },
      "maxUses": 100,
      "useCount": 5,
      "expiresAt": "2024-01-27T10:00:00Z",
      "status": "ACTIVE",
      "isValid": true,
      "createdAt": "2024-01-20T10:00:00Z"
    }
  ]
}
```

---

### 2.3 获取邀请详情（公开）

**GET** `/invitations/:code`

根据邀请码获取邀请详情，用于落地页展示。**无需认证**。

**路径参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| code | string | 8位邀请码 |

**响应示例：**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "inviter": {
      "id": "user_uuid",
      "nickname": "小明",
      "avatar": "https://..."
    },
    "circle": {
      "id": "circle_uuid",
      "name": "闺蜜",
      "emoji": "💕",
      "memberCount": 5
    },
    "isValid": true
  }
}
```

**错误码：**
| 错误码 | 说明 |
|--------|------|
| 1004 | 邀请不存在 |

---

### 2.4 获取邀请完整信息

**GET** `/invitations/:id/detail`

获取邀请的完整信息，需要认证。

**请求头：**
```
Authorization: Bearer <token>
```

**路径参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 邀请ID |

**响应示例：**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "id": "invitation_uuid",
    "code": "ABC123XY",
    "inviteUrl": "https://youkong.app/i/ABC123XY",
    "inviter": {
      "id": "user_uuid",
      "nickname": "小明",
      "avatar": "https://..."
    },
    "circle": {
      "id": "circle_uuid",
      "name": "闺蜜",
      "emoji": "💕",
      "color": "#EC4899",
      "memberCount": 5
    },
    "maxUses": 100,
    "useCount": 5,
    "expiresAt": "2024-01-27T10:00:00Z",
    "status": "ACTIVE",
    "isValid": true,
    "createdAt": "2024-01-20T10:00:00Z"
  }
}
```

---

### 2.5 禁用邀请链接

**DELETE** `/invitations/:id`

禁用一个邀请链接，只能禁用自己创建的邀请。

**请求头：**
```
Authorization: Bearer <token>
```

**路径参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 邀请ID |

**响应示例：**
```json
{
  "code": 0,
  "message": "邀请已禁用",
  "data": null
}
```

**错误码：**
| 错误码 | 说明 |
|--------|------|
| 1004 | 邀请不存在 |
| 1005 | 无权限禁用此邀请 |

---

### 2.6 接受邀请

**POST** `/invitations/:code/accept`

接受邀请，自动建立好友关系并加入圈子。

**请求头：**
```
Authorization: Bearer <token>
```

**路径参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| code | string | 8位邀请码 |

**响应示例：**
```json
{
  "code": 0,
  "message": "已接受邀请",
  "data": {
    "joinedCircle": {
      "id": "circle_uuid",
      "name": "闺蜜",
      "emoji": "💕",
      "color": "#EC4899",
      "memberCount": 6
    }
  }
}
```

**错误码：**
| 错误码 | 说明 |
|--------|------|
| 1001 | 不能接受自己的邀请 |
| 1004 | 邀请不存在 |
| 5000 | 邀请已失效 / 已接受过此邀请 |

---

### 2.7 获取邀请海报

**GET** `/invitations/:id/poster`

生成并返回邀请海报图片（PNG格式）。

**请求头：**
```
Authorization: Bearer <token>
```

**路径参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 邀请ID |

**响应：**
```
Content-Type: image/png
Content-Disposition: inline; filename="invite_poster.png"

<PNG 图片二进制数据>
```

**错误码：**
| 错误码 | 说明 |
|--------|------|
| 1004 | 邀请不存在 |
| 1005 | 无权限获取此海报 |

---

### 2.8 获取邀请二维码

**GET** `/invitations/:id/qrcode`

生成并返回邀请二维码图片（PNG格式）。

**请求头：**
```
Authorization: Bearer <token>
```

**路径参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 邀请ID |

**响应：**
```
Content-Type: image/png
Content-Disposition: inline; filename="invite_qrcode.png"

<PNG 图片二进制数据>
```

---

## 3. 好友接口

### 3.1 获取好友列表

**GET** `/friends`

获取当前用户的好友列表。

**请求头：**
```
Authorization: Bearer <token>
```

**响应示例：**
```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "user": {
        "id": "user_uuid",
        "nickname": "小红",
        "avatar": "https://..."
      },
      "source": "INVITATION",
      "createdAt": "2024-01-20T10:00:00Z"
    },
    {
      "user": {
        "id": "user_uuid2",
        "nickname": "小蓝",
        "avatar": "https://..."
      },
      "source": "SEARCH",
      "createdAt": "2024-01-19T10:00:00Z"
    }
  ]
}
```

---

### 3.2 删除好友

**DELETE** `/friends/:userId`

删除一个好友（双向删除）。

**请求头：**
```
Authorization: Bearer <token>
```

**路径参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| userId | string | 好友用户ID |

**响应示例：**
```json
{
  "code": 0,
  "message": "已删除好友",
  "data": null
}
```

**错误码：**
| 错误码 | 说明 |
|--------|------|
| 5000 | 不是好友关系 |

---

### 3.3 获取我邀请的好友

**GET** `/friends/invited-by-me`

获取通过我的邀请链接加入的好友。

**请求头：**
```
Authorization: Bearer <token>
```

**响应示例：**
```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "user": {
        "id": "user_uuid",
        "nickname": "小红",
        "avatar": "https://..."
      },
      "source": "INVITATION",
      "circleName": "闺蜜",
      "createdAt": "2024-01-20T10:00:00Z"
    }
  ]
}
```

---

### 3.4 获取邀请我的好友

**GET** `/friends/invited-me`

获取邀请我加入的好友。

**请求头：**
```
Authorization: Bearer <token>
```

**响应示例：**
```json
{
  "code": 0,
  "message": "成功",
  "data": [
    {
      "user": {
        "id": "user_uuid",
        "nickname": "小明",
        "avatar": "https://..."
      },
      "source": "INVITATION",
      "circleName": "闺蜜",
      "createdAt": "2024-01-20T10:00:00Z"
    }
  ]
}
```

---

## 4. 数据类型定义

### 4.1 邀请状态
| 值 | 说明 |
|-----|------|
| ACTIVE | 有效 |
| DISABLED | 已禁用 |
| EXPIRED | 已过期 |

### 4.2 好友来源
| 值 | 说明 |
|-----|------|
| INVITATION | 通过邀请链接 |
| SEARCH | 通过搜索 |
| MANUAL | 手动添加 |

---

## 5. 错误码汇总

| 错误码 | 说明 | HTTP状态码 |
|--------|------|-----------|
| 0 | 成功 | 200 |
| 1001 | 参数错误 | 400 |
| 1002 | 未授权 | 401 |
| 1003 | Token已过期 | 401 |
| 1004 | 资源不存在 | 404 |
| 1005 | 无权限 | 403 |
| 5000 | 服务器内部错误 | 500 |

---

## 6. 使用示例

### 6.1 创建邀请并分享

```typescript
// 1. 创建邀请链接
const createRes = await fetch('/api/v1/invitations', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    circleId: 'circle_uuid',
    expiresDays: 7
  })
});
const { data: invitation } = await createRes.json();

// 2. 获取海报图片
const posterUrl = `/api/v1/invitations/${invitation.id}/poster`;

// 3. 分享到微信
shareToWechat({
  title: '邀请你加入有空',
  imageUrl: posterUrl,
  link: invitation.inviteUrl
});
```

### 6.2 接受邀请

```typescript
// 1. 获取邀请详情
const infoRes = await fetch(`/api/v1/invitations/${inviteCode}`);
const { data: inviteInfo } = await infoRes.json();

if (!inviteInfo.isValid) {
  alert('邀请已失效');
  return;
}

// 2. 微信登录（带邀请码）
const loginRes = await fetch('/api/v1/auth/wechat/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    code: wechatCode,
    inviteCode: inviteCode
  })
});
const { data: loginResult } = await loginRes.json();

// 3. 登录成功，已自动成为好友并加入圈子
console.log('加入圈子:', loginResult.joinedCircle);
```
