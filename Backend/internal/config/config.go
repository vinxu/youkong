package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig
	DB         DBConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Tencent    TencentConfig
	Wechat     WechatConfig
	Invitation InvitationConfig
	LLM        LLMConfig
}

type LLMConfig struct {
	Provider string // openrouter
	APIKey   string
	Model    string
}

type ServerConfig struct {
	Port string
	Mode string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
}

type TencentConfig struct {
	SecretID      string
	SecretKey     string
	SMSAppID      string
	SMSSignName   string
	SMSTemplateID string
	COSBucket     string
	COSRegion     string
}

type WechatConfig struct {
	AppID     string
	AppSecret string
}

type InvitationConfig struct {
	BaseURL           string
	DefaultExpireDays int
	DefaultMaxUses    int
	RateLimitPerDay   int
}

func (d *DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 设置默认值
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("SERVER_MODE", "debug")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "3306")
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("JWT_EXPIRE_HOURS", 168)
	viper.SetDefault("INVITATION_BASE_URL", "https://youkong.app/i/")
	viper.SetDefault("INVITATION_DEFAULT_EXPIRE_DAYS", 7)
	viper.SetDefault("INVITATION_DEFAULT_MAX_USES", 100)
	viper.SetDefault("INVITATION_RATE_LIMIT_PER_DAY", 10)
	viper.SetDefault("LLM_PROVIDER", "openrouter")
	viper.SetDefault("LLM_MODEL", "google/gemini-2.5-pro-preview-06-05")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: viper.GetString("SERVER_PORT"),
			Mode: viper.GetString("SERVER_MODE"),
		},
		DB: DBConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_NAME"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			Secret:      viper.GetString("JWT_SECRET"),
			ExpireHours: viper.GetInt("JWT_EXPIRE_HOURS"),
		},
		Tencent: TencentConfig{
			SecretID:      viper.GetString("TENCENT_SECRET_ID"),
			SecretKey:     viper.GetString("TENCENT_SECRET_KEY"),
			SMSAppID:      viper.GetString("TENCENT_SMS_SDK_APP_ID"),
			SMSSignName:   viper.GetString("TENCENT_SMS_SIGN_NAME"),
			SMSTemplateID: viper.GetString("TENCENT_SMS_TEMPLATE_ID"),
			COSBucket:     viper.GetString("TENCENT_COS_BUCKET"),
			COSRegion:     viper.GetString("TENCENT_COS_REGION"),
		},
		Wechat: WechatConfig{
			AppID:     viper.GetString("WECHAT_APP_ID"),
			AppSecret: viper.GetString("WECHAT_APP_SECRET"),
		},
		Invitation: InvitationConfig{
			BaseURL:           viper.GetString("INVITATION_BASE_URL"),
			DefaultExpireDays: viper.GetInt("INVITATION_DEFAULT_EXPIRE_DAYS"),
			DefaultMaxUses:    viper.GetInt("INVITATION_DEFAULT_MAX_USES"),
			RateLimitPerDay:   viper.GetInt("INVITATION_RATE_LIMIT_PER_DAY"),
		},
		LLM: LLMConfig{
			Provider: viper.GetString("LLM_PROVIDER"),
			APIKey:   viper.GetString("LLM_API_KEY"),
			Model:    viper.GetString("LLM_MODEL"),
		},
	}

	return cfg, nil
}
