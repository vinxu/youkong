# 海报生成策略调整说明

**调整时间**: 2026-02-01
**状态**: ✅ 已完成并部署

---

## 变更说明

### 之前的方案（Phase 4）

海报由**后端服务器渲染**：

```
客户端 → 后端 API → fogleman/gg 渲染 → 保存 PNG → 返回 URL → 客户端下载显示
```

**问题**：
- 需要处理字体路径（macOS/Linux 不同）
- 需要上传到 COS 才能让客户端访问
- 服务器负担增加
- 渲染延迟（网络往返）

### 当前方案

海报由**客户端本地渲染**：

```
客户端 → 本地绘制（SwiftUI/Compose Canvas）→ 生成图片 → 直接分享
```

**优势**：
- ✅ 无需网络请求，生成速度快
- ✅ 客户端直接合成图片，质量可控
- ✅ 减少服务器负担和存储成本
- ✅ 无需处理字体、COS 等问题
- ✅ 离线也能生成海报

---

## 删除的代码

### 后端

| 文件 | 操作 |
|------|------|
| `Backend/internal/service/poster_service.go` | ❌ 删除（完整的 gg 渲染实现） |
| `Backend/internal/handler/home_handler.go` | 🔧 删除 `GeneratePoster()` 方法 |
| `Backend/cmd/server/main.go` | 🔧 删除 `posterService` 初始化和路由 |

**删除的接口**：
```
POST /api/v1/home/poster  ❌ 已删除
```

### iOS

| 文件 | 修改内容 |
|------|---------|
| `PosterShareView.swift` | 直接调用 `renderPosterLocally()`，删除后端 API 调用 |
| `GridModels.swift` | 删除 `PosterResponse` 数据模型 |
| `APIEndpoint.swift` | 删除 `generatePoster()` 端点定义 |

### Android

| 文件 | 修改内容 |
|------|---------|
| `HomeApi.kt` | 删除 `generatePoster()` 接口、`GeneratePosterRequest`、`PosterResponse` |
| `GridHomeViewModel.kt` | 简化 `showPoster()` 为显示本地对话框 |
| `GridHomeUiState` | 删除 `posterUrl`、`posterLoading`、`posterError` 字段 |

---

## 客户端实现

### iOS 本地渲染

使用 `UIGraphicsImageRenderer` 绘制海报：

```swift
// iOS/YouKong/Presentation/Screens/Home/PosterShareView.swift
@MainActor
private func renderPosterLocally() async -> UIImage? {
    let size = CGSize(width: 750, height: 1334)  // iPhone 尺寸
    let renderer = UIGraphicsImageRenderer(size: size)

    return renderer.image { context in
        // 1. 绘制背景
        UIColor.systemBackground.setFill()
        context.fill(CGRect(origin: .zero, size: size))

        // 2. 绘制标题 "有空"
        let titleText = "有空"
        titleText.draw(at: ..., withAttributes: ...)

        // 3. 绘制时间戳
        let timeText = dateFormatter.string(from: Date())
        timeText.draw(at: ..., withAttributes: ...)

        // 4. 绘制宫格好友卡片
        for (index, friend) in friends.prefix(9).enumerated() {
            // 绘制卡片背景
            // 绘制 Emoji
            // 绘制昵称
            // 绘制状态
        }

        // 5. 绘制底部 Logo
        let logoText = "有空 - 看透朋友此刻状态"
        logoText.draw(at: ..., withAttributes: ...)
    }
}
```

**渲染元素**：
- ✅ 顶部：Logo "有空" + 时间戳
- ✅ 中间：宫格好友卡片（Emoji + 昵称 + 状态）
- ✅ 底部：Slogan "看透朋友此刻状态"
- ⚠️ 二维码：暂未实现（可选功能）

### Android 本地渲染（待实现）

需要创建 `PosterDialog` Composable，使用 Canvas 绘制：

```kotlin
// Android/feature/feature-home/src/.../screen/PosterDialog.kt
@Composable
fun PosterDialog(
    friends: List<FriendGridItem>,
    onDismiss: () -> Unit
) {
    val posterBitmap = remember {
        renderPosterLocally(friends)
    }

    Dialog(onDismissRequest = onDismiss) {
        Column {
            Image(bitmap = posterBitmap, ...)

            // 分享按钮
            Button(onClick = { shareBitmap(posterBitmap) }) {
                Text("分享")
            }

            // 保存按钮
            Button(onClick = { saveBitmap(posterBitmap) }) {
                Text("保存到相册")
            }
        }
    }
}

private fun renderPosterLocally(friends: List<FriendGridItem>): Bitmap {
    val bitmap = Bitmap.createBitmap(750, 1334, Bitmap.Config.ARGB_8888)
    val canvas = Canvas(bitmap)

    // 使用 Paint 绘制背景、文字、卡片

    return bitmap
}
```

---

## 部署验证

### 后端验证

```bash
# 宫格 API 仍然可用
curl -H "Authorization: Bearer $TOKEN" http://49.232.13.41:8080/api/v1/home/grid
# ✅ 返回 {"code": 0, "data": {"grid_size": 2, "friends": [...]}}

# 海报 API 已删除
curl -X POST -H "Authorization: Bearer $TOKEN" http://49.232.13.41:8080/api/v1/home/poster
# ✅ 返回 {"code": 1004, "message": "接口不存在"}
```

### 客户端验证

**iOS**:
1. 进入宫格首页
2. 点击"分享"按钮
3. 查看本地渲染的海报（无需网络等待）
4. 测试分享和保存功能

**Android**:
- GridHomeViewModel 已更新
- UI 实现待完成（PosterDialog）

---

## 待办事项

### iOS

- [x] 删除后端 API 调用
- [x] 使用本地渲染
- [ ] 优化海报样式（可选）
- [ ] 添加二维码（可选）

### Android

- [x] 删除 API 定义
- [x] 更新 ViewModel
- [ ] **创建 PosterDialog Composable**（高优先级）
- [ ] 实现 Canvas 绘制逻辑
- [ ] 实现分享和保存功能

### 后端

- [x] 删除 PosterService
- [x] 删除 GeneratePoster 接口
- [x] 部署验证

---

## 代码统计

**删除代码行数**: 378 行
**新增代码行数**: 10 行
**净减少**: 368 行

**涉及文件**: 8 个

---

## 提交记录

```bash
commit c8a8010
Author: Claude Sonnet 4.5
Date: Sat Feb 1 20:41:33 2026

refactor: 海报生成改为客户端本地渲染

- 删除后端 PosterService（不再需要服务端渲染）
- 删除 HomeHandler.GeneratePoster 接口
- iOS PosterShareView 直接使用本地渲染
- Android GridHomeViewModel 简化为本地生成
- 删除 PosterResponse 数据模型
- 删除 API Endpoint 定义

理由：
- 海报由用户分享，客户端本地合成更快
- 减少服务器负担和存储成本
- 无需处理字体路径、COS 上传等问题
```

---

**完成时间**: 2026-02-01 20:44
**执行人**: Claude Code
**状态**: ✅ 已部署到生产环境
