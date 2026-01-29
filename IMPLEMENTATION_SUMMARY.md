# 大模型推理系统改进实施总结

## 改进目标

将大模型推理从**过度约束**改为**完全自由推理**，让描述更自然、更真实、更有信息量。

### 核心变化

- ❌ 移除字数限制（10字/15字 → 20-40字）
- ❌ 移除隐私过度保护（允许"摸鱼"、"刷手机"等自然表达）
- ✅ 大模型完全自由生成状态描述
- ✅ iOS 前端展示完整的自然语言描述

---

## 已完成的改进

### 1. 后端 - 规则引擎版本 (GenerateFreeReason)

**文件**: `Backend/internal/pkg/llm/openrouter.go` (行 327-363)

**改动**:
```diff
- 根据以下脱敏后的状态信息，生成一句简短的推测性描述（10字以内）
+ 根据以下状态信息，生成一句自然的口语化描述

- 3. 绝对不能提及具体APP名称或具体行为
- 4. 不能说"刷手机"、"玩手机"等暴露隐私的表述
+ 2. 可以使用"在摸鱼"、"在刷手机"、"在看剧"等真实描述
+ 3. 不限制长度，表达清楚即可（建议20-30字）

- "可能有空"
- "应该在忙"
+ "看起来在公司摸鱼刷手机"
+ "可能在家躺着追剧"
+ "深夜了还在赶项目，估计没空"
```

### 2. 后端 - 福尔摩斯版本 (HolmesAnalyzer)

**文件**: `Backend/internal/pkg/llm/holmes_analyzer.go` (行 354-431)

**改动**:
```diff
- "reasoning": "你的推理过程（50-100字，像侦探一样分析）"
+ "reasoning": "你的完整推理过程（不限长度，详细分析）"

- "summary": "在咖啡厅休闲"
+ "summary": "看起来在咖啡厅边喝咖啡边刷手机摸鱼，应该有空约"

+ ## 输出要求
+ - summary: 用自然口语化描述当前状态（20-40字），可以使用"摸鱼"、"刷手机"、"追剧"等真实表达
+ - emoji: 可以自由选择任何 Unicode emoji，不限于预设列表
+ - reasoning: 详细的推理过程，不限长度

+ ## Emoji 建议（可自由选择任何 Unicode emoji）
+ 🎮 游戏 | 📺 追剧 | 💼 工作 | ☕ 咖啡厅/摸鱼
+ ...
+ 🏋️ 健身 | 🍕 美食 | 🎉 娱乐 | 😊 休闲
+
+ 你也可以根据具体情况选择其他更合适的 emoji。
```

### 3. iOS 前端 - 好友列表展示

**文件**: `iOS/YouKong/Presentation/Screens/FriendsList/FriendsListView.swift` (行 145-187)

**改动**:
```diff
- HStack(spacing: 0) {
+ VStack(alignment: .leading, spacing: 6) {
+     HStack(spacing: 0) {
-         // 状态指示符
+         // 状态指示符 + Emoji
+         HStack(spacing: 4) {
              Text(CLIConstants.statusSymbol(for: friend.probability))
+             if let emoji = friend.emoji {
+                 Text(emoji)
+             }
+         }

-         // 状态文字
-         Text(statusText(for: friend.probability))
+     }
+
+     // 自然语言描述
+     if friend.hasData {
+         Text("> \(friend.displayActivity)")
+             .font(.system(size: 13, design: .monospaced))
+             .foregroundColor(.gray)
+             .lineLimit(2)  // 支持多行
+             .padding(.leading, 44)
+     }
+ }
```

**移除**: `statusText(for:)` 函数（不再需要固定文字映射）

---

## 数据模型验证

### FriendRecommendation (iOS)

**文件**: `iOS/YouKong/Domain/Entities/FriendRecommendation.swift`

**已有字段** ✅:
```swift
struct FriendRecommendation {
    let emoji: String?      // ✅ 活动 emoji
    let activity: String?   // ✅ 活动描述（自然语言）
    let reason: String      // ✅ 有空程度状态

    var displayActivity: String {
        activity ?? reason  // ✅ 优先使用 activity，fallback 到 reason
    }
}
```

**无需修改** - 数据模型已经完美支持！

---

## 预期效果对比

### 改进前

**后端生成**:
```json
{
  "reason": "可能有空",
  "probability": 75,
  "emoji": "📱"
}
```

**iOS 展示**:
```
[●] 小明                75%  有空
```

### 改进后

**后端生成**:
```json
{
  "reason": "看起来在咖啡厅边喝咖啡边刷手机摸鱼，应该有空约",
  "probability": 75,
  "emoji": "☕"
}
```

**iOS 展示**:
```
[●] ☕ 小明              75%
    > 看起来在咖啡厅边喝咖啡边刷手机摸鱼，
      应该有空约
```

---

## 验证方式

### 后端验证

1. **启动后端服务**:
```bash
cd Backend
make dev
```

2. **测试规则引擎版本**:
```bash
curl -X GET "http://localhost:8080/api/v1/friends/free-probability" \
  -H "Authorization: Bearer {token}"
```

**预期结果**:
```json
{
  "friends": [{
    "reason": "看起来在公司摸鱼刷手机",  // ✅ 超过10字
    "probability": 65,
    "emoji": "📱"
  }]
}
```

3. **测试福尔摩斯版本**:
```bash
curl -X GET "http://localhost:8080/api/v1/agent/status" \
  -H "Authorization: Bearer {token}" \
  -X POST \
  -d '{"screen": {...}, "location": {...}}'
```

**预期结果**:
```json
{
  "reasoning": "根据线索，此人在周五晚上22:30，位于家中，屏幕显示娱乐内容已使用1小时23分钟...",
  "summary": "周五晚上在家躺着追剧，应该挺有空的",  // ✅ 超过15字
  "probability": 82,
  "confidence": "high",
  "emoji": "📺"  // ✅ 可以是任何emoji
}
```

