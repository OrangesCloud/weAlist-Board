package database

import (
	"context"
	"project-board-api/internal/config"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

var RedisClient *redis.Client

func InitRedis(cfg config.Config, log *zap.Logger) error {
	var client *redis.Client

	// redis:// 형식 URL 있으면 우선 사용
	if cfg.Redis.URL != "" {
		opts, err := redis.ParseURL(cfg.Redis.URL)
		if err != nil {
			return err
		}
		client = redis.NewClient(opts)
	} else {
		// Password 처리: "NONE"이나 플레이스홀더는 빈 문자열로 처리
		password := cfg.Redis.Password
		if password == "NONE" || password == "CHANGE_THIS_REDIS_AUTH_TOKEN_32_CHAR_MIN" {
			password = ""
		}
		
		client = redis.NewClient(&redis.Options{
			Addr:     "redis:6379",       // docker-compose 내 컨테이너 이름
			Password: password,
			DB:       1,
		})
	}

	// 연결 테스트 (실패해도 서비스는 시작)
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Warn("Redis connection failed, but service will continue without Redis", zap.Error(err))
		RedisClient = nil
		return nil
	}

	RedisClient = client
	log.Info("Redis connection established successfully", zap.String("addr", "redis:6379"), zap.Int("db", cfg.Redis.DB))
	return nil
}

func GetRedis() *redis.Client {
	// Return nil instead of panicking to allow tests to run without Redis
	return RedisClient
}
