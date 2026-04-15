package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"tango/internal/application/command"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// APIKeyLookup is a function that resolves a hashed API key to the owning user ID.
// Injected at startup to avoid circular imports.
type APIKeyLookup func(ctx context.Context, keyHash string) (*domain.APIKey, error)

// ── Models ───────────────────────────────────────

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Handler struct {
	repo           domain.UserRepository
	changePassword *command.ChangePasswordHandler
}

func NewHandler(repo domain.UserRepository, changePassword *command.ChangePasswordHandler) *Handler {
	return &Handler{repo: repo, changePassword: changePassword}
}

// ── Password ─────────────────────────────────────

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func clearAuthCookies(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/api/auth", "", false, true)
}

// ── JWT ──────────────────────────────────────────

func GenerateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"type":    "access",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func GenerateRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type":    "refresh",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func VerifyToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return token.Claims.(jwt.MapClaims), nil
}

// ── Handlers ─────────────────────────────────────

// Login godoc
// @Summary Login
// @Description Authenticates a user and returns an access token. A refresh token is set in an httpOnly cookie.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Credentials"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}

	user, err := h.repo.FindByEmail(c.Request.Context(), req.Email)
	if err != nil {
		_ = c.Error(response.Internal(""))
		return
	}
	if user == nil || !CheckPassword(req.Password, user.PasswordHash) {
		_ = c.Error(response.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is incorrect"))
		return
	}

	accessToken, err := GenerateAccessToken(user.ID)
	if err != nil {
		_ = c.Error(response.Internal("Token generation failed"))
		return
	}
	refreshToken, err := GenerateRefreshToken(user.ID)
	if err != nil {
		_ = c.Error(response.Internal("Token generation failed"))
		return
	}

	c.SetCookie("access_token", accessToken, 900, "/", "", false, true)
	c.SetCookie("refresh_token", refreshToken, 7*24*3600, "/api/auth", "", false, true)
	response.OK(c, TokenResponse{AccessToken: accessToken})
}

// Refresh godoc
// @Summary Refresh access token
// @Description Exchanges the refresh token cookie for a new access token.
// @Tags auth
// @Produce json
// @Success 200 {object} TokenResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		_ = c.Error(response.Unauthorized("Refresh token is missing"))
		return
	}

	claims, err := VerifyToken(refreshToken)
	if err != nil || claims["type"] != "refresh" {
		_ = c.Error(response.Unauthorized("Refresh token is invalid"))
		return
	}

	userID := claims["user_id"].(string)
	newAccessToken, _ := GenerateAccessToken(userID)
	c.SetCookie("access_token", newAccessToken, 900, "/", "", false, true)
	response.OK(c, TokenResponse{AccessToken: newAccessToken})
}

// Logout godoc
// @Summary Logout
// @Description Clears auth cookies.
// @Tags auth
// @Produce json
// @Success 200 {object} MessageResponse
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	clearAuthCookies(c)
	response.OK(c, MessageResponse{Message: "Logged out"})
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=6"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword godoc
// @Summary Change password
// @Description Changes the current user's password and clears auth cookies so they must log in again.
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Password change payload"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/change-password [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}

	if h.changePassword == nil {
		_ = c.Error(response.Internal("Change password handler is not configured"))
		return
	}

	if err := h.changePassword.Handle(c.Request.Context(), command.ChangePasswordCommand{
		ID:              c.GetString("user_id"),
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		switch err {
		case domain.ErrInvalidCredentials:
			_ = c.Error(response.Unauthorized("Current password is incorrect"))
		case domain.ErrInvalidInput:
			_ = c.Error(response.BadRequest("Invalid password payload"))
		default:
			_ = c.Error(response.Internal(""))
		}
		return
	}

	clearAuthCookies(c)
	response.OK(c, MessageResponse{Message: "Password changed. Please log in again."})
}

// ── Middleware ───────────────────────────────────

// Middleware returns a Gin handler that authenticates via JWT cookie or X-API-Key header.
// apiKeyLookup may be nil if API key auth is not needed.
func Middleware(apiKeyLookup ...APIKeyLookup) gin.HandlerFunc {
	var lookup APIKeyLookup
	if len(apiKeyLookup) > 0 {
		lookup = apiKeyLookup[0]
	}

	return func(c *gin.Context) {
		// 1. Try API key header first
		if lookup != nil {
			if rawKey := c.GetHeader("X-API-Key"); rawKey != "" {
				sum := sha256.Sum256([]byte(rawKey))
				keyHash := hex.EncodeToString(sum[:])

				apiKey, err := lookup(c.Request.Context(), keyHash)
				if err != nil || apiKey == nil {
					_ = c.Error(response.Unauthorized("Invalid API key"))
					c.Abort()
					return
				}
				if apiKey.IsExpired() {
					_ = c.Error(response.New(http.StatusUnauthorized, "API_KEY_EXPIRED", "API key has expired"))
					c.Abort()
					return
				}
				c.Set("user_id", apiKey.UserID)
				c.Set("auth_method", "api_key")
				c.Set("api_key_id", apiKey.ID)
				c.Next()
				return
			}
		}

		// 2. Fall back to JWT cookie
		token, err := c.Cookie("access_token")
		if err != nil || token == "" {
			_ = c.Error(response.Unauthorized("Missing access token"))
			c.Abort()
			return
		}

		claims, err := VerifyToken(token)
		if err != nil {
			_ = c.Error(response.Unauthorized("Invalid token"))
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("auth_method", "jwt")
		c.Next()
	}
}

func SeedDemoData(ctx context.Context, userRepo domain.UserRepository, roleRepo domain.RoleRepository) error {
	roles := systemRoles()
	for _, role := range roles {
		if err := roleRepo.EnsureRole(ctx, role); err != nil {
			return err
		}
	}

	existing, err := userRepo.FindByEmail(ctx, "demo.admin@example.com")
	if err != nil {
		return err
	}
	userID := "user_0123"
	if existing == nil {
		passwordHash, err := HashPassword("password123")
		if err != nil {
			return err
		}
		user, err := domain.NewUser(userID, "demo.admin@example.com", "demo.admin", "Demo", "Admin", "", "", passwordHash)
		if err != nil {
			return err
		}
		if _, err := userRepo.Save(ctx, user); err != nil {
			return err
		}
	} else {
		userID = existing.ID
	}

	adminRole, err := roleRepo.GetByName(ctx, "admin")
	if err != nil {
		return err
	}
	return roleRepo.AssignRoleToUser(ctx, userID, adminRole.ID)
}

func systemRoles() []*domain.Role {
	now := time.Now().UTC()
	return []*domain.Role{
		{ID: "role_super_admin", Name: "super_admin", Description: "Super administrator role", IsSystem: true, CreatedAt: now, UpdatedAt: now},
		{ID: "role_admin", Name: "admin", Description: "Administrator role", IsSystem: true, CreatedAt: now, UpdatedAt: now},
		{ID: "role_user", Name: "user", Description: "Default user role", IsSystem: true, CreatedAt: now, UpdatedAt: now},
	}
}
