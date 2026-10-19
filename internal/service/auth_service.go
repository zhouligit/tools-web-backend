package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/find-work/tools-web-backend/internal/model"
	"github.com/find-work/tools-web-backend/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const (
	SessionCookieName = "tools_session"
	minPasswordLen    = 8
	maxPasswordLen    = 72
	bcryptCost        = 12
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

type AuthService struct {
	users    *store.UserStore
	sessions *store.SessionStore
	ttl      time.Duration

	mu       sync.Mutex
	failures map[string]failInfo
}

type failInfo struct {
	count   int
	until   time.Time
	lastAt  time.Time
}

func NewAuthService(users *store.UserStore, sessions *store.SessionStore, ttl time.Duration) *AuthService {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	return &AuthService{
		users:    users,
		sessions: sessions,
		ttl:      ttl,
		failures: make(map[string]failInfo),
	}
}

func (s *AuthService) Register(ctx context.Context, username, password string) (*model.User, error) {
	username = strings.TrimSpace(username)
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	existing, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("用户名已存在")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user, err := s.users.Create(ctx, username, string(hash), "user")
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "1062") {
			return nil, fmt.Errorf("用户名已存在")
		}
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, username, password, clientKey string) (string, *model.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", nil, fmt.Errorf("用户名或密码错误")
	}
	if err := s.checkRateLimit(clientKey); err != nil {
		return "", nil, err
	}

	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return "", nil, err
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		s.recordFailure(clientKey)
		return "", nil, fmt.Errorf("用户名或密码错误")
	}
	s.clearFailures(clientKey)

	sessionID, err := newSessionID()
	if err != nil {
		return "", nil, err
	}
	expires := time.Now().Add(s.ttl)
	if err := s.sessions.Create(ctx, sessionID, user.ID, expires); err != nil {
		return "", nil, err
	}
	return sessionID, user, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.sessions.Delete(ctx, sessionID)
}

func (s *AuthService) UserFromSession(ctx context.Context, sessionID string) (*model.User, error) {
	if sessionID == "" {
		return nil, nil
	}
	userID, expiresAt, err := s.sessions.GetUserID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if userID == 0 {
		return nil, nil
	}
	if time.Now().After(expiresAt) {
		_ = s.sessions.Delete(ctx, sessionID)
		return nil, nil
	}
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) SessionTTL() time.Duration {
	return s.ttl
}

func validateUsername(username string) error {
	if !usernameRe.MatchString(username) {
		return fmt.Errorf("用户名需为 3–32 位字母、数字或下划线")
	}
	return nil
}

func validatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < minPasswordLen {
		return fmt.Errorf("密码至少 %d 位", minPasswordLen)
	}
	if len(password) > maxPasswordLen {
		return fmt.Errorf("密码过长")
	}
	return nil
}

func newSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *AuthService) checkRateLimit(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.failures[key]
	if !ok {
		return nil
	}
	if time.Now().Before(info.until) {
		return fmt.Errorf("登录尝试过多，请稍后再试")
	}
	return nil
}

func (s *AuthService) recordFailure(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info := s.failures[key]
	info.count++
	info.lastAt = time.Now()
	if info.count >= 8 {
		info.until = time.Now().Add(15 * time.Minute)
		info.count = 0
	}
	s.failures[key] = info
}

func (s *AuthService) clearFailures(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.failures, key)
}
