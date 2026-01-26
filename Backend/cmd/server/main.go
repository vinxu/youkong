package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"youkong/internal/config"
	"youkong/internal/handler"
	"youkong/internal/middleware"
	"youkong/internal/pkg/jwt"
	"youkong/internal/pkg/poster"
	"youkong/internal/pkg/tencent"
	"youkong/internal/pkg/wechat"
	"youkong/internal/repository"
	"youkong/internal/service"
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

	// 初始化腾讯云SMS客户端
	smsClient, err := tencent.NewSMSClient(
		cfg.Tencent.SecretID,
		cfg.Tencent.SecretKey,
		cfg.Tencent.SMSAppID,
		cfg.Tencent.SMSSignName,
		cfg.Tencent.SMSTemplateID,
	)
	if err != nil {
		logger.Warn("初始化SMS客户端失败，短信功能将不可用", zap.Error(err))
	}

	// 初始化JWT管理器
	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.ExpireHours)

	// 初始化Repository
	userRepo := repository.NewUserRepository(db)
	circleRepo := repository.NewCircleRepository(db)
	availabilityRepo := repository.NewAvailabilityRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	wechatRepo := repository.NewWechatRepository(db)
	invitationRepo := repository.NewInvitationRepository(db)
	friendshipRepo := repository.NewFriendshipRepository(db)

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

	// 初始化Service
	smsService := service.NewSMSService(smsClient, redisClient)
	authService := service.NewAuthService(userRepo, smsService, jwtManager)
	userService := service.NewUserService(userRepo)
	circleService := service.NewCircleService(circleRepo, userRepo)
	availabilityService := service.NewAvailabilityService(availabilityRepo, circleRepo, userRepo)
	conversationService := service.NewConversationService(messageRepo, userRepo)
	wechatService := service.NewWechatService(wechatRepo, userRepo, invitationRepo, friendshipRepo, circleRepo, wechatClient, jwtManager)
	invitationService := service.NewInvitationService(invitationRepo, circleRepo, userRepo, friendshipRepo, cfg.Invitation.BaseURL)
	friendshipService := service.NewFriendshipService(friendshipRepo, userRepo, invitationRepo, circleRepo)
	agentService := service.NewAgentService(redisClient, userRepo, friendshipRepo)
	contactService := service.NewContactService(userRepo, friendshipRepo)

	// 初始化Handler
	authHandler := handler.NewAuthHandler(authService, wechatService)
	userHandler := handler.NewUserHandler(userService)
	circleHandler := handler.NewCircleHandler(circleService)
	availabilityHandler := handler.NewAvailabilityHandler(availabilityService)
	conversationHandler := handler.NewConversationHandler(conversationService)
	invitationHandler := handler.NewInvitationHandler(invitationService, posterGenerator)
	friendshipHandler := handler.NewFriendshipHandler(friendshipService)
	agentHandler := handler.NewAgentHandler(agentService)
	contactHandler := handler.NewContactHandler(contactService)

	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.Logger(logger))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

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
				users.GET("/search", userHandler.SearchUsers)
				users.GET("/:id", userHandler.GetUser)
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

			// 有空状态模块
			availabilities := authorized.Group("/availabilities")
			{
				availabilities.GET("/friends", availabilityHandler.GetFriendsAvailabilities)
				availabilities.GET("/mine", availabilityHandler.GetMyAvailabilities)
				availabilities.POST("", availabilityHandler.CreateAvailability)
				availabilities.DELETE("/:id", availabilityHandler.CancelAvailability)
			}

			// 会话消息模块
			conversations := authorized.Group("/conversations")
			{
				conversations.GET("", conversationHandler.GetConversations)
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
				friends.DELETE("/:userId", friendshipHandler.RemoveFriend)
				friends.GET("/invited-by-me", friendshipHandler.GetInvitedByMe)
				friends.GET("/invited-me", friendshipHandler.GetInvitedMe)
				friends.GET("/free-probability", agentHandler.GetFreeProbability)
			}

			// Agent 模块
			agent := authorized.Group("/agent")
			{
				agent.POST("/status", agentHandler.ReportStatus)
				agent.POST("/query", agentHandler.QueryAgentData)
			}

			// 通讯录模块
			contacts := authorized.Group("/contacts")
			{
				contacts.POST("/match", contactHandler.MatchContacts)
				contacts.POST("/add-friends", contactHandler.BatchAddFriends)
			}
		}
	}

	// 启动服务器
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	logger.Info("服务器启动", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("服务器启动失败", zap.Error(err))
	}
}
