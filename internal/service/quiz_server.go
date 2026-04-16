package service

import (
	"context"
	"database/sql"
	"errors"
	
	"quiz-server/internal/config"
	"quiz-server/internal/model"
	"quiz-server/internal/pkg/jwt"
	"quiz-server/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound    = errors.New("用户不存在")
	ErrWrongPassword   = errors.New("密码错误")
	ErrUserExists      = errors.New("用户名已存在")
	ErrInternalDBError = errors.New("数据库内部错误")
)

type QuizService struct {
	repo *repository.Storage
	cfg  config.Config 
}

func NewQuizService(repo *repository.Storage, cfg config.Config) *QuizService {
	return &QuizService{repo: repo, cfg: cfg}
}

// Register 处理注册逻辑 
func (s *QuizService) Register(ctx context.Context, username, password string) error {
	// 加盐哈希加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}

	err = s.repo.CreateUser(ctx, username, string(hashedPassword))
	if err != nil {
		return ErrUserExists
	}
	return nil
}

// Login 处理登录逻辑，返回生成的 JWT Token
func (s *QuizService) Login(ctx context.Context, username, password string) (string, error) {
	storedHashedPwd, err := s.repo.GetUserPassword(ctx, username)
	if err == sql.ErrNoRows {
		return "", ErrUserNotFound
	}
	if err != nil {
		return "", ErrInternalDBError
	}

	// 使用 bcrypt 校验哈希密码
	err = bcrypt.CompareHashAndPassword([]byte(storedHashedPwd), []byte(password))
	if err != nil {
		return "", ErrWrongPassword
	}

	// 密码校验通过，颁发防伪身份证 Token
	token, err := jwt.GenerateToken(username, s.cfg.JWTSecret, s.cfg.JWTExpire)
	if err != nil {
		return "", errors.New("生成令牌失败")
	}

	return token, nil
}

func (s *QuizService) GetQuizPaper(ctx context.Context) ([]model.Question, error) {
	return s.repo.GetRandomQuestions(ctx, 5)
}

func (s *QuizService) SubmitResult(ctx context.Context, username string, score, timeTaken int) (bool, error) {
	return s.repo.UpdateBestScore(ctx, username, score, timeTaken)
}

func (s *QuizService) GetLeaderboard(ctx context.Context) ([]model.LeaderboardEntry, error) {
	return s.repo.GetTopScores(ctx, 10)
}