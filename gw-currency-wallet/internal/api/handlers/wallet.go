package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gw-currency-wallet/internal/api/middleware"
	"gw-currency-wallet/internal/service"
	"github.com/sirupsen/logrus"
)

// WalletHandler обработчик для операций с кошельком
type WalletHandler struct {
	service *service.WalletService
	logger  *logrus.Logger
}

// NewWalletHandler создает новый обработчик кошелька
func NewWalletHandler(service *service.WalletService, logger *logrus.Logger) *WalletHandler {
	return &WalletHandler{
		service: service,
		logger:  logger,
	}
}

// DepositRequest запрос на пополнение
type DepositRequest struct {
	Amount   float64 `json:"amount" binding:"required,gt=0"`
	Currency string  `json:"currency" binding:"required,oneof=USD EUR RUB"`
}

// WithdrawRequest запрос на вывод
type WithdrawRequest struct {
	Amount   float64 `json:"amount" binding:"required,gt=0"`
	Currency string  `json:"currency" binding:"required,oneof=USD EUR RUB"`
}

// GetBalance возвращает баланс пользователя
// @Summary Get user balance
// @Description Get balance for all currencies
// @Tags wallet
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/balance [get]
func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	balances, err := h.service.GetUserBalances(c.Request.Context(), userID)
	if err != nil {
		h.logger.Errorf("Failed to get balances: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get balances"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": balances})
}

// Deposit пополняет счет пользователя
// @Summary Deposit funds
// @Description Add funds to user account
// @Tags wallet
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body DepositRequest true "Deposit data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/wallet/deposit [post]
func (h *WalletHandler) Deposit(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	newBalances, err := h.service.Deposit(c.Request.Context(), userID, req.Currency, req.Amount)
	if err != nil {
		h.logger.Errorf("Failed to deposit: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Account topped up successfully",
		"new_balance": newBalances,
	})
}

// Withdraw выводит средства со счета
// @Summary Withdraw funds
// @Description Withdraw funds from user account
// @Tags wallet
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body WithdrawRequest true "Withdrawal data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/wallet/withdraw [post]
func (h *WalletHandler) Withdraw(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	newBalances, err := h.service.Withdraw(c.Request.Context(), userID, req.Currency, req.Amount)
	if err != nil {
		h.logger.Errorf("Failed to withdraw: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Withdrawal successful",
		"new_balance": newBalances,
	})
}
