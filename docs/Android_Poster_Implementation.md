# Android 海报本地渲染实现文档

**完成时间**: 2026-02-01
**状态**: ✅ 编译通过，待测试

---

## 功能概述

实现了 Android 端的海报本地渲染和分享功能，与 iOS 保持一致的设计和布局。

---

## 新增文件

### PosterDialog.kt

**路径**: `Android/feature/feature-home/src/main/java/com/youkong/feature/home/screen/PosterDialog.kt`

**核心功能**:

#### 1. 海报渲染 (`renderPosterLocally()`)

使用原生 Android Canvas API 绘制海报：

```kotlin
fun renderPosterLocally(context: Context, friends: List<FriendGridItem>): Bitmap {
    val bitmap = Bitmap.createBitmap(750, 1334, Bitmap.Config.ARGB_8888)
    val canvas = Canvas(bitmap)

    // 1. 背景色 #F5F5F5
    // 2. Logo "有空" (48sp, 黑色, 居中)
    // 3. 时间戳 (16sp, 灰色, 居中)
    // 4. 宫格好友卡片
    // 5. Slogan "看透朋友此刻状态" (14sp, 灰色, 底部)

    return bitmap
}
```

**绘制元素**:
- ✅ 顶部 Logo + 时间戳
- ✅ 宫格卡片（200x200dp，圆角 12dp）
- ✅ 卡片内容：Emoji (40sp) + 昵称 (16sp) + 状态 (14sp)
- ✅ 底部 Slogan
- ⚠️ 二维码：未实现（可选）

#### 2. 好友卡片 (`drawFriendCard()`)

```kotlin
private fun drawFriendCard(canvas: Canvas, x: Float, y: Float, size: Float, friend: FriendGridItem) {
    // 白色背景 + 灰色边框
    canvas.drawRoundRect(rect, 12f, 12f, cardPaint)

    // Emoji 居中显示
    canvas.drawText(friend.emoji, x + size/2, y + 80f, emojiPaint)

    // 昵称（加粗）
    canvas.drawText(friend.nickname, x + size/2, y + 130f, namePaint)

    // 状态（灰色）
    canvas.drawText(friend.status, x + size/2, y + 160f, statusPaint)
}
```

#### 3. 分享功能 (`shareBitmap()`)

使用 `FileProvider` 分享图片：

```kotlin
private fun shareBitmap(context: Context, bitmap: Bitmap) {
    // 1. 保存到缓存目录 /cache/posters/
    val file = File(cachePath, "youkong_poster_${timestamp}.png")
    FileOutputStream(file).use { bitmap.compress(PNG, 100, it) }

    // 2. 获取 content:// URI
    val uri = FileProvider.getUriForFile(context, "${packageName}.fileprovider", file)

    // 3. 创建分享 Intent
    val shareIntent = Intent(ACTION_SEND).apply {
        type = "image/png"
        putExtra(EXTRA_STREAM, uri)
        addFlags(FLAG_GRANT_READ_URI_PERMISSION)
    }

    context.startActivity(Intent.createChooser(shareIntent, "分享海报"))
}
```

#### 4. 保存到相册 (`saveBitmap()`)

使用 `MediaStore` API (Android 10+):

```kotlin
private fun saveBitmap(context: Context, bitmap: Bitmap) {
    val values = ContentValues().apply {
        put(DISPLAY_NAME, "youkong_poster_${timestamp}.png")
        put(MIME_TYPE, "image/png")
        put(RELATIVE_PATH, "Pictures/YouKong")
    }

    val uri = contentResolver.insert(EXTERNAL_CONTENT_URI, values)
    contentResolver.openOutputStream(uri)?.use {
        bitmap.compress(PNG, 100, it)
    }
}
```

**优势**: Android 10+ 无需 `WRITE_EXTERNAL_STORAGE` 权限

---

## 修改文件

### 1. GridHomeScreen.kt

**集成 PosterDialog**:

```kotlin
@Composable
fun GridHomeScreen(...) {
    Scaffold(...) {
        // ... 宫格内容

        // 海报对话框
        if (uiState.showPosterDialog) {
            PosterDialog(
                friends = uiState.friends,
                onDismiss = { viewModel.hidePoster() }
            )
        }
    }
}
```

### 2. GridHomeViewModel.kt

**修复空值处理**:

```kotlin
fun loadGrid() {
    val response = homeApi.getGridData()
    _uiState.update {
        it.copy(
            friends = response.data!!.friends,  // !! 非空断言
            gridSize = response.data!!.gridSize
        )
    }
}
```

**说明**: 使用 `!!` 是因为如果 API 返回成功但 data 为 null，应该抛出异常而不是静默失败。

### 3. build.gradle.kts

**添加 core-network 依赖**:

```kotlin
dependencies {
    implementation(project(":core:core-network"))  // 新增
    implementation("com.google.accompanist:accompanist-swiperefresh:0.32.0")
}
```

**原因**: GridHomeViewModel 直接使用 `HomeApi`，需要引入 core-network 模块。

### 4. file_paths.xml

**添加 posters 缓存路径**:

```xml
<paths>
    <cache-path name="images" path="images/" />
    <cache-path name="posters" path="posters/" />  <!-- 新增 -->
</paths>
```

**用途**: FileProvider 需要声明可访问的路径。

---

## UI 交互流程

```
用户点击"分享"按钮
    ↓
GridHomeViewModel.showPoster()
    ↓
uiState.showPosterDialog = true
    ↓
PosterDialog 显示
    ↓
LaunchedEffect 触发
    ↓
renderPosterLocally() 生成 Bitmap
    ↓
显示海报预览
    ↓
用户操作：
- 点击"分享" → shareBitmap()
- 点击"保存到相册" → saveBitmap()
- 点击"关闭" → hidePoster()
```

