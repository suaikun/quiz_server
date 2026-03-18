package handler

import (
	"context"
	"net/http"
	"time"

	"quiz-server/internal/config"
	"quiz-server/internal/model"
	"quiz-server/internal/service"

	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	svc *service.QuizService
	cfg config.Config
}

func NewHTTPHandler(svc *service.QuizService, cfg config.Config) *HTTPHandler {
	return &HTTPHandler{svc: svc, cfg: cfg}
}

func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// 1. 【公开接口】：注册、登录、看排行榜不需要身份证
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)
		api.GET("/leaderboard", h.GetLeaderboard) // 这里用到了 GetLeaderboard

		// 2. 【私密接口】：创建一个受 JWTAuthMiddleware 保护的路由组
		// 只有带有合法 Token 的请求才能访问这里的接口
		authApi := api.Group("/")
		authApi.Use(JWTAuthMiddleware(h.cfg.JWTSecret))
		{
			authApi.GET("/quiz", h.GetQuiz)
			authApi.POST("/submit", h.Submit)
		}
	}
}

func handleError(c *gin.Context, err error, defaultMsg string) {
	if err == context.DeadlineExceeded {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "系统处理超时，请稍后再试"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": defaultMsg + ": " + err.Error()})
}

func (h *HTTPHandler) Register(c *gin.Context) {
	var req model.UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，请提供 username 和 password"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.svc.Register(ctx, req.Username, req.Password)
	if err != nil {
		if err == context.DeadlineExceeded {
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "注册超时"})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "注册成功"})
}

func (h *HTTPHandler) Login(c *gin.Context) {
	var req model.UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 接收返回的 token
	token, err := h.svc.Login(ctx, req.Username, req.Password)
	if err != nil {
		if err == context.DeadlineExceeded {
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "登录超时"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 把令牌发给前端，前端以后每次请求都要带上它！
	c.JSON(http.StatusOK, gin.H{"message": "登录成功", "token": token})
}

func (h *HTTPHandler) GetQuiz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	questions, err := h.svc.GetQuizPaper(ctx)
	if err != nil {
		handleError(c, err, "获取试卷失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "获取成功", "data": questions})
}

func (h *HTTPHandler) Submit(c *gin.Context) {
	var req model.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误或缺失"})
		return
	}

	trustedUsername := c.MustGet("username").(string)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	isRecord, err := h.svc.SubmitResult(ctx, trustedUsername, req.Score, req.TimeTaken)
	if err != nil {
		handleError(c, err, "成绩提交失败")
		return
	}

	msg := "成绩已记录"
	if isRecord {
		msg = "恭喜！您打破了个人历史最高记录！"
	}

	c.JSON(http.StatusOK, gin.H{"message": msg, "is_new_record": isRecord})
}

// GetLeaderboard 获取排行榜
func (h *HTTPHandler) GetLeaderboard(c *gin.Context) {
	// Redis 查询极快，如果 2 秒还没返回说明 Redis 出大问题了，直接超时切断
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	entries, err := h.svc.GetLeaderboard(ctx)
	if err != nil {
		handleError(c, err, "获取排行榜失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "获取成功", "data": entries})
}