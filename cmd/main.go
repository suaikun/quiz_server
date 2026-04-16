package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"quiz-server/internal/config"
	"quiz-server/internal/handler"
	"quiz-server/internal/repository"
	"quiz-server/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

// initDB 初始化 MySQL 连接池
func initDB(cfg config.Config) *sql.DB {
	db, err := sql.Open("mysql", cfg.DBDSN)
	if err != nil || db.Ping() != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	// 使用配置项动态设置连接池
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLife)

	fmt.Println("MySQL 连接成功")
	return db
}

// initRedis 初始化 Redis 客户端
func initRedis(cfg config.Config) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	fmt.Println("Redis 连接成功")
	return rdb
}

func main() {
	// 系统启动第一步：加载所有环境配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v\n", err)
	}

	// 初始化底层数据存储
	db := initDB(cfg)
	rdb := initRedis(cfg)
	repo := repository.NewStorage(db, rdb)
	
	// 依赖注入
	svc := service.NewQuizService(repo, cfg)
	httpHandler := handler.NewHTTPHandler(svc, cfg) 

	// 配置并启动 Gin 引擎
	gin.SetMode(cfg.GinMode)
	r := gin.Default()
	httpHandler.RegisterRoutes(r)

	// 准备 HTTP 服务器实例
	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: r,
	}

	// 放到独立的协程中启动，不阻塞主线程
	go func() {
		fmt.Printf("答题系统 RESTful API 已启动，监听地址 %s\n", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器异常崩溃: %v\n", err)
		}
	}()

	// 6. 监听退出信号，实现优雅退出 (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("\n 收到关闭信号，准备优雅关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("服务器被迫关闭 (存在未处理完的请求): ", err)
	}

	log.Println("正在断开 MySQL 数据库连接...")
	db.Close()
	log.Println("正在断开 Redis 连接...")
	rdb.Close()

	log.Println("服务器已彻底安全退出。再见！")
}