package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gw-currency-wallet/internal/api/middleware"
	"gw-currency-wallet/internal/service"
	"github.com/sirupsen/logrus"
)

// AuthHandler обработчик для аутентификации
type AuthHandler struct {
	service       *service.WalletService
	jwtMiddleware *middleware.JWTMiddleware
	logger        *logrus.Logger
}

// NewAuthHandler создает новый обработчик аутентификации
func NewAuthHandler(service *service.WalletService, jwtMiddleware *middleware.JWTMiddleware, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		service:       service,
		jwtMiddleware: jwtMiddleware,
		logger:        logger,
	}
}

// RegisterRequest запрос на регистрацию
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest запрос на авторизацию
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register регистрирует нового пользователя
// @Summary Register a new user
// @Description Register a new user with username, email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration data"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/v1/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Регистрируем пользователя
	if err := h.service.RegisterUser(c.Request.Context(), req.Username, req.Email, req.Password); err != nil {
		if err.Error() == "username already exists" || err.Error() == "email already exists" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Errorf("Failed to register user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

// Login авторизует пользователя
// @Summary Login user
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Аутентифицируем пользователя
	user, err := h.service.AuthenticateUser(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Генерируем JWT токен
	token, err := h.jwtMiddleware.GenerateToken(user.ID, user.Username, 24*3600*1000000000) // 24 hours
	if err != nil {
		h.logger.Errorf("Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
