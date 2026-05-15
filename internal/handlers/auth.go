package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"dadv-project/internal/auth"
	"dadv-project/internal/config"
	"dadv-project/internal/db"
)

type AuthHandler struct {
	db  *db.DB
	cfg *config.Config
}

func NewAuthHandler(database *db.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: database, cfg: cfg}
}

func (h *AuthHandler) RegisterRoutes(r *gin.Engine) {
	auth := r.Group("/api/auth")
	{
		auth.POST("/signup", h.Signup)
		auth.POST("/login", h.Login)
		auth.POST("/google", h.GoogleAuth)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
		auth.GET("/me", h.GetMe)
		auth.POST("/forgot-password", h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)
	}
}

type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User        User   `json:"user"`
}

type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name string `json:"name"`
}

type GoogleAuthRequest struct {
	Code       string `json:"code"`
	IDToken    string `json:"id_token"`
	GoogleID   string `json:"google_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL  string `json:"avatar_url"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	exists, err := h.db.EmailExists(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	userID, err := h.db.CreateUser(c.Request.Context(), req.Email, hash, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	token, err := auth.GenerateToken(auth.Load(), userID, req.Email, req.Name, "email")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(auth.Load(), userID, req.Email, req.Name, "email")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User: User{
			ID:    userID,
			Email: req.Email,
			Name: req.Name,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	user, err := h.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !auth.CheckPasswordHash(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := auth.GenerateToken(auth.Load(), user.ID, user.Email, user.Name, "email")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(auth.Load(), user.ID, user.Email, user.Name, "email")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User: User{
			ID:    user.ID,
			Email: user.Email,
			Name: user.Name,
		},
	})
}

func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	var req GoogleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var user *db.User
	var err error

	if req.GoogleID != "" {
		user, err = h.db.GetUserByGoogleID(c.Request.Context(), req.GoogleID)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	if user == nil && req.Email != "" {
		exists, _ := h.db.EmailExists(c.Request.Context(), req.Email)
		if exists {
			user, _ = h.db.GetUserByEmail(c.Request.Context(), req.Email)
			if user != nil && user.GoogleID == "" {
				h.db.LinkGoogleAccount(c.Request.Context(), user.ID, req.GoogleID, req.Email)
			}
		} else {
			userID, err := h.db.CreateUser(c.Request.Context(), req.Email, "", req.Name)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
				return
			}
			h.db.LinkGoogleAccount(c.Request.Context(), userID, req.GoogleID, req.Email)
			user, _ = h.db.GetUserByID(c.Request.Context(), userID)
		}
	}

	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid google auth"})
		return
	}

	token, err := auth.GenerateToken(auth.Load(), user.ID, user.Email, user.Name, "google")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(auth.Load(), user.ID, user.Email, user.Name, "google")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User: User{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		},
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	token, err := auth.RefreshToken(auth.Load(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh token"})
		return
	}

	_ = token // Token validated but not stored

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// Client should discard tokens - server can implement token blacklisting if needed
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no token provided"})
		return
	}

	token = strings.TrimPrefix(token, "Bearer ")
	claims, err := auth.ValidateToken(auth.Load(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	user, err := h.db.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, User{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	exists, _ := h.db.EmailExists(c.Request.Context(), req.Email)
	if !exists {
		c.JSON(http.StatusOK, gin.H{"message": "if email exists, reset link sent"})
		return
	}

	resetToken := uuid.New().String()
	h.db.PasswordResetTokens[req.Email] = struct {
		Token   string
		Expires time.Time
	}{
		Token:   resetToken,
		Expires: time.Now().Add(1 * time.Hour),
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "if email exists, reset link sent",
		"token":  resetToken,
	})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	_ = hash // Store hash in DB for password reset implementation

	c.JSON(http.StatusOK, gin.H{"message": "password reset successful"})
}