---

## 技术细节

### 1. 宫格大小计算

```kotlin
private fun calculateGridSize(count: Int): Int {
    return when {
        count <= 1 -> 1  // 1x1
        count <= 4 -> 2  // 2x2
        count <= 9 -> 3  // 3x3
        else -> 4        // 4x4 (最多 16 个)
    }
}
```

### 2. 文字尺寸

使用 `scaledDensity` 实现 sp 单位：

```kotlin
val textSize = 16f * context.resources.displayMetrics.scaledDensity
```

**警告**: `scaledDensity` 已弃用，但仍可用。未来可能需要迁移到新 API。

### 3. 颜色定义

```kotlin
背景: #F5F5F5 (浅灰)
Logo: #000000 (黑色)
时间戳: #808080 (中灰)
卡片背景: #FFFFFF (白色)
卡片边框: #DCDCDC (淡灰)
Slogan: #646464 (深灰)
```

### 4. FileProvider 配置

**AndroidManifest.xml** (已存在):

```xml
<provider
    android:name="androidx.core.content.FileProvider"
    android:authorities="${applicationId}.fileprovider"
    android:exported="false"
    android:grantUriPermissions="true">
    <meta-data
        android:name="android.support.FILE_PROVIDER_PATHS"
        android:resource="@xml/file_paths" />
</provider>
```

---

## 编译结果

```bash
./gradlew :feature:feature-home:compileDebugKotlin
BUILD SUCCESSFUL in 4s
```

**警告**:
- `scaledDensity` 已弃用 (6 处，不影响功能)
- SwipeRefresh 迁移提示 (已知，非阻塞)

---

## 与 iOS 对比

| 功能 | iOS | Android | 状态 |
|------|-----|---------|------|
| 海报尺寸 | 750x1334 | 750x1334 | ✅ 一致 |
| Logo | UIGraphicsImageRenderer | Canvas | ✅ 一致 |
| 宫格布局 | 自动计算 | 自动计算 | ✅ 一致 |
| 时间戳 | DateFormatter | SimpleDateFormat | ✅ 一致 |
| 分享功能 | ShareLink | Intent.ACTION_SEND | ✅ 实现 |
| 保存相册 | UIImageWriteToSavedPhotosAlbum | MediaStore | ✅ 实现 |
| 二维码 | ❌ 未实现 | ❌ 未实现 | 🟡 可选 |
| 降级方案 | 后端 API → 本地 | 仅本地 | 🟢 已简化 |

---

## 测试步骤

### 前置条件

1. 确保有至少 2 个好友
2. 好友有状态数据（emoji、status）

### 测试步骤

1. **显示对话框**
   - 点击宫格首页"分享"按钮
   - 验证：对话框弹出，显示加载中

2. **海报生成**
   - 等待 1-2 秒
   - 验证：显示海报预览
   - 检查：Logo、时间、好友卡片、Slogan 是否正确

3. **分享功能**
   - 点击"分享"按钮
   - 验证：弹出系统分享菜单
   - 选择分享目标（如微信）
   - 验证：图片正确发送

4. **保存相册**
   - 点击"保存到相册"按钮
   - 打开相册 App
   - 验证：在 `Pictures/YouKong/` 目录找到图片

5. **关闭对话框**
   - 点击"关闭"按钮
   - 验证：对话框消失，回到宫格首页

### 边界情况

- 只有 1 个好友（自己）
- 9 个好友（3x3 满宫格）
- 超过 16 个好友（只显示前 16 个）
- 好友昵称过长（是否截断）
- 网络断开（应该仍能生成海报）

---

## 已知问题

### 1. scaledDensity 弃用警告

**影响**: 无（仍可正常使用）

**解决方案**: 未来迁移到推荐的 API（如 `TypedValue.applyDimension()`）

### 2. TODO 注释

```kotlin
// TODO: 显示保存成功提示
// TODO: 显示保存失败提示
```

**影响**: 保存成功/失败没有 UI 反馈

**解决方案**: 添加 SnackBar 或 Toast 提示

### 3. 二维码未实现

**影响**: 海报底部缺少邀请二维码

**解决方案**: 可选功能，如需实现可使用 `zxing` 库

---

## 优化建议

### 高优先级

- [ ] 添加保存成功/失败 Toast 提示
- [ ] 处理存储权限拒绝情况（Android < 10）

### 中优先级

- [ ] 添加海报样式自定义（背景色、字体大小）
- [ ] 支持邀请二维码生成
- [ ] 优化文字排版（自动换行、居中对齐）

### 低优先级

- [ ] 添加水印（Logo 半透明）
- [ ] 支持多种海报模板
- [ ] 添加生成动画效果

---

## 代码统计

**新增文件**: 1 个 (PosterDialog.kt, 364 行)
**修改文件**: 4 个
**总变更**: +364 行, -2 行

---

## 提交记录

```bash
commit a664612
Author: Claude Sonnet 4.5
Date: Sat Feb 1 21:03:52 2026

feat: 实现 Android 海报本地渲染和分享功能

新增:
- PosterDialog.kt - 海报分享对话框
  - 使用 Canvas 本地渲染海报
  - 支持分享到其他应用（FileProvider）
  - 支持保存到相册（MediaStore API）
  - 750x1334 尺寸，与 iOS 一致

修改:
- GridHomeScreen.kt - 集成 PosterDialog
- GridHomeViewModel.kt - 修复 API 响应空值处理
- build.gradle.kts - 添加 core-network 依赖
- file_paths.xml - 添加 posters 缓存路径
```

---

**完成时间**: 2026-02-01 21:05
**执行人**: Claude Code
**状态**: ✅ 编译通过，等待真机测试
