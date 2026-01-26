package tencent

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr, password string, db int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接Redis失败: %w", err)
	}

	return &RedisClient{client: client}, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

// 验证码相关操作
const (
	smsCodePrefix     = "sms:code:"
	smsCodeExpire     = 5 * time.Minute
	smsRateLimitKey   = "sms:rate:"
	smsRateLimitExpire = 1 * time.Minute
)

func (r *RedisClient) SetSMSCode(ctx context.Context, phone, code string) error {
	key := smsCodePrefix + phone
	return r.client.Set(ctx, key, code, smsCodeExpire).Err()
}

func (r *RedisClient) GetSMSCode(ctx context.Context, phone string) (string, error) {
	key := smsCodePrefix + phone
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) DeleteSMSCode(ctx context.Context, phone string) error {
	key := smsCodePrefix + phone
	return r.client.Del(ctx, key).Err()
}

func (r *RedisClient) CheckSMSRateLimit(ctx context.Context, phone string) (bool, error) {
	key := smsRateLimitKey + phone
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists == 0, nil
}

func (r *RedisClient) SetSMSRateLimit(ctx context.Context, phone string) error {
	key := smsRateLimitKey + phone
	return r.client.Set(ctx, key, "1", smsRateLimitExpire).Err()
}

// 通用操作
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	return result > 0, err
}

// GetBytes 获取字节数据
func (r *RedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return r.client.Get(ctx, key).Bytes()
}

// IsNil 检查错误是否为 key 不存在
func IsNil(err error) bool {
	return err != nil && err.Error() == "redis: nil"
}
