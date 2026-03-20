package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"strings"

	"quiz-server/internal/model" 

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

// GetUserPassword 获取用户的哈希密码
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

// GetRandomQuestions 【重构亮点2】：废弃 ORDER BY RAND()，采用 Redis SRANDMEMBER 实现 O(1) 随机抽题
func (s *Storage) GetRandomQuestions(ctx context.Context, limit int) ([]model.Question, error) {
	// 1. 尝试从 Redis 的 Set 集合中随机抽取 limit 个题目 ID
	idsStr, err := s.rdb.SRandMemberN(ctx, "quiz:questions:ids", int64(limit)).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	var questionIDs []interface{}

	// 2. 缓存穿透补偿机制：如果 Redis 中没有 ID 缓存（系统刚启动），则从 MySQL 预热全量 ID
	if len(idsStr) == 0 {
		rows, err := s.db.QueryContext(ctx, "SELECT id FROM questions")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var allIDs []interface{}
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			allIDs = append(allIDs, id)
		}

		if len(allIDs) > 0 {
			// 将全量 ID 写入 Redis 的 Set 中，永不过期（除非题库更新）
			s.rdb.SAdd(ctx, "quiz:questions:ids", allIDs...)
			
			// 为本次请求随机挑选前 limit 个（下次请求就会直接命中 Redis 缓存了）
			for i := 0; i < limit && i < len(allIDs); i++ {
				questionIDs = append(questionIDs, allIDs[i])
			}
		}
	} else {
		for _, idStr := range idsStr {
			questionIDs = append(questionIDs, idStr)
		}
	}

	// 如果题库完全为空，直接返回
	if len(questionIDs) == 0 {
		return []model.Question{}, nil
	}

	// 3. 利用拿到的随机 ID，通过主键索引去 MySQL 极速拉取试题内容
	placeholders := make([]string, len(questionIDs))
	for i := range placeholders {
		placeholders[i] = "?" // 构造 (?, ?, ?) 占位符
	}
	querySQL := fmt.Sprintf(
		"SELECT id, text, opt_a, opt_b, opt_c, opt_d, answer FROM questions WHERE id IN (%s)",
		strings.Join(placeholders, ","),
	)

	rows, err := s.db.QueryContext(ctx, querySQL, questionIDs...)
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

	// IN 查询返回的结果顺序往往不是随机的，所以我们在内存里打乱它
	rand.Shuffle(len(questions), func(i, j int) {
		questions[i], questions[j] = questions[j], questions[i]
	})

	return questions, nil
}

// UpdateBestScore 【重构亮点1】：使用 MySQL UPSERT 彻底解决高并发下的“脏写/数据覆盖”漏洞
func (s *Storage) UpdateBestScore(ctx context.Context, username string, newScore, newTime int) (bool, error) {
	// 使用一行 SQL 搞定并发安全！
	// ON DUPLICATE KEY UPDATE: 如果冲突了，只有在“新分数更高”或“同分但耗时更短”时，才更新值
	query := `
		INSERT INTO user_scores (username, score, time_taken) 
		VALUES (?, ?, ?) 
		ON DUPLICATE KEY UPDATE 
			time_taken = IF(VALUES(score) > score OR (VALUES(score) = score AND VALUES(time_taken) < time_taken), VALUES(time_taken), time_taken),
			score = GREATEST(score, VALUES(score))
	`
	
	result, err := s.db.ExecContext(ctx, query, username, newScore, newTime)
	if err != nil {
		return false, err
	}

	// 巧妙判断是否刷新了最高记录：
	// MySQL 中 UPSERT 操作的 RowsAffected() 规则：
	// 插入新记录 = 1，更新老记录 = 2，数据没有变化(未打破记录) = 0
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	shouldUpdateCache := rowsAffected > 0

	// 只有当真正插入了新成绩，或打破了旧记录时，我们才去更新 Redis 排行榜
	if shouldUpdateCache {
		zScore := encodeZSetScore(newScore, newTime)
		s.rdb.ZAdd(ctx, "quiz:leaderboard:zset", redis.Z{
			Score:  zScore,
			Member: username,
		})
	}

	return shouldUpdateCache, nil
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