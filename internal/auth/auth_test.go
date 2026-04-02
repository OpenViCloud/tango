package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tango/internal/application/command"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepository struct {
	users map[string]*domain.User
}

func newFakeUserRepository(users ...*domain.User) *fakeUserRepository {
	repo := &fakeUserRepository{
		users: make(map[string]*domain.User, len(users)),
	}
	for _, user := range users {
		repo.users[user.ID] = cloneUser(user)
	}
	return repo
}

func (r *fakeUserRepository) Save(_ context.Context, user *domain.User) (*domain.User, error) {
	r.users[user.ID] = cloneUser(user)
	return cloneUser(user), nil
}

func (r *fakeUserRepository) Update(_ context.Context, user *domain.User) (*domain.User, error) {
	if _, ok := r.users[user.ID]; !ok {
		return nil, domain.ErrUserNotFound
	}
	r.users[user.ID] = cloneUser(user)
	return cloneUser(user), nil
}

func (r *fakeUserRepository) Delete(_ context.Context, id string) error {
	delete(r.users, id)
	return nil
}

func (r *fakeUserRepository) FindByID(_ context.Context, id string) (*domain.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	return cloneUser(user), nil
}

func (r *fakeUserRepository) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return cloneUser(user), nil
		}
	}
	return nil, nil
}

func (r *fakeUserRepository) GetByID(_ context.Context, id string) (*domain.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return cloneUser(user), nil
}

func (r *fakeUserRepository) GetAll(_ context.Context, _ domain.UserListOptions) (*domain.UserListResult, error) {
	return &domain.UserListResult{}, nil
}

func cloneUser(user *domain.User) *domain.User {
	if user == nil {
		return nil
	}
	copy := *user
	return &copy
}

func newTestUser(t *testing.T) *domain.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate hash: %v", err)
	}

	return &domain.User{
		ID:           "user-1",
		Email:        "admin@example.com",
		FirstName:    "Admin",
		LastName:     "User",
		PasswordHash: string(hash),
		Status:       domain.UserStatusActive,
		CreatedAt:    time.Now().UTC().Add(-time.Hour),
		UpdatedAt:    time.Now().UTC().Add(-time.Hour),
	}
}

func TestChangePasswordClearsCookiesAndReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := newTestUser(t)
	repo := newFakeUserRepository(user)
	handler := NewHandler(repo, command.NewChangePasswordHandler(repo))

	router := gin.New()
	router.Use(response.Middleware(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))))
	router.POST("/api/auth/change-password", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		handler.ChangePassword(c)
	})

	body := []byte(`{"current_password":"old-password","new_password":"new-password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	setCookies := rec.Header().Values("Set-Cookie")
	if len(setCookies) < 2 {
		t.Fatalf("expected auth cookies to be cleared, got %v", setCookies)
	}
	var sawAccess, sawRefresh bool
	for _, header := range setCookies {
		if strings.Contains(header, "access_token=") {
			sawAccess = true
		}
		if strings.Contains(header, "refresh_token=") {
			sawRefresh = true
		}
	}
	if !sawAccess || !sawRefresh {
		t.Fatalf("expected cleared access and refresh cookies, got %v", setCookies)
	}

	updated, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("new-password")) != nil {
		t.Fatal("expected new password to be persisted")
	}
}

func TestChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user := newTestUser(t)
	repo := newFakeUserRepository(user)
	handler := NewHandler(repo, command.NewChangePasswordHandler(repo))

	router := gin.New()
	router.Use(response.Middleware(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))))
	router.POST("/api/auth/change-password", func(c *gin.Context) {
		c.Set("user_id", user.ID)
		handler.ChangePassword(c)
	})

	body := []byte(`{"current_password":"wrong-password","new_password":"new-password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Message == "" {
		t.Fatal("expected error message in unauthorized response")
	}
}
