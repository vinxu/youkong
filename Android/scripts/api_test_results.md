# YouKong API 测试结果

**测试时间**: 2026-01-25
**API 地址**: http://49.232.13.41:8080/api/v1
**测试账号**: 13800000001 / 111111

## 测试结果汇总

| # | 接口 | 方法 | 状态 | 说明 |
|---|------|------|------|------|
| 1 | `/auth/sms/verify` | POST | ✅ 通过 | 验证码登录，返回 Token |
| 2 | `/auth/refresh` | POST | ✅ 通过 | Token 刷新 |
| 3 | `/users/me` | GET | ✅ 通过 | 获取当前用户信息 |
| 4 | `/users/me` | PUT | ✅ 通过 | 更新用户信息 |
| 5 | `/users/search` | GET | ✅ 通过 | 搜索用户 |
| 6 | `/circles` | GET | ✅ 通过 | 获取圈子列表 |
| 7 | `/circles` | POST | ✅ 通过 | 创建圈子 |
| 8 | `/circles/:id` | GET | ✅ 通过 | 获取圈子详情 |
| 9 | `/availabilities/friends` | GET | ✅ 通过 | 获取好友有空 |
| 10 | `/availabilities/mine` | GET | ✅ 通过 | 获取我的有空 |
| 11 | `/availabilities` | POST | ✅ 通过 | 发布有空 |
| 12 | `/availabilities/:id` | DELETE | ✅ 通过 | 取消有空 |
| 13 | `/conversations` | GET | ✅ 通过 | 获取会话列表 |

## 详细测试记录

### 1. 验证码登录
```json
POST /auth/sms/verify
Request: {"phone":"13800000001","code":"111111"}
Response: {
  "code": 0,
  "message": "成功",
  "data": {
    "token": "eyJhbGc...",
    "user": {
      "id": "baec1253-58a7-4716-8c8a-48710a63c674",
      "phone": "13800000001",
      "nickname": "用户0001"
    },
    "isNewUser": false
  }
}
```

### 2. 创建圈子
```json
POST /circles
Request: {"name":"好朋友","emoji":"🎉","color":"#EC4899"}
Response: {
  "code": 0,
  "data": {
    "id": "405b1b2f-09c7-4b3d-9f1b-d43bffbc082a",
    "name": "好朋友",
    "emoji": "🎉",
    "color": "#EC4899",
    "ownerId": "baec1253-58a7-4716-8c8a-48710a63c674"
  }
}
```

### 3. 发布有空
```json
POST /availabilities
Request: {
  "startTime": "2026-01-26T14:00:00+08:00",
  "endTime": "2026-01-26T18:00:00+08:00",
  "locationType": "FLEXIBLE",
  "circleIds": ["405b1b2f-09c7-4b3d-9f1b-d43bffbc082a"]
}
Response: {
  "code": 0,
  "data": {
    "id": "62cac6e8-be69-41a1-92b0-08db71e21d40",
    "user": {"id": "...", "nickname": "用户0001"},
    "startTime": "2026-01-26T14:00:00+08:00",
    "endTime": "2026-01-26T18:00:00+08:00",
    "location": {"type": "FLEXIBLE"},
    "status": "ACTIVE",
    "circleIds": ["405b1b2f-09c7-4b3d-9f1b-d43bffbc082a"]
  }
}
```

### 4. 搜索用户
```json
GET /users/search?keyword=用户
Response: {
  "code": 0,
  "data": [
    {"id": "baec1253-...", "nickname": "用户0001"},
    {"id": "dd200480-...", "nickname": "用户0002"}
  ]
}
```

## Android 端数据模型兼容性

| 字段 | 后端返回 | Android 定义 | 状态 |
|------|----------|--------------|------|
| user.lastActiveAt | `{"Time": "...", "Valid": true}` | 未定义 | ⚠️ 需忽略 |
| circle.updatedAt | 返回 | 未定义 | ⚠️ 需忽略 |

**注意**: 已配置 `ignoreUnknownKeys = true`，未定义字段会被自动忽略，不影响解析。

## 测试账号

| 手机号 | 验证码 | 说明 |
|--------|--------|------|
| 13800000001 | 111111 | 测试账号1 |
| 13800000002 | 222222 | 测试账号2 |
| 13800000003 | 333333 | 测试账号3 |
| 13800138000 | 123456 | 测试账号4 |
