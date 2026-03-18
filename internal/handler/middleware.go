package handler 

import (
	"net/http"
	"strings"

	"quiz-server/internal/pkg/jwt" // 引入我们自己写的 jwt 工具包

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware 是一个 Gin 中间件，用来保护需要登录才能访问的接口
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 尝试从 HTTP 请求头中获取 Token
		// 规范的做法是放在 Authorization 头里，格式为 "Bearer <token>"
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 如果没带 Token，直接拒绝访问，返回 401 状态码
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录，请求头中缺少 Authorization"})
			c.Abort() // 终止请求链，不再往下执行后续的业务逻辑
			return
		}

		// 2. 检查 Token 的格式是否合法
		// 正常应该是 "Bearer eyJhbGciOiJIUz..." 这样的格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization 格式错误，应为 Bearer <token>"})
			c.Abort()
			return
		}

		// 3. 调用我们自己写的 jwt 工具包去解析并验证这个 Token
		tokenString := parts[1]
		claims, err := jwt.ParseToken(tokenString, secret)
		if err != nil {
			// 如果 Token 被篡改了，或者时间过期了，解析就会失败
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的或已过期的 Token: " + err.Error()})
			c.Abort()
			return
		}

		// 4. 【核心防御机制】将解析出来的、绝对可信的用户身份信息存入当前的请求上下文中
		// 这样后续的 Submit 等接口就可以直接调用 c.MustGet("username") 获取安全的身份，杜绝作弊
		c.Set("username", claims.Username)

		// 5. 验证通过，放行请求，让它继续去执行后面真正的业务逻辑（比如 GetQuiz 或 Submit）
		c.Next()
	}
}