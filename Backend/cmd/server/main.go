package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"youkong/internal/config"
	"youkong/internal/handler"
	"youkong/internal/job"
	"youkong/internal/middleware"
	"youkong/internal/pkg/asr"
	"youkong/internal/pkg/jwt"
	"youkong/internal/pkg/llm"
	"youkong/internal/pkg/poster"
	"youkong/internal/pkg/push"
	"youkong/internal/pkg/tencent"
	"youkong/internal/pkg/wechat"
	"youkong/internal/pkg/ws"
	"youkong/internal/repository"
	"youkong/internal/service"
)

// 版本信息（通过 ldflags 注入）
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	var logger *zap.Logger
	if cfg.Server.Mode == "debug" {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync()

	// 连接数据库
	db, err := sqlx.Connect("mysql", cfg.DB.DSN())
	if err != nil {
		logger.Fatal("连接数据库失败", zap.Error(err))
	}
	defer db.Close()
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	logger.Info("数据库连接成功")

	// 连接Redis
	redisClient, err := tencent.NewRedisClient(cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		logger.Fatal("连接Redis失败", zap.Error(err))
	}
	defer redisClient.Close()
	logger.Info("Redis连接成功")

	// 初始化腾讯云SMS客户端（支持 STS 临时密钥）
	var smsClient *tencent.SMSClient
	if cfg.Tencent.Token != "" {
		// 使用 STS 临时密钥
		smsClient, err = tencent.NewSMSClientWithToken(
			cfg.Tencent.SecretID,
			cfg.Tencent.SecretKey,
			cfg.Tencent.Token,
			cfg.Tencent.SMSAppID,
			cfg.Tencent.SMSSignName,
			cfg.Tencent.SMSTemplateID,
		)
		if err != nil {
			logger.Warn("初始化SMS客户端失败（STS模式），短信功能将不可用", zap.Error(err))
		} else {
			logger.Info("SMS客户端初始化成功（使用STS临时密钥）")
		}
	} else {
		// 使用永久密钥
		smsClient, err = tencent.NewSMSClient(
			cfg.Tencent.SecretID,
			cfg.Tencent.SecretKey,
			cfg.Tencent.SMSAppID,
			cfg.Tencent.SMSSignName,
			cfg.Tencent.SMSTemplateID,
		)
		if err != nil {
			logger.Warn("初始化SMS客户端失败，短信功能将不可用", zap.Error(err))
		} else {
			logger.Info("SMS客户端初始化成功（使用永久密钥）")
		}
	}

	// 初始化JWT管理器
	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.ExpireHours)

	// 初始化 阿里云 ASR 客户端
	var asrClient *asr.AliyunASRClient
	if cfg.AliyunASR.AccessKeyID != "" && cfg.AliyunASR.AccessKeySecret != "" {
		asrClient = asr.NewAliyunASRClient(
			cfg.AliyunASR.AccessKeyID,
			cfg.AliyunASR.AccessKeySecret,
			cfg.AliyunASR.AppKey,
		)
		logger.Info("阿里云 ASR 客户端初始化成功")
	} else {
		logger.Warn("阿里云 ASR 配置未设置，语音识别将使用模拟模式")
	}

	// 初始化Repository
	userRepo := repository.NewUserRepository(db)
	circleRepo := repository.NewCircleRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	wechatRepo := repository.NewWechatRepository(db)
	invitationRepo := repository.NewInvitationRepository(db)
	friendshipRepo := repository.NewFriendshipRepository(db)
	friendRequestRepo := repository.NewFriendRequestRepository(db)
	memoryRepo := repository.NewMemoryRepository(db)
	memoryDocRepo := repository.NewUserMemoryDocumentRepository(db)
	deviceTokenRepo := repository.NewDeviceTokenRepository(db)
	userProfileRepo := repository.NewUserProfileRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)
	userSettingsRepo := repository.NewUserSettingsRepository(db)
	predictionRepo := repository.NewPredictionRepository(db)

	// 初始化微信客户端
	var wechatClient *wechat.Client
	if cfg.Wechat.AppID != "" && cfg.Wechat.AppSecret != "" {
		wechatClient = wechat.NewClient(cfg.Wechat.AppID, cfg.Wechat.AppSecret)
		logger.Info("微信客户端初始化成功")
	} else {
		logger.Warn("微信配置未设置，微信登录功能将不可用")
	}

	// 初始化海报生成器
	posterGenerator := poster.NewGenerator(cfg.Invitation.BaseURL)

	// 初始化 WebSocket 管理器
	wsManager := ws.NewManager()

	// 初始化推送客户端
	var pushManager *push.Manager
	{
		// APNs 客户端 (iOS)
		apnsClient, err := push.NewAPNsClient(cfg.APNs)
		if err != nil {
			logger.Warn("初始化 APNs 客户端失败", zap.Error(err))
		}

		// TPNS 客户端 (Android)
		tpnsClient := push.NewTPNSClient(cfg.TPNS)

		pushManager = push.NewManager(apnsClient, tpnsClient)
		if pushManager.IsEnabled() {
			logger.Info("推送服务初始化成功")
		} else {
			logger.Warn("推送服务未启用（缺少配置）")
		}
	}

	// 初始化 NotificationService（需要在 conversationService 之前）
	notificationService := service.NewNotificationService(deviceTokenRepo, pushManager, wsManager)

	// 初始化 LLM 客户端
	var llmClient *llm.OpenRouterClient
	if cfg.LLM.APIKey != "" {
		llmClient = llm.NewOpenRouterClient(cfg.LLM.APIKey, cfg.LLM.Model)
		logger.Info("LLM 客户端初始化成功", zap.String("model", cfg.LLM.Model))
	} else {
		logger.Warn("LLM_API_KEY 未配置，将使用默认理由生成")
	}

	// 初始化Service
	smsService := service.NewSMSService(smsClient, redisClient)
	authService := service.NewAuthService(userRepo, smsService, jwtManager)
	userService := service.NewUserService(userRepo)
	circleService := service.NewCircleService(circleRepo, userRepo)
	conversationService := service.NewConversationService(messageRepo, userRepo, notificationService, wsManager)
	wechatService := service.NewWechatService(wechatRepo, userRepo, invitationRepo, friendshipRepo, circleRepo, wechatClient, jwtManager)
	invitationService := service.NewInvitationService(invitationRepo, circleRepo, userRepo, friendshipRepo, cfg.Invitation.BaseURL)
	friendshipService := service.NewFriendshipService(friendshipRepo, userRepo, invitationRepo, circleRepo, friendRequestRepo)
	userProfileService := service.NewUserProfileService(userProfileRepo)
	agentService := service.NewAgentService(redisClient, userRepo, friendshipRepo, memoryRepo, userProfileService, llmClient)
	memoryService := service.NewMemoryService(memoryRepo, redisClient, llmClient)
	contactService := service.NewContactService(userRepo, friendshipRepo)
	homeService := service.NewHomeService(friendshipRepo, userRepo, memoryRepo, redisClient)
	voiceScheduleService := service.NewVoiceScheduleService(scheduleRepo, memoryRepo, userProfileService, redisClient, asrClient, llmClient, cfg.LLM.APIKey)
	voiceScheduleServiceV4 := service.NewVoiceScheduleServiceV4(scheduleRepo, memoryRepo, memoryDocRepo, userProfileService, redisClient, asrClient, cfg.LLM.APIKey)
	predictionService := service.NewPredictionService(predictionRepo, scheduleRepo, memoryRepo, userProfileService, llmClient)

	// 初始化 Agent Chat Service（Tool Agent 框架）
	var agentChatService *service.AgentChatService
	if cfg.LLM.APIKey != "" {
		agentChatService = service.NewAgentChatService(
			cfg.LLM.APIKey,
			cfg.LLM.Model,
			redisClient,
			userRepo,
			friendshipRepo,
			memoryRepo,
			scheduleRepo,
			agentService,
			memoryService,
			asrClient, // 语音识别客户端
		)
		logger.Info("Agent Chat Service 初始化成功（Tool Agent 框架）")
	} else {
		logger.Warn("LLM_API_KEY 未配置，Agent Chat 功能将不可用")
	}

	// 初始化Handler
	authHandler := handler.NewAuthHandler(authService, wechatService)
	userHandler := handler.NewUserHandler(userService, posterGenerator, cfg.Invitation.BaseURL, messageRepo, userSettingsRepo)
	circleHandler := handler.NewCircleHandler(circleService)
	conversationHandler := handler.NewConversationHandler(conversationService)
	invitationHandler := handler.NewInvitationHandler(invitationService, posterGenerator)
	friendshipHandler := handler.NewFriendshipHandler(friendshipService)
	agentHandler := handler.NewAgentHandler(agentService, memoryService, voiceScheduleService, agentChatService, scheduleRepo)
	agentHandler.SetVoiceScheduleServiceV4(voiceScheduleServiceV4) // 设置 V4 服务

	// 初始化模型测试服务（用于 Qwen vs Kimi vs Claude 对比测试）
	if cfg.LLM.APIKey != "" || cfg.LLM.KimiAPIKey != "" || cfg.LLM.ClaudeAPIKey != "" {
		modelTestService := service.NewModelTestService(cfg.LLM.APIKey, cfg.LLM.KimiAPIKey, cfg.LLM.ClaudeAPIKey)
		agentHandler.SetModelTestService(modelTestService)
		logger.Info("模型测试服务初始化成功",
			zap.Bool("qwen_enabled", cfg.LLM.APIKey != ""),
			zap.Bool("kimi_enabled", cfg.LLM.KimiAPIKey != ""),
			zap.Bool("claude_enabled", cfg.LLM.ClaudeAPIKey != ""))
	}
	contactHandler := handler.NewContactHandler(contactService)
	deployHandler := handler.NewDeployHandler(&cfg.Deploy, logger)
	wsHandler := handler.NewWSHandler(wsManager, jwtManager)
	deviceHandler := handler.NewDeviceHandler(notificationService)
	homeHandler := handler.NewHomeHandler(homeService)
	userProfileHandler := handler.NewUserProfileHandler(userProfileService)
	predictionHandler := handler.NewPredictionHandler(predictionService)

	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	// 创建路由
	r := gin.New()
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.CORS())
	r.Use(middleware.Logger(logger))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"version":   Version,
			"commit":    Commit,
			"buildTime": BuildTime,
		})
	})

	// 部署 webhook
	r.POST("/deploy", deployHandler.Deploy)

	// WebSocket 端点
	r.GET("/ws", wsHandler.HandleWS)

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 认证模块（无需登录）
		auth := v1.Group("/auth")
		{
			auth.POST("/sms/send", authHandler.SendSMS)
			auth.POST("/sms/verify", authHandler.VerifySMS)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/wechat/login", authHandler.WechatLogin)
		}

		// 公开邀请接口（无需登录）
		v1.GET("/invite/:code", invitationHandler.GetInvitationByCode)

		// 需要认证的路由
		authorized := v1.Group("")
		authorized.Use(middleware.Auth(jwtManager))
		{
			// 用户模块
			users := authorized.Group("/users")
			{
				users.GET("/me", userHandler.GetMe)
				users.PUT("/me", userHandler.UpdateMe)
				users.GET("/me/poster", userHandler.GetMyPoster)
				users.GET("/me/invite", userHandler.GetMyInviteInfo)
				users.GET("/me/badge", userHandler.GetBadgeCount)
				users.GET("/settings", userHandler.GetSettings)
				users.PUT("/settings", userHandler.UpdateSettings)
				users.GET("/search", userHandler.SearchUsers)
				users.GET("/:id", userHandler.GetUser)
			}

			// 用户画像模块
			profile := authorized.Group("/profile")
			{
				profile.GET("", userProfileHandler.GetMyProfile)
				profile.PUT("", userProfileHandler.UpsertProfile)
				profile.POST("/simple", userProfileHandler.SaveSimpleProfile) // 简化版（引导流程）
				profile.GET("/status", userProfileHandler.GetProfileStatus)
				profile.DELETE("", userProfileHandler.DeleteProfile)
			}

			// 圈子模块
			circles := authorized.Group("/circles")
			{
				circles.GET("", circleHandler.GetCircles)
				circles.POST("", circleHandler.CreateCircle)
				circles.GET("/:id", circleHandler.GetCircle)
				circles.PUT("/:id", circleHandler.UpdateCircle)
				circles.DELETE("/:id", circleHandler.DeleteCircle)
				circles.POST("/:id/members", circleHandler.AddMember)
				circles.DELETE("/:id/members/:userId", circleHandler.RemoveMember)
			}

			// 首页宫格模块
			home := authorized.Group("/home")
			{
				home.GET("/grid", homeHandler.GetGrid)
			}

			// 会话消息模块
			conversations := authorized.Group("/conversations")
			{
				conversations.GET("", conversationHandler.GetConversations)
				conversations.POST("", conversationHandler.CreateConversation)
				conversations.GET("/:id/messages", conversationHandler.GetMessages)
				conversations.POST("/:id/messages", conversationHandler.SendMessage)
			}

			// 邀请模块
			invitations := authorized.Group("/invitations")
			{
				invitations.POST("", invitationHandler.CreateInvitation)
				invitations.GET("", invitationHandler.GetMyInvitations)
				invitations.GET("/:id", invitationHandler.GetInvitationDetail)
				invitations.DELETE("/:id", invitationHandler.DisableInvitation)
				invitations.GET("/:id/poster", invitationHandler.GetPoster)
				invitations.GET("/:id/qrcode", invitationHandler.GetQRCode)
			}

			// 接受邀请（使用邀请码）
			authorized.POST("/invite/:code/accept", invitationHandler.AcceptInvitation)

			// 好友模块
			friends := authorized.Group("/friends")
			{
				friends.GET("", friendshipHandler.GetFriends)
				friends.POST("/add-by-phone", friendshipHandler.AddFriendByPhone)
				friends.DELETE("/:userId", friendshipHandler.RemoveFriend)
				friends.GET("/invited-by-me", friendshipHandler.GetInvitedByMe)
				friends.GET("/invited-me", friendshipHandler.GetInvitedMe)
				friends.GET("/free-probability", agentHandler.GetFreeProbability)
				friends.GET("/holmes-probability", agentHandler.GetHolmesFreeProbability) // 福尔摩斯版（带推理过程）
				friends.GET("/:id/holmes", agentHandler.GetHolmesAnalysis)               // 单个好友详情

				// 好友请求
				friends.POST("/request", friendshipHandler.SendFriendRequest)
				friends.GET("/requests/received", friendshipHandler.GetReceivedRequests)
				friends.GET("/requests/sent", friendshipHandler.GetSentRequests)
				friends.POST("/requests/:id/handle", friendshipHandler.HandleFriendRequest)
				friends.GET("/requests/count", friendshipHandler.GetPendingRequestCount)
			}

			// Agent 模块
			agent := authorized.Group("/agent")
			{
				agent.POST("/status", agentHandler.ReportStatus)
				agent.POST("/status/stream", agentHandler.ReportStatusStream)   // 流式推理（SSE）
				agent.POST("/status/stream2", agentHandler.ReportStatus2Stream) // Holmes 2.0 流式推理
				agent.POST("/query", agentHandler.QueryAgentData)
				agent.GET("/memory", agentHandler.GetMemory)
				agent.GET("/my-analysis", agentHandler.GetMyAnalysis)                             // 获取我的分析结果
				agent.POST("/status-options", agentHandler.GenerateStatusOptionsStream)           // 流式生成状态选项
				agent.POST("/select-status", agentHandler.SelectStatus)                           // 选择状态并记录
				agent.POST("/onboarding-status-options", agentHandler.GetOnboardingStatusOptions) // 引导流程状态选项
				agent.POST("/voice-schedule/stream", agentHandler.VoiceScheduleStream)            // 语音时刻表（SSE）
				agent.POST("/voice-schedule/interact", agentHandler.VoiceScheduleInteract)        // 语音时刻表交互
				agent.POST("/voice-schedule/text", agentHandler.VoiceScheduleText)                // 语音时刻表文本测试
				agent.POST("/voice-schedule/v4/text", agentHandler.VoiceScheduleTextV4)           // V4 版本语音时刻表（简化架构）
				agent.POST("/chat/stream", agentHandler.AgentChatStream)                          // Tool Agent 聊天（SSE）
				agent.POST("/chat", agentHandler.AgentChat)                                       // Tool Agent 聊天（非流式）
				agent.POST("/voice/stream", agentHandler.AgentVoiceChatStream)                    // Tool Agent 语音聊天（SSE）
				agent.GET("/my-schedule/history", agentHandler.GetMyScheduleHistory)              // 我的状态时刻表历史（分页）
				agent.PUT("/my-schedule/:date/item", agentHandler.UpdateScheduleItem)             // 更新时刻表条目
				agent.DELETE("/my-schedule/:date/item", agentHandler.DeleteScheduleItem)          // 删除时刻表条目

				// 当下状态推理
				agent.POST("/infer-status", agentHandler.InferStatus)         // AI 推断当下状态
				agent.POST("/status-feedback", agentHandler.StatusFeedback)   // 状态反馈（用户修正）

				// AI 状态推测
				agent.POST("/prediction/start", predictionHandler.StartPrediction)      // 开始推测任务
				agent.GET("/prediction/latest", predictionHandler.GetLatestPrediction)  // 获取最近的推测任务
				agent.GET("/prediction/pending", predictionHandler.GetPendingPrediction) // 获取待确认的推测任务
				agent.GET("/prediction/:id", predictionHandler.GetPrediction)           // 获取推测任务状态
				agent.POST("/prediction/:id/confirm", predictionHandler.ConfirmPrediction) // 确认推测结果
				agent.POST("/prediction/:id/reject", predictionHandler.RejectPrediction)  // 放弃推测结果

				// 模型对比测试接口（Qwen vs Kimi）
				agent.GET("/test/model/cases", agentHandler.GetTestCases)                   // 获取测试用例
				agent.POST("/test/model/single", agentHandler.TestModelSingle)              // 单个测试
				agent.POST("/test/model/category", agentHandler.TestModelByCategory)        // 按分类测试
				agent.POST("/test/model/comparison", agentHandler.TestModelComparison)      // 完整对比测试
				agent.POST("/test/model/report", agentHandler.TestModelReport)              // 生成 Markdown 报告
			}

			// 通讯录模块
			contacts := authorized.Group("/contacts")
			{
				contacts.POST("/match", contactHandler.MatchContacts)
				contacts.POST("/add-friends", contactHandler.BatchAddFriends)
			}

			// 设备模块（推送 Token）
			devices := authorized.Group("/devices")
			{
				devices.POST("/token", deviceHandler.RegisterToken)
				devices.DELETE("/token", deviceHandler.UnregisterToken)
			}
		}
	}

	// 静态文件服务（SPA 支持）
	webDir := cfg.Deploy.WebDir
	if webDir != "" {
		// 检查目录是否存在
		if _, err := os.Stat(webDir); err == nil {
			logger.Info("启用静态文件服务", zap.String("dir", webDir))

			// 处理所有未匹配的路由
			r.NoRoute(func(c *gin.Context) {
				path := c.Request.URL.Path

				// API 路由返回 404
				if strings.HasPrefix(path, "/api/") {
					c.JSON(http.StatusNotFound, gin.H{"code": 1004, "message": "接口不存在"})
					return
				}

				// 尝试提供静态文件
				filePath := filepath.Join(webDir, path)
				if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
					c.File(filePath)
					return
				}

				// 对于 SPA 路由，返回 index.html
				indexPath := filepath.Join(webDir, "index.html")
				if _, err := os.Stat(indexPath); err == nil {
					c.File(indexPath)
					return
				}

				// 如果连 index.html 都没有
				c.JSON(http.StatusNotFound, gin.H{"code": 1004, "message": "页面不存在"})
			})
		} else {
			logger.Warn("静态文件目录不存在", zap.String("dir", webDir))
		}
	}

	// 初始化并启动状态时刻表调度器
	statusScheduler := job.NewStatusScheduler(scheduleRepo, memoryRepo, redisClient)
	statusScheduler.Start()
	defer statusScheduler.Stop()
	logger.Info("状态时刻表调度器已启动")

	// 初始化并启动每日状态推测任务
	if llmClient != nil {
		dailyPredictionJob := job.NewDailyPredictionJob(
			userSettingsRepo,
			scheduleRepo,
			memoryRepo,
			userProfileService,
			llmClient,
		)
		dailyPredictionJob.Start()
		defer dailyPredictionJob.Stop()
		logger.Info("每日状态推测任务已启动")
	} else {
		logger.Warn("LLM 未配置，每日状态推测任务未启动")
	}

	// 启动服务器
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	logger.Info("服务器启动", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("服务器启动失败", zap.Error(err))
	}
}
// build-27 test
