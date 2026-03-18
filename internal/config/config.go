package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServerAddr      string
	GinMode         string
	ShutdownTimeout time.Duration

	DBDSN          string
	DBMaxOpenConns int
	DBMaxIdleConns int
	DBConnMaxLife  time.Duration

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret string
	JWTExpire time.Duration // 【修复 2】：新增 JWT 过期时间配置项
}

func Load() (Config, error) {
	cfg := Config{
		ServerAddr:      getEnv("SERVER_ADDR", ":8080"),
		GinMode:         getEnv("GIN_MODE", "release"),
		ShutdownTimeout: time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_SEC", 5)) * time.Second,

		DBDSN:          strings.TrimSpace(os.Getenv("MYSQL_DSN")),
		DBMaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 100),
		DBMaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 100),
		DBConnMaxLife:  time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MIN", 60)) * time.Minute,

		RedisAddr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		JWTSecret: strings.TrimSpace(os.Getenv("JWT_SECRET")),
		
		JWTExpire: time.Duration(getEnvInt("JWT_EXPIRE_HOURS", 24)) * time.Hour, 
	}

	if cfg.DBDSN == "" {
		return Config{}, errors.New("致命错误: 环境变量 MYSQL_DSN 未设置")
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "default_temp_secret_key_for_dev_only"
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}