package repository

import (
	"context"
	"database/sql"
	"math"

	"quiz-server/internal/model" // 假设你的 go.mod 模块名为 quiz-server

	"github.com/redis/go-redis/v9"
)

// Storage 结构体封装了底层的数据源
type Storage struct {
	db  *sql.DB
	rdb *redis.Client
}

// NewStorage 是工厂函数，用于依赖注入
func NewStorage(db *sql.DB, rdb *redis.Client) *Storage {
	return &Storage{db: db, rdb: rdb}
}

// GetUserPassword 获取用户的哈希密码 (目前暂为明文)
func (s *Storage) GetUserPassword(ctx context.Context, username string) (string, error) {
	var pwd string
	err := s.db.QueryRowContext(ctx, "SELECT password FROM users WHERE username = ?", username).Scan(&pwd)
	return pwd, err
}

// CreateUser 创建新用户
func (s *Storage) CreateUser(ctx context.Context, username, password string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO users (username, password) VALUES (?, ?)", username, password)
	return err
}

// GetRandomQuestions 从数据库随机获取题目
func (s *Storage) GetRandomQuestions(ctx context.Context, limit int) ([]model.Question, error) {
	querySQL := "SELECT id, text, opt_a, opt_b, opt_c, opt_d, answer FROM questions ORDER BY RAND() LIMIT ?"
	rows, err := s.db.QueryContext(ctx, querySQL, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []model.Question
	for rows.Next() {
		var q model.Question
		var a, b, c, d string
		if err := rows.Scan(&q.ID, &q.Text, &a, &b, &c, &d, &q.Answer); err != nil {
			return nil, err
		}
		q.Options = []string{a, b, c, d}
		questions = append(questions, q)
	}

	// 检查遍历过程中是否发生了错误
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return questions, nil
}

// UpdateBestScore 更新用户最高分并同步 Redis
func (s *Storage) UpdateBestScore(ctx context.Context, username string, newScore, newTime int) (bool, error) {
	var currentScore, currentTime int
	err := s.db.QueryRowContext(ctx, "SELECT score, time_taken FROM user_scores WHERE username = ?", username).Scan(&currentScore, &currentTime)

	shouldUpdateCache := false

	if err == sql.ErrNoRows {
		// 首次答题
		_, err = s.db.ExecContext(ctx, "INSERT INTO user_scores (username, score, time_taken) VALUES (?, ?, ?)", username, newScore, newTime)
		if err == nil {
			shouldUpdateCache = true
		}
	} else if err == nil {
		// 打破记录
		if newScore > currentScore || (newScore == currentScore && newTime < currentTime) {
			_, err = s.db.ExecContext(ctx, "UPDATE user_scores SET score = ?, time_taken = ? WHERE username = ?", newScore, newTime, username)
			if err == nil {
				shouldUpdateCache = true
			}
		}
	}

	if shouldUpdateCache {
		zScore := encodeZSetScore(newScore, newTime)
		s.rdb.ZAdd(ctx, "quiz:leaderboard:zset", redis.Z{
			Score:  zScore,
			Member: username,
		})
	}
	return shouldUpdateCache, err
}

// GetTopScores 从 Redis 获取排行榜
func (s *Storage) GetTopScores(ctx context.Context, topN int64) ([]model.LeaderboardEntry, error) {
	results, err := s.rdb.ZRevRangeWithScores(ctx, "quiz:leaderboard:zset", 0, topN-1).Result()
	if err != nil {
		return nil, err
	}

	var entries []model.LeaderboardEntry
	for _, z := range results {
		username := z.Member.(string)
		score, timeTaken := decodeZSetScore(z.Score)
		entries = append(entries, model.LeaderboardEntry{
			Username:  username,
			Score:     score,
			TimeTaken: timeTaken,
		})
	}
	return entries, nil
}

// 内部工具函数：不再对外暴露
func encodeZSetScore(score int, timeTaken int) float64 {
	return float64(score) - (float64(timeTaken) / 100000.0)
}

func decodeZSetScore(zScore float64) (score int, timeTaken int) {
	score = int(math.Ceil(zScore))
	timeDiff := float64(score) - zScore
	timeTaken = int(math.Round(timeDiff * 1000000.0))
	return score, timeTaken
}