### iOS 验证

1. **构建并运行**:
```bash
cd iOS
xcodebuild -scheme YouKong -configuration Debug build
# 或在 Xcode 中直接运行
```

2. **打开好友列表**:
   - 导航到好友列表页面
   - 下拉刷新数据

3. **预期展示**:
```
┌─────────────────────────────────────┐
│ [●] ☕ 小明              75%         │
│     > 看起来在咖啡厅边喝咖啡边刷手    │
│       机摸鱼，应该有空约               │
│                                       │
│ [◐] 📺 小红              45%         │
│     > 周五晚上在家躺着追剧，应该      │
│       挺有空的                         │
│                                       │
│ [○] 💼 小刚              15%         │
│     > 深夜了还在公司加班赶项目，      │
│       估计没空                         │
└─────────────────────────────────────┘
```

4. **验证点**:
   - ✅ 展示 emoji
   - ✅ 展示完整的自然语言描述（20-40字）
   - ✅ 支持多行展示（最多2行）
   - ✅ CLI 风格的 "> " 前缀
   - ✅ 描述中包含"摸鱼"、"刷手机"、"追剧"等真实词汇

---

## 隐私与用户体验平衡

### 保留的隐私保护机制

1. **客户端脱敏**:
   - 只发送活动类型（entertainment/productivity），不发送具体APP名称
   - 只发送位置类型（home/work），不发送精确GPS坐标

2. **推测性表达**:
   - 使用"看起来"、"可能"、"应该"等词，表明是推测而非监控

3. **用户控制**:
   - 用户可以随时停止状态上报
   - 用户可以选择不展示自己的状态

### 为什么允许"摸鱼"、"刷手机"

| 场景 | 改进前 | 改进后 |
|------|--------|--------|
| 工作日晚上在家 | "可能有空" | "晚上在家刷手机摸鱼，应该有空" |
| 周末咖啡厅 | "应该在休息" | "周末在咖啡厅边喝咖啡边看书，挺悠闲" |
| 深夜办公室 | "可能在忙" | "深夜了还在公司赶项目，估计没空" |

**好处**:
- ✅ 更真实、更有趣的状态描述
- ✅ 帮助朋友更好地判断是否打扰
- ✅ 保持趣味性和可读性
- ✅ 仍然是推测性表达，非隐私暴露

---

## 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 描述过长影响UI | 低 | 限制最多2行展示，超出省略 ✅ |
| 大模型生成不当内容 | 低 | 保留基本约束（推测性表达） ✅ |
| 用户认为隐私泄露 | 中 | 文档说明"推测"而非"监控" |
| iOS UI布局问题 | 低 | 使用VStack + lineLimit，兼容性好 ✅ |

---

## 回滚方案

如果用户反馈不佳，可以快速回滚：

1. **后端**: 恢复旧的 prompt（10字限制）
2. **iOS**: 移除 displayActivity 展示，恢复固定文字

所有改动都在 prompt 层面，无需修改模型或数据库。

---

## Android 扩展（未实现）

### 数据模型准备

**文件**: `Android/core/core-domain/src/main/java/com/youkong/core/domain/model/FriendRecommendation.kt`

```kotlin
data class FriendRecommendation(
    val friendId: String,
    val name: String,
    val avatar: String?,
    val probability: Int,
    val confidence: Confidence,
    val reason: String,
    val color: String,
    val emoji: String?,      // ✅ 已有
    val activity: String?,   // ✅ 已有
    val updatedAt: Long,
) {
    val hasData: Boolean get() = probability >= 0
    val displayActivity: String get() = activity ?: reason
}
```

### 好友列表界面（参考）

```kotlin
@Composable
fun FriendRow(friend: FriendRecommendation) {
    Column(modifier = Modifier.fillMaxWidth().padding(16.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            friend.emoji?.let { Text(text = it, fontSize = 16.sp) }
            Text(text = friend.name, fontSize = 15.sp, modifier = Modifier.weight(1f))
            Text(text = "${friend.probability}%", fontSize = 13.sp)
        }

        if (friend.hasData) {
            Text(
                text = "> ${friend.displayActivity}",
                fontSize = 13.sp,
                color = Color.Gray,
                maxLines = 2,
                modifier = Modifier.padding(start = 24.dp, top = 4.dp)
            )
        }
    }
}
```

---

## 实施总结

### 改动文件

| 文件 | 改动类型 | 行数 |
|------|---------|------|
| `Backend/internal/pkg/llm/openrouter.go` | 修改 Prompt | 329-363 |
| `Backend/internal/pkg/llm/holmes_analyzer.go` | 修改 Prompt | 401-431 |
| `iOS/.../FriendsListView.swift` | UI 重构 | 145-187 |

### 总改动量

- **后端**: 2 个文件，~60 行
- **iOS**: 1 个文件，~40 行
- **Android**: 0 个文件（预留扩展）

### 测试状态

- ✅ 后端 Prompt 已修改
- ✅ iOS UI 已重构
- ⏳ 等待端到端测试
- ⏳ 等待用户反馈

---

## 下一步行动

1. **启动后端服务**并测试生成效果
2. **在 iOS 模拟器/真机**中验证展示效果
3. **上报真实状态数据**，观察推理质量
4. **收集用户反馈**，必要时微调 Prompt
5. **Android 实现**（当功能开发到好友列表时）

---

## 联系与支持

如有问题或需要调整，请参考：
- 计划文档: `CLI_TRANSFORMATION_PROGRESS.md`
- API 文档: `Backend/API.md`
- 福尔摩斯框架: `docs/Holmes_Agent_Guide.md`
