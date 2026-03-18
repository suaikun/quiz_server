package model

// Question 存放题库的数据模型
type Question struct {
	ID      int      `json:"id"`
	Text    string   `json:"text"`
	Options []string `json:"options"`
	Answer  string   `json:"-"` 
}

// LeaderboardEntry 排行榜展现用的数据模型
type LeaderboardEntry struct {
	Username  string `json:"username"`
	Score     int    `json:"score"`
	TimeTaken int    `json:"time_taken"`
}

// ================= HTTP 请求载荷定义 =================

// UserRequest 定义了注册/登录时前端传过来的 JSON 格式
type UserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// SubmitRequest 定义了玩家交卷时传过来的 JSON 格式
type SubmitRequest struct {
	Username  string `json:"username" binding:"required"`
	Score     int    `json:"score" binding:"required"`
	TimeTaken int    `json:"time_taken" binding:"required"`
}