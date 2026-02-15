package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

// SceneService AI 生成像素场景+互动选项
type SceneService struct {
	memoryRepo  *repository.MemoryRepository
	userRepo    *repository.UserRepository
	llmAdapter  *agent.LLMAdapter
	redisClient *tencent.RedisClient
}

// NewSceneService 创建场景服务
func NewSceneService(
	memoryRepo *repository.MemoryRepository,
	userRepo *repository.UserRepository,
	apiKey string,
	model string,
	redisClient *tencent.RedisClient,
) *SceneService {
	llmAdapter := agent.NewLLMAdapter(&agent.LLMAdapterConfig{
		APIKey: apiKey,
		Model:  model,
	})
	return &SceneService{
		memoryRepo:  memoryRepo,
		userRepo:    userRepo,
		llmAdapter:  llmAdapter,
		redisClient: redisClient,
	}
}

// GenerateScene 为用户的当前推断结果生成像素场景和互动选项
func (s *SceneService) GenerateScene(ctx context.Context, userID string, inference *model.CurrentStatusInference) (*model.SceneEnrichment, error) {
	// 1. 获取用户信息和画像
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	personaText := ""
	memory, err := s.memoryRepo.GetCoreMemory(ctx, userID)
	if err == nil && memory != nil {
		personaText = memory.PersonaText
	}

	nickname := user.Nickname
	if nickname == "" {
		nickname = "用户"
	}
	if personaText == "" {
		personaText = "暂无画像"
	}

	// 2. 构建 prompt
	prompt := fmt.Sprintf(`你是一个场景设计师。为用户的当前状态设计动画状态、像素场景和2个互动选项。

用户: %s
画像: %s
当前状态: %s %s
地点: %s
时间: %s

**1. rive_state（动画状态，从以下选一个最匹配的）:**
idle, walking, eating, sleeping, working, gaming, exercising, reading, drinking, cooking, driving, shopping, chatting, relaxing, running

映射规则:
- 吃饭/聚餐/火锅/烧烤等 → eating
- 走路/散步/逛街 → walking
- 跑步/慢跑 → running
- 睡觉/午睡/休息(躺) → sleeping
- 工作/开会/上班/办公 → working
- 玩手机/玩游戏/打游戏 → gaming
- 运动/健身/打球/瑜伽 → exercising
- 看书/学习/阅读 → reading
- 喝咖啡/喝茶/喝奶茶 → drinking
- 做饭/烹饪 → cooking
- 开车/坐车/坐地铁/通勤 → driving
- 购物/逛超市 → shopping
- 聊天/打电话/视频 → chatting
- 在家/看电视/刷手机/放松 → relaxing
- 其他/不确定 → idle

**2. 像素场景**（从以下选项中选择最合适的组合）:
- pose: standing/sitting/lying/running/walking/crouching
- arms: down/up/forward/holding/waving/typing
- expression: normal/happy/sleepy/focused/excited/tired
- prop: food/laptop/book/phone/weights/coffee/controller/ball/bag/umbrella/none
- surface: ground/chair/desk/bed/couch/none

**3. 互动选项**（基于这个人的性格和当前状态，设计2个有趣的互动）:
- 要有创意，不要通用的"打招呼"
- 要结合这个人的画像特点
- 互动是好友对这个人做的动作
- 每个互动: emoji + 按钮文字(2-5字) + 推送文案(一句话描述动作)

输出纯JSON，不要任何其他文字:
{"rive_state":"","scene":{"pose":"","arms":"","expression":"","prop":"","surface":""},"interactions":[{"emoji":"","label":"","push_text":""},{"emoji":"","label":"","push_text":""}]}`,
		nickname, personaText,
		inference.Emoji, inference.Activity,
		inference.Place, time.Now().Format("15:04"))

	// 3. 调用 LLM
	result, err := s.llmAdapter.Chat(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 4. 解析 JSON
	jsonStr := extractSceneJSON(result)
	var enrichment model.SceneEnrichment
	if err := json.Unmarshal([]byte(jsonStr), &enrichment); err != nil {
		return nil, fmt.Errorf("解析场景JSON失败: %w, raw=%s", err, result)
	}

	// 5. 验证+修正场景字段
	validateScene(&enrichment.Scene)
	validateRiveState(&enrichment)

	// 6. 确保有 2 个互动选项
	if len(enrichment.Interactions) < 2 {
		enrichment.Interactions = append(enrichment.Interactions, model.InteractionOption{
			Emoji:    "👋",
			Label:    "打招呼",
			PushText: fmt.Sprintf("向%s挥了挥手", nickname),
		})
	}
	if len(enrichment.Interactions) > 2 {
		enrichment.Interactions = enrichment.Interactions[:2]
	}

	// 7. 缓存到 Redis（和推断结果生命周期一致）
	s.cacheScene(ctx, userID, &enrichment)

	fmt.Printf("[场景] 生成完成 user=%s rive_state=%s pose=%s prop=%s interactions=%d\n",
		userID, enrichment.RiveState, enrichment.Scene.Pose, enrichment.Scene.Prop, len(enrichment.Interactions))

	return &enrichment, nil
}

// GetCachedScene 获取缓存的场景数据
func (s *SceneService) GetCachedScene(ctx context.Context, userID string) (*model.SceneEnrichment, error) {
	if s.redisClient == nil {
		return nil, fmt.Errorf("redis 未配置")
	}

	key := fmt.Sprintf("scene:%s", userID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if err != nil || data == nil {
		return nil, fmt.Errorf("无缓存")
	}

	var enrichment model.SceneEnrichment
	if err := json.Unmarshal(data, &enrichment); err != nil {
		return nil, err
	}
	return &enrichment, nil
}

// GetCachedScenes 批量获取缓存的场景数据
func (s *SceneService) GetCachedScenes(ctx context.Context, userIDs []string) map[string]*model.SceneEnrichment {
	result := make(map[string]*model.SceneEnrichment)
	if s.redisClient == nil {
		return result
	}

	for _, uid := range userIDs {
		if scene, err := s.GetCachedScene(ctx, uid); err == nil {
			result[uid] = scene
		}
	}
	return result
}

// cacheScene 缓存场景到 Redis
func (s *SceneService) cacheScene(ctx context.Context, userID string, enrichment *model.SceneEnrichment) {
	if s.redisClient == nil {
		return
	}

	key := fmt.Sprintf("scene:%s", userID)
	data, err := json.Marshal(enrichment)
	if err != nil {
		return
	}
	// 缓存 2 小时（和推断频率匹配）
	_ = s.redisClient.Set(ctx, key, data, 2*time.Hour)
}

// extractSceneJSON 从 LLM 输出中提取 JSON
func extractSceneJSON(text string) string {
	// 去除 markdown code block
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "```json"); idx != -1 {
		text = text[idx+7:]
	} else if idx := strings.Index(text, "```"); idx != -1 {
		text = text[idx+3:]
	}
	if idx := strings.LastIndex(text, "```"); idx != -1 {
		text = text[:idx]
	}
	text = strings.TrimSpace(text)

	// 找到第一个 { 和最后一个 }
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start != -1 && end != -1 && end > start {
		return text[start : end+1]
	}
	return text
}

// validateScene 验证并修正场景字段
func validateScene(scene *model.SceneConfig) {
	validPoses := map[string]bool{"standing": true, "sitting": true, "lying": true, "running": true, "walking": true, "crouching": true}
	validArms := map[string]bool{"down": true, "up": true, "forward": true, "holding": true, "waving": true, "typing": true}
	validExpressions := map[string]bool{"normal": true, "happy": true, "sleepy": true, "focused": true, "excited": true, "tired": true}
	validProps := map[string]bool{"food": true, "laptop": true, "book": true, "phone": true, "weights": true, "coffee": true, "controller": true, "ball": true, "bag": true, "umbrella": true, "none": true}
	validSurfaces := map[string]bool{"ground": true, "chair": true, "desk": true, "bed": true, "couch": true, "none": true}

	if !validPoses[scene.Pose] {
		scene.Pose = "standing"
	}
	if !validArms[scene.Arms] {
		scene.Arms = "down"
	}
	if !validExpressions[scene.Expression] {
		scene.Expression = "normal"
	}
	if !validProps[scene.Prop] {
		scene.Prop = "none"
	}
	if !validSurfaces[scene.Surface] {
		scene.Surface = "none"
	}
}

// validateRiveState 验证并修正 rive_state
func validateRiveState(enrichment *model.SceneEnrichment) {
	validStates := map[string]bool{
		"idle": true, "walking": true, "eating": true, "sleeping": true,
		"working": true, "gaming": true, "exercising": true, "reading": true,
		"drinking": true, "cooking": true, "driving": true, "shopping": true,
		"chatting": true, "relaxing": true, "running": true,
	}
	if !validStates[enrichment.RiveState] {
		enrichment.RiveState = "idle"
	}
